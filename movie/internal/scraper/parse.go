package scraper

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	titleWithYear = regexp.MustCompile(`(?i)^[\d.\-*\s]*\*{0,2}([A-Za-z0-9][A-Za-z0-9\s':,&.-]{1,60}?)\*{0,2}\s*\((\d{4})\)`)
	boldTitle     = regexp.MustCompile(`\*\*([^*]{2,60})\*\*`)
	yearInTitle   = regexp.MustCompile(`\((\d{4})\)`)
	skipPrefixes  = []string{
		"i ", "if ", "the ", "this ", "that ", "sounds ", "comedy:", "drama:",
		"horror:", "also ", "try ", "watch ", "check ", "you ", "my ", "for ",
		"have ", "here", "not ", "yes ", "no ", "lol", "imo", "edit:",
	}
)

func parseMoviesFromComment(body string) []Movie {
	var movies []Movie
	seen := make(map[string]bool)

	add := func(title string, year int) {
		title = strings.TrimSpace(strings.Trim(title, `"'`))
		if title == "" || !looksLikeMovieTitle(title) {
			return
		}
		key := movieKey(title, year)
		if seen[key] {
			return
		}
		seen[key] = true
		movies = append(movies, Movie{Title: title, Year: year})
	}

	for _, match := range titleWithYear.FindAllStringSubmatch(body, -1) {
		year := 0
		fmt.Sscanf(match[2], "%d", &year)
		add(match[1], year)
	}

	for _, match := range boldTitle.FindAllStringSubmatch(body, -1) {
		add(match[1], 0)
	}

	firstLine := strings.TrimSpace(strings.SplitN(body, "\n", 2)[0])
	firstLine = strings.TrimLeft(firstLine, "0123456789.-* ")
	if titleWithYear.MatchString(firstLine) || boldTitle.MatchString(firstLine) {
		return movies
	}

	if idx := strings.Index(firstLine, " - "); idx > 2 {
		add(firstLine[:idx], 0)
	} else if idx := strings.Index(firstLine, ", "); idx > 2 && idx < 50 {
		add(firstLine[:idx], 0)
	} else if looksLikeMovieTitle(firstLine) {
		if m := yearInTitle.FindStringSubmatch(firstLine); len(m) > 1 {
			year := 0
			fmt.Sscanf(m[1], "%d", &year)
			add(strings.TrimSpace(yearInTitle.ReplaceAllString(firstLine, "")), year)
		} else {
			add(firstLine, 0)
		}
	}
	return movies
}

func looksLikeMovieTitle(title string) bool {
	lower := strings.ToLower(strings.TrimSpace(title))
	if len(lower) < 3 || len(lower) > 70 {
		return false
	}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}
	if strings.Contains(lower, "http") || strings.Contains(lower, "reddit") {
		return false
	}
	if len(strings.Fields(lower)) > 12 {
		return false
	}
	if titleWithYear.FindStringSubmatch(title) == nil && strings.Contains(lower, " is ") {
		return false
	}
	return true
}
