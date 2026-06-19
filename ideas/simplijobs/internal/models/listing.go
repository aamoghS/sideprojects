package models

import (
	"fmt"
	"strings"
	"time"
)

// Listing represents a single job posting from the SimplifyJobs GitHub repo.
type Listing struct {
	ID          string   `json:"id"`
	CompanyName string   `json:"company_name"`
	Title       string   `json:"title"`
	Locations   []string `json:"locations"`
	URL         string   `json:"url"`
	DateUpdated int64    `json:"date_updated"`
	IsVisible   bool     `json:"is_visible"`
	Source      string   `json:"source"`
	CompanyURL  string   `json:"company_url"`
}

// TimeAgo returns a human-readable relative time string for when the listing was updated.
func (l Listing) TimeAgo() string {
	diff := time.Now().Unix() - l.DateUpdated

	switch {
	case diff < 0:
		return "just now"
	case diff < 60:
		return "just now"
	case diff < 3600:
		m := diff / 60
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case diff < 86400:
		h := diff / 3600
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case diff < 604800:
		d := diff / 86400
		if d == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", d)
	case diff < 2592000:
		w := diff / 604800
		if w == 1 {
			return "1w ago"
		}
		return fmt.Sprintf("%dw ago", w)
	default:
		return time.Unix(l.DateUpdated, 0).Format("Jan 2")
	}
}

// LocationString returns a formatted string of the listing's locations.
func (l Listing) LocationString() string {
	if len(l.Locations) == 0 {
		return "—"
	}
	if len(l.Locations) == 1 {
		return l.Locations[0]
	}
	if len(l.Locations) <= 2 {
		return strings.Join(l.Locations, " · ")
	}
	return fmt.Sprintf("%s +%d more", l.Locations[0], len(l.Locations)-1)
}

// PostedDate returns the date_updated as a formatted date string.
func (l Listing) PostedDate() string {
	return time.Unix(l.DateUpdated, 0).Format("Jan 2, 2006")
}
