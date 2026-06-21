package scraper

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const maxPageBytes = 2 << 20

var tagStripper = regexp.MustCompile(`(?s)<[^>]*>`)

func fetchPage(ctx context.Context, client *Client, userAgent, rawURL string) (string, error) {
	resp, err := client.Get(ctx, userAgent, rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(rawURL, "reddit.com") && redditBlocked(resp.StatusCode) {
			return "", fmt.Errorf("reddit blocked (%d)", resp.StatusCode)
		}
		return "", fmt.Errorf("status %d for %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func stripHTML(raw string) string {
	raw = tagStripper.ReplaceAllString(raw, " ")
	raw = html.UnescapeString(raw)
	return strings.TrimSpace(strings.Join(strings.Fields(raw), " "))
}

func extractMDBlocks(page string) []string {
	re := regexp.MustCompile(`(?s)<div class="md">(.*?)</div>`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(page, -1) {
		text := stripHTML(m[1])
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}
