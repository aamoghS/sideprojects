package filter

import (
	"sort"
	"strings"

	"github.com/bootcamp/simplijobs/internal/models"
)

// MatchesCompany returns true if the listing's company name contains the query (case-insensitive).
func MatchesCompany(l models.Listing, query string) bool {
	return strings.Contains(strings.ToLower(l.CompanyName), strings.ToLower(query))
}

// MatchesLocation returns true if any of the listing's locations contain the query (case-insensitive).
func MatchesLocation(l models.Listing, query string) bool {
	q := strings.ToLower(query)
	for _, loc := range l.Locations {
		if strings.Contains(strings.ToLower(loc), q) {
			return true
		}
	}
	return false
}

// SortByDate sorts listings by date_updated in descending order (newest first).
func SortByDate(listings []models.Listing) {
	sort.Slice(listings, func(i, j int) bool {
		return listings[i].DateUpdated > listings[j].DateUpdated
	})
}

// Apply runs all the provided filter functions against a slice of listings,
// returning only those that pass every filter.
func Apply(listings []models.Listing, filters ...func(models.Listing) bool) []models.Listing {
	result := make([]models.Listing, 0, len(listings))
	for _, l := range listings {
		pass := true
		for _, fn := range filters {
			if !fn(l) {
				pass = false
				break
			}
		}
		if pass {
			result = append(result, l)
		}
	}
	return result
}

// VisibleOnly returns a filter function that keeps only visible listings.
func VisibleOnly() func(models.Listing) bool {
	return func(l models.Listing) bool {
		return l.IsVisible
	}
}

// NewerThan returns a filter function that keeps only listings updated after the given timestamp.
func NewerThan(timestamp int64) func(models.Listing) bool {
	return func(l models.Listing) bool {
		return l.DateUpdated > timestamp
	}
}

// CompanyContains returns a filter function that matches listings by company name.
func CompanyContains(query string) func(models.Listing) bool {
	return func(l models.Listing) bool {
		return MatchesCompany(l, query)
	}
}

// LocationContains returns a filter function that matches listings by location.
func LocationContains(query string) func(models.Listing) bool {
	return func(l models.Listing) bool {
		return MatchesLocation(l, query)
	}
}
