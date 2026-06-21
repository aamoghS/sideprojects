package seo

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Analyzer checks SEO aspects of a webpage
type Analyzer struct {
	client *http.Client
}

// NewAnalyzer creates a new SEO analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		client: HTTPClientFromEnv(),
	}
}

// AnalysisResult holds the SEO analysis results
type AnalysisResult struct {
	URL               string
	Title             string
	TitleLength       int
	MetaDescription   string
	DescriptionLength int
	H1Tags            []string
	HasViewportMeta   bool
	HasOpenGraph      bool
	HasTwitterCard    bool
	CanonicalURL      string
	ImageCount        int
	ImagesWithAlt     int
	InternalLinks     int
	ExternalLinks     int
	WordCount         int
	Issues            []string
	Fixes             []Fix
	Score             int
}

// findFirstTag finds the first occurrence of a tag
func findFirstTag(n *Node, tagName string) *Node {
	tagName = strings.ToLower(tagName)
	if n.Type == ElementNode && n.Data == tagName {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findFirstTag(c, tagName); result != nil {
			return result
		}
	}
	return nil
}

// Analyze performs SEO analysis on a URL
func (a *Analyzer) Analyze(url string) (*AnalysisResult, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "SEO-Analyzer/1.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got status code %d", resp.StatusCode)
	}

	doc, err := NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return analyzeDocument(doc, url), nil
}

func (a *Analyzer) AnalyzeWithReader(r io.Reader, url string) (*AnalysisResult, error) {
	doc, err := NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return analyzeDocument(doc, url), nil
}

// PrintResult prints the analysis in a readable format
func PrintResult(r *AnalysisResult) {
	fmt.Printf("\n=== SEO Analysis for %s ===\n\n", r.URL)

	fmt.Printf("Score: %d/100\n\n", r.Score)

	fmt.Printf("Title: %s\n", r.Title)
	fmt.Printf("Title Length: %d chars\n\n", r.TitleLength)

	fmt.Printf("Meta Description: %s\n", r.MetaDescription)
	fmt.Printf("Description Length: %d chars\n\n", r.DescriptionLength)

	fmt.Printf("H1 Tags (%d):\n", len(r.H1Tags))
	for _, h1 := range r.H1Tags {
		fmt.Printf("  - %s\n", h1)
	}
	fmt.Println()

	fmt.Printf("Technical SEO:\n")
	fmt.Printf("  - Viewport Meta: %v\n", r.HasViewportMeta)
	fmt.Printf("  - Open Graph: %v\n", r.HasOpenGraph)
	fmt.Printf("  - Twitter Card: %v\n", r.HasTwitterCard)
	fmt.Printf("  - Canonical URL: %s\n\n", r.CanonicalURL)

	fmt.Printf("Content:\n")
	fmt.Printf("  - Images: %d (with alt: %d)\n", r.ImageCount, r.ImagesWithAlt)
	fmt.Printf("  - Internal Links: %d\n", r.InternalLinks)
	fmt.Printf("  - External Links: %d\n", r.ExternalLinks)
	fmt.Printf("  - Word Count: %d\n\n", r.WordCount)

	if len(r.Issues) > 0 {
		fmt.Printf("Issues Found (%d):\n", len(r.Issues))
		for _, issue := range r.Issues {
			fmt.Printf("  - %s\n", issue)
		}
	} else {
		fmt.Println("No issues found!")
	}

	if len(r.Fixes) > 0 {
		fmt.Printf("\nRecommended fixes (%d, +%d pts potential):\n", len(r.Fixes), TotalFixImpact(r.Fixes))
		for i, fix := range r.Fixes {
			fmt.Printf("  %d. [%s] %s\n", i+1, strings.ToUpper(fix.Priority), fix.Issue)
			fmt.Printf("     %s\n", fix.Action)
		}
	}
}