package scraper

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	wikiParaRE   = regexp.MustCompile(`(?s)<p>(.*?)</p>`)
	wikiSearchRE = regexp.MustCompile(`(?i)class="mw-search-result-heading"[^>]*>\s*<a href="(/wiki/[^"]+)"`)
)

func ScrapeWikiPlot(ctx context.Context, client *Client, userAgent, title string, year int, wikiOverride string) string {
	if ctx.Err() != nil {
		return ""
	}

	wikiCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	candidates := []string{}
	if wikiOverride != "" {
		candidates = append(candidates, wikiOverride)
	}
	if year > 0 {
		candidates = append(candidates, fmt.Sprintf("%s (%d film)", title, year))
	}
	candidates = append(candidates, title+" (film)", title)

	seen := make(map[string]bool)
	for _, page := range candidates {
		if page == "" || seen[page] {
			continue
		}
		seen[page] = true
		if plot := fetchWikiPagePlot(wikiCtx, client, userAgent, page, year); plot != "" {
			return plot
		}
		if wikiCtx.Err() != nil {
			return ""
		}
	}

	if resolved := searchWikiPage(wikiCtx, client, userAgent, title, year); resolved != "" {
		return fetchWikiPagePlot(wikiCtx, client, userAgent, resolved, year)
	}
	return ""
}

func fetchWikiPagePlot(ctx context.Context, client *Client, userAgent, pageTitle string, year int) string {
	slug := strings.ReplaceAll(pageTitle, " ", "_")
	pageURL := "https://en.wikipedia.org/wiki/" + url.PathEscape(slug)

	page, err := fetchPage(ctx, client, userAgent, pageURL)
	if err != nil {
		return ""
	}

	for _, m := range wikiParaRE.FindAllStringSubmatch(page, -1) {
		text := stripHTML(m[1])
		if text == "" || strings.HasPrefix(text, "Coordinates:") {
			continue
		}
		if !looksLikeFilmPlot(text, year) {
			continue
		}
		if len(text) > 280 {
			text = text[:277] + "..."
		}
		return text
	}
	return ""
}

func searchWikiPage(ctx context.Context, client *Client, userAgent, title string, year int) string {
	terms := []string{}
	if year > 0 {
		terms = append(terms, fmt.Sprintf("%s (%d film)", title, year))
	}
	terms = append(terms, title+" film", title)

	for _, term := range terms {
		if ctx.Err() != nil {
			return ""
		}
		searchURL := "https://en.wikipedia.org/w/index.php?search=" + url.QueryEscape(term)
		page, err := fetchPage(ctx, client, userAgent, searchURL)
		if err != nil {
			continue
		}

		m := wikiSearchRE.FindStringSubmatch(page)
		if len(m) < 2 {
			continue
		}
		pageTitle := strings.TrimPrefix(m[1], "/wiki/")
		pageTitle, _ = url.PathUnescape(pageTitle)
		pageTitle = strings.ReplaceAll(pageTitle, "_", " ")
		if strings.Contains(strings.ToLower(pageTitle), "disambiguation") {
			continue
		}
		if plot := fetchWikiPagePlot(ctx, client, userAgent, pageTitle, year); plot != "" {
			return pageTitle
		}
	}
	return ""
}

func looksLikeFilmPlot(extract string, year int) bool {
	lower := strings.ToLower(extract)
	if strings.Contains(lower, "disambiguation") || strings.Contains(lower, "may refer to") {
		return false
	}
	filmCue := strings.Contains(lower, "film") ||
		strings.Contains(lower, "movie") ||
		strings.Contains(lower, "directed by") ||
		strings.Contains(lower, "animated")
	if !filmCue {
		return false
	}
	if year > 0 && !strings.Contains(extract, fmt.Sprintf("%d", year)) {
		return false
	}
	return true
}
