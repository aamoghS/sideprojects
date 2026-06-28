package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aamoghS/sideprojects/ideas/simplijobs/internal/models"
)

// Source URLs for the SimplifyJobs GitHub repositories.
var sourceURLs = map[string]string{
	"internships": "https://raw.githubusercontent.com/SimplifyJobs/Summer2026-Internships/dev/.github/scripts/listings.json",
	"newgrad":     "https://raw.githubusercontent.com/SimplifyJobs/New-Grad-Positions/dev/.github/scripts/listings.json",
}

// SourceLabel returns a human-readable label for the given source.
func SourceLabel(source string) string {
	switch source {
	case "internships":
		return "Internships"
	case "newgrad":
		return "New Grad"
	default:
		return source
	}
}

// SourceEmoji returns an emoji icon for the given source.
func SourceEmoji(source string) string {
	switch source {
	case "internships":
		return "🎓"
	case "newgrad":
		return "💼"
	default:
		return "📋"
	}
}

// ValidSources returns the list of valid source identifiers.
func ValidSources() []string {
	return []string{"internships", "newgrad"}
}

// IsValidSource checks if the given source string is a recognized source.
func IsValidSource(source string) bool {
	_, ok := sourceURLs[source]
	return ok
}

// FetchListings downloads and parses the listings.json from GitHub for the given source.
func FetchListings(source string) ([]models.Listing, error) {
	url, ok := sourceURLs[source]
	if !ok {
		return nil, fmt.Errorf("unknown source: %q (valid: internships, newgrad)", source)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch listings from %s: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching %s listings", resp.StatusCode, source)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var listings []models.Listing
	if err := json.Unmarshal(body, &listings); err != nil {
		return nil, fmt.Errorf("failed to parse listings JSON: %w", err)
	}

	return listings, nil
}
