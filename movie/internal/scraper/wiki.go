package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
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
		if plot := fetchWikiSummary(wikiCtx, client, userAgent, page, year); plot != "" {
			return plot
		}
		if wikiCtx.Err() != nil {
			return ""
		}
	}

	if resolved := resolveWikiPage(wikiCtx, client, userAgent, title, year); resolved != "" {
		return fetchWikiSummary(wikiCtx, client, userAgent, resolved, year)
	}
	return ""
}

func fetchWikiSummary(ctx context.Context, client *Client, userAgent, pageTitle string, year int) string {
	summaryURL := fmt.Sprintf(
		"https://en.wikipedia.org/api/rest_v1/page/summary/%s",
		strings.ReplaceAll(pageTitle, " ", "_"),
	)

	resp, err := client.Get(ctx, userAgent, summaryURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var summary struct {
		Extract string `json:"extract"`
		Type    string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return ""
	}
	if summary.Type == "disambiguation" || !looksLikeFilmPlot(summary.Extract, year) {
		return ""
	}

	plot := summary.Extract
	if len(plot) > 280 {
		plot = plot[:277] + "..."
	}
	return plot
}

func resolveWikiPage(ctx context.Context, client *Client, userAgent, title string, year int) string {
	terms := []string{}
	if year > 0 {
		terms = append(terms, fmt.Sprintf("%s (%d film)", title, year))
	}
	terms = append(terms, title+" film", title)

	for _, term := range terms {
		if ctx.Err() != nil {
			return ""
		}
		wikiURL := fmt.Sprintf(
			"https://en.wikipedia.org/w/api.php?action=opensearch&search=%s&limit=3&namespace=0&format=json",
			url.QueryEscape(term),
		)
		resp, err := client.Get(ctx, userAgent, wikiURL)
		if err != nil {
			continue
		}

		var result []interface{}
		decodeErr := json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if decodeErr != nil || len(result) < 2 {
			continue
		}

		titles, ok := result[1].([]interface{})
		if !ok {
			continue
		}
		for _, t := range titles {
			page, ok := t.(string)
			if !ok || strings.Contains(strings.ToLower(page), "disambiguation") {
				continue
			}
			if plot := fetchWikiSummary(ctx, client, userAgent, page, year); plot != "" {
				return page
			}
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
