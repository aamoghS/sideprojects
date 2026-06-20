package seo

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SitemapUrl represents a URL entry in the sitemap
type SitemapUrl struct {
	Loc        string    `xml:"loc"`
	LastMod    time.Time `xml:"lastmod,omitempty"`
	ChangeFreq string    `xml:"changefreq,omitempty"`
	Priority   float64   `xml:"priority,omitempty"`
}

// Sitemap represents the sitemap XML structure
type Sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	Urls    []SitemapUrl `xml:"url"`
}

// SitemapIndexEntry represents an entry in sitemap index
type SitemapIndexEntry struct {
	Loc     string    `xml:"loc"`
	LastMod time.Time `xml:"lastmod,omitempty"`
}

// SitemapIndex represents a sitemap index file
type SitemapIndex struct {
	XMLName  xml.Name            `xml:"sitemapindex"`
	Xmlns    string              `xml:"xmlns,attr"`
	Sitemaps []SitemapIndexEntry `xml:"sitemap"`
}

// URLSetOption is a functional option for configuring sitemap URLs
type URLSetOption func(*SitemapUrl)

// WithChangeFreq sets the change frequency
func WithChangeFreq(freq string) URLSetOption {
	return func(u *SitemapUrl) {
		u.ChangeFreq = freq
	}
}

// WithPriority sets the priority (0.0 - 1.0)
func WithPriority(p float64) URLSetOption {
	return func(u *SitemapUrl) {
		u.Priority = p
	}
}

// WithLastMod sets the last modified date
func WithLastMod(t time.Time) URLSetOption {
	return func(u *SitemapUrl) {
		u.LastMod = t
	}
}

// NewSitemap creates a new sitemap
func NewSitemap(baseURL string, urls []string, opts ...URLSetOption) *Sitemap {
	sitemap := &Sitemap{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Urls:  make([]SitemapUrl, 0, len(urls)),
	}

	for _, u := range urls {
		sitemapUrl := SitemapUrl{
			Loc:      baseURL + u,
			Priority: 0.5,
		}
		for _, opt := range opts {
			opt(&sitemapUrl)
		}
		sitemap.Urls = append(sitemap.Urls, sitemapUrl)
	}

	return sitemap
}

// Generate creates a sitemap.xml file
func (s *Sitemap) Generate(w io.Writer) error {
	_, err := w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(s)
}

// GenerateIndex creates a sitemap index file for multiple sitemaps
func GenerateIndex(w io.Writer, sitemaps []string) error {
	index := SitemapIndex{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Sitemaps: make([]SitemapIndexEntry, len(sitemaps)),
	}

	for i, sm := range sitemaps {
		index.Sitemaps[i] = SitemapIndexEntry{
			Loc:     sm,
			LastMod: time.Now(),
		}
	}

	_, err := w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(index)
}

// findAllAnchors finds all anchor elements and collects their hrefs
func (n *Node) findAllAnchors() []string {
	var hrefs []string
	if n.Type == ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				hrefs = append(hrefs, attr.Val)
				break
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		hrefs = append(hrefs, c.findAllAnchors()...)
	}
	return hrefs
}

// SitemapGenerator crawls a site and generates a sitemap
type SitemapGenerator struct {
	baseURL  *url.URL
	client   *http.Client
	visited  map[string]bool
	urls     []string
	maxDepth int
}

// NewSitemapGenerator creates a new sitemap generator
func NewSitemapGenerator(base string, maxDepth int) (*SitemapGenerator, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}

	return &SitemapGenerator{
		baseURL:  u,
		client:   &http.Client{},
		visited:  make(map[string]bool),
		urls:     []string{},
		maxDepth: maxDepth,
	}, nil
}

// Crawl crawls the site and collects URLs
func (sg *SitemapGenerator) Crawl() error {
	return sg.crawl(sg.baseURL.String(), 0)
}

func (sg *SitemapGenerator) crawl(currentURL string, depth int) error {
	if depth > sg.maxDepth {
		return nil
	}

	if sg.visited[currentURL] {
		return nil
	}
	sg.visited[currentURL] = true
	sg.urls = append(sg.urls, currentURL)

	req, err := http.NewRequest("GET", currentURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "SitemapGenerator/1.0")

	resp, err := sg.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	hrefs := doc.findAllAnchors()
	for _, href := range hrefs {
		absURL := sg.resolveURL(href)
		if absURL == "" {
			continue
		}

		// Only follow internal links
		linkHost := strings.TrimPrefix(strings.TrimPrefix(absURL, "https://"), "http://")
		if idx := strings.Index(linkHost, "/"); idx > 0 {
			linkHost = linkHost[:idx]
		}
		if linkHost != sg.baseURL.Host {
			continue
		}

		sg.crawl(absURL, depth+1)
	}

	return nil
}

func (sg *SitemapGenerator) resolveURL(href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return sg.baseURL.Scheme + "://" + sg.baseURL.Host + href
	}
	return ""
}

// GetURLs returns the collected URLs
func (sg *SitemapGenerator) GetURLs() []string {
	return sg.urls
}

// GenerateSitemap creates a sitemap from crawled URLs
func (sg *SitemapGenerator) GenerateSitemap(w io.Writer) error {
	sitemap := &Sitemap{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Urls:  make([]SitemapUrl, 0, len(sg.urls)),
	}

	for _, u := range sg.urls {
		sitemap.Urls = append(sitemap.Urls, SitemapUrl{
			Loc:      u,
			Priority: 0.5,
		})
	}

	_, err := w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(sitemap)
}

// SimpleSitemapExample demonstrates basic sitemap generation
func SimpleSitemapExample() string {
	sitemap := NewSitemap("https://aamogh.vercel.app", []string{
		"/",
		"/about",
		"/projects",
		"/contact",
	}, WithPriority(1.0))

	var out strings.Builder
	sitemap.Generate(&out)
	return out.String()
}