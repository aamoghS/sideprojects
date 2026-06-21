package seo

import (
	"fmt"
	"net/url"
	"strings"
)

type Fix struct {
	ID       string `json:"id"`
	Priority string `json:"priority"`
	Impact   int    `json:"impact"`
	Issue    string `json:"issue"`
	Action   string `json:"action"`
}

func BuildFixes(r *AnalysisResult) []Fix {
	var fixes []Fix
	for _, issue := range r.Issues {
		if f, ok := fixForIssue(issue); ok {
			fixes = append(fixes, f)
		}
	}
	if r.CanonicalURL == "" {
		fixes = append(fixes, Fix{
			ID: "canonical-missing", Priority: "high", Impact: 10,
			Issue:  "Missing canonical URL",
			Action: "Add <link rel=\"canonical\" href=\"PAGE_URL\"> pointing to the preferred URL for this page.",
		})
	} else if !canonicalMatches(r.URL, r.CanonicalURL) {
		fixes = append(fixes, Fix{
			ID: "canonical-mismatch", Priority: "high", Impact: 10,
			Issue:  fmt.Sprintf("Canonical %q does not match page URL", r.CanonicalURL),
			Action: "Set canonical to the URL you want indexed, or redirect this URL to the canonical target.",
		})
	}
	return fixes
}

func fixForIssue(issue string) (Fix, bool) {
	type rule struct {
		id       string
		priority string
		impact   int
		match    func(string) bool
		action   string
	}
	rules := []rule{
		{
			id: "title-missing", priority: "high", impact: 15,
			match: func(s string) bool { return s == "Missing title tag" },
			action: "Add a unique <title> (~50–60 chars) with primary keyword near the front.",
		},
		{
			id: "title-length", priority: "medium", impact: 10,
			match: func(s string) bool { return strings.HasPrefix(s, "Title too ") },
			action: "Rewrite title to ~50–60 characters: specific, readable, one primary keyword.",
		},
		{
			id: "meta-missing", priority: "high", impact: 15,
			match: func(s string) bool { return s == "Missing meta description" },
			action: "Add <meta name=\"description\" content=\"...\"> (~120–155 chars) that matches page intent.",
		},
		{
			id: "meta-length", priority: "medium", impact: 5,
			match: func(s string) bool { return strings.HasPrefix(s, "Meta description too ") },
			action: "Trim or expand meta description to ~120–155 characters with a clear value prop.",
		},
		{
			id: "h1-missing", priority: "high", impact: 15,
			match: func(s string) bool { return s == "No H1 tag found" },
			action: "Add exactly one H1 that states the page topic; match it closely to the title.",
		},
		{
			id: "h1-multiple", priority: "medium", impact: 10,
			match: func(s string) bool { return strings.HasPrefix(s, "Multiple H1 tags") },
			action: "Keep one H1 for the main topic; demote extras to H2/H3.",
		},
		{
			id: "viewport", priority: "high", impact: 10,
			match: func(s string) bool { return strings.Contains(s, "viewport") },
			action: "Add <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">.",
		},
		{
			id: "image-alt", priority: "medium", impact: 5,
			match: func(s string) bool { return strings.Contains(s, "missing alt text") },
			action: "Add descriptive alt text to every content image; leave decorative images alt=\"\".",
		},
		{
			id: "word-count", priority: "low", impact: 5,
			match: func(s string) bool { return strings.HasPrefix(s, "Low word count") },
			action: "Expand thin pages with useful, unique copy that answers the search intent.",
		},
		{
			id: "open-graph", priority: "low", impact: 5,
			match: func(s string) bool { return s == "Missing Open Graph tags" },
			action: "Add og:title, og:description, og:url, and og:image for share previews.",
		},
		{
			id: "twitter-card", priority: "low", impact: 3,
			match: func(s string) bool { return s == "Missing Twitter Card meta tags" },
			action: "Add twitter:card and twitter:title (and image if you use OG images).",
		},
	}
	for _, r := range rules {
		if r.match(issue) {
			return Fix{ID: r.id, Priority: r.priority, Impact: r.impact, Issue: issue, Action: r.action}, true
		}
	}
	return Fix{}, false
}

func canonicalMatches(pageURL, canonical string) bool {
	a, err := url.Parse(strings.TrimSpace(pageURL))
	if err != nil {
		return true
	}
	b, err := url.Parse(strings.TrimSpace(canonical))
	if err != nil {
		return true
	}
	a.Path = strings.TrimSuffix(a.Path, "/")
	b.Path = strings.TrimSuffix(b.Path, "/")
	if a.Path == "" {
		a.Path = "/"
	}
	if b.Path == "" {
		b.Path = "/"
	}
	return strings.EqualFold(a.Host, b.Host) && a.Path == b.Path && a.Scheme == b.Scheme
}

func TotalFixImpact(fixes []Fix) int {
	n := 0
	for _, f := range fixes {
		n += f.Impact
	}
	return n
}
