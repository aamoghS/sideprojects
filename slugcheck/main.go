package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

type urlEntry struct {
	Loc string `xml:"loc"`
}

type urlSet struct {
	URLs []urlEntry `xml:"url"`
}

type sitemapIndex struct {
	Sitemaps []urlEntry `xml:"sitemap"`
}

func main() {
	sitemapURL := flag.String("sitemap", "", "sitemap URL (default: site/sitemap.xml)")
	maxHops := flag.Int("max-hops", 5, "flag redirect chains longer than this")
	timeout := flag.Duration("timeout", 12*time.Second, "per-request timeout")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: slugcheck [flags] https://example.com\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	site, err := url.Parse(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	if site.Scheme == "" {
		site.Scheme = "https"
	}
	if *sitemapURL == "" {
		*sitemapURL = strings.TrimRight(site.String(), "/") + "/sitemap.xml"
	}

	client := &http.Client{
		Timeout: *timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	locURLs, err := collectSitemapURLs(client, *sitemapURL)
	if err != nil {
		log.Fatal(err)
	}
	if len(locURLs) == 0 {
		log.Fatal("no URLs in sitemap")
	}

	fmt.Printf("sitemap: %s (%d urls)\n\n", *sitemapURL, len(locURLs))

	type resolved struct {
		from  string
		final string
		hops  int
	}
	byFinal := make(map[string][]resolved)
	var chains []resolved

	for _, raw := range locURLs {
		final, hops, err := resolve(client, raw)
		if err != nil {
			log.Printf("skip %s: %v", raw, err)
			continue
		}
		r := resolved{from: raw, final: final, hops: hops}
		byFinal[final] = append(byFinal[final], r)
		if hops > *maxHops {
			chains = append(chains, r)
		}
	}

	collisions := 0
	for final, entries := range byFinal {
		if len(entries) < 2 {
			continue
		}
		collisions++
		fmt.Printf("collision -> %s\n", final)
		for _, e := range entries {
			fmt.Printf("  %s\n", e.from)
		}
		fmt.Println()
	}
	if collisions == 0 {
		fmt.Println("no slug collisions (distinct sitemap locs land on different finals)")
	}

	if len(chains) > 0 {
		fmt.Printf("\nredirect chains > %d hops:\n", *maxHops)
		for _, c := range chains {
			fmt.Printf("  %d hops  %s -> %s\n", c.hops, c.from, c.final)
		}
	} else {
		fmt.Printf("\nno redirect chains longer than %d hops\n", *maxHops)
	}

	slugDupes := findSlugDupes(locURLs)
	if len(slugDupes) > 0 {
		fmt.Println("\nsame trailing slug, different paths:")
		for slug, urls := range slugDupes {
			fmt.Printf("  /%s\n", slug)
			for _, u := range urls {
				fmt.Printf("    %s\n", u)
			}
		}
	}
}

func collectSitemapURLs(client *http.Client, sitemapURL string) ([]string, error) {
	body, err := fetch(client, sitemapURL)
	if err != nil {
		return nil, err
	}

	var idx sitemapIndex
	if xml.Unmarshal(body, &idx) == nil && len(idx.Sitemaps) > 0 {
		var all []string
		for _, sm := range idx.Sitemaps {
			part, err := collectSitemapURLs(client, sm.Loc)
			if err != nil {
				return nil, err
			}
			all = append(all, part...)
		}
		return all, nil
	}

	var set urlSet
	if err := xml.Unmarshal(body, &set); err != nil {
		return nil, fmt.Errorf("parse sitemap: %w", err)
	}
	out := make([]string, 0, len(set.URLs))
	for _, u := range set.URLs {
		if u.Loc != "" {
			out = append(out, u.Loc)
		}
	}
	return out, nil
}

func fetch(client *http.Client, raw string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "slugcheck/0.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s: %s", raw, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func resolve(client *http.Client, raw string) (string, int, error) {
	req, err := http.NewRequest(http.MethodHead, raw, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", "slugcheck/0.1")

	hops := 0
	cl := *client
	cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		hops = len(via)
		if hops >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}

	resp, err := cl.Do(req)
	if err != nil {
		// some hosts hate HEAD; try GET without reading much
		req, err = http.NewRequest(http.MethodGet, raw, nil)
		if err != nil {
			return "", 0, err
		}
		req.Header.Set("User-Agent", "slugcheck/0.1")
		hops = 0
		cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			hops = len(via)
			if hops >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		}
		resp, err = cl.Do(req)
		if err != nil {
			return "", 0, err
		}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.Request.URL.String(), hops, nil
}

func findSlugDupes(urls []string) map[string][]string {
	bySlug := make(map[string][]string)
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		slug := strings.Trim(path.Base(u.Path), "/")
		if slug == "" {
			continue
		}
		bySlug[slug] = append(bySlug[slug], raw)
	}
	out := make(map[string][]string)
	for slug, list := range bySlug {
		if len(list) > 1 {
			out[slug] = list
		}
	}
	return out
}
