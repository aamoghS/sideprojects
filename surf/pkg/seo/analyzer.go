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
		client: &http.Client{},
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

	result := &AnalysisResult{
		URL:    url,
		Issues: []string{},
		H1Tags: []string{},
		Score:  100,
	}

	// Analyze title
	titleNode := findFirstTag(doc, "title")
	var title string
	if titleNode != nil {
		title = strings.TrimSpace(titleNode.textContent())
	}
	result.Title = title
	result.TitleLength = len(title)
	if result.TitleLength == 0 {
		result.Issues = append(result.Issues, "Missing title tag")
		result.Score -= 15
	} else if result.TitleLength < 30 {
		result.Issues = append(result.Issues, "Title too short (less than 30 characters)")
		result.Score -= 10
	} else if result.TitleLength > 60 {
		result.Issues = append(result.Issues, "Title too long (more than 60 characters)")
		result.Score -= 10
	}

	// Analyze meta description
	desc := doc.findMetaByName("description")
	result.MetaDescription = desc
	result.DescriptionLength = len(desc)
	if result.DescriptionLength == 0 {
		result.Issues = append(result.Issues, "Missing meta description")
		result.Score -= 15
	} else if result.DescriptionLength < 120 {
		result.Issues = append(result.Issues, "Meta description too short (less than 120 characters)")
		result.Score -= 5
	} else if result.DescriptionLength > 160 {
		result.Issues = append(result.Issues, "Meta description too long (more than 160 characters)")
		result.Score -= 5
	}

	// Check viewport meta
	result.HasViewportMeta = doc.findMetaByName("viewport") != ""
	if !result.HasViewportMeta {
		result.Issues = append(result.Issues, "Missing viewport meta tag (important for mobile)")
		result.Score -= 10
	}

	// Check Open Graph
	result.HasOpenGraph = doc.findMetaByProperty("og:")
	if !result.HasOpenGraph {
		result.Issues = append(result.Issues, "Missing Open Graph tags")
		result.Score -= 5
	}

	// Check Twitter Card
	result.HasTwitterCard = doc.findMetaByName("twitter:card") != ""
	if !result.HasTwitterCard {
		result.Issues = append(result.Issues, "Missing Twitter Card meta tags")
		result.Score -= 3
	}

	// Check canonical URL
	result.CanonicalURL = doc.findLinkByRel("canonical")

	// Analyze H1 tags
	h1Elements := doc.findElements("h1")
	for _, h1 := range h1Elements {
		text := strings.TrimSpace(h1.textContent())
		if text != "" {
			result.H1Tags = append(result.H1Tags, text)
		}
	}
	if len(result.H1Tags) == 0 {
		result.Issues = append(result.Issues, "No H1 tag found")
		result.Score -= 15
	} else if len(result.H1Tags) > 1 {
		result.Issues = append(result.Issues, fmt.Sprintf("Multiple H1 tags found (%d)", len(result.H1Tags)))
		result.Score -= 10
	}

	// Analyze images
	result.ImageCount, result.ImagesWithAlt = doc.findImages()
	if result.ImageCount > 0 && result.ImagesWithAlt < result.ImageCount {
		missing := result.ImageCount - result.ImagesWithAlt
		result.Issues = append(result.Issues, fmt.Sprintf("%d images missing alt text", missing))
		result.Score -= 5
	}

	// Extract hostname for link analysis
	host := url
	if idx := strings.Index(url, "://"); idx > 0 {
		host = url[idx+3:]
	}
	if idx := strings.Index(host, "/"); idx > 0 {
		host = host[:idx]
	}

	// Analyze links
	result.InternalLinks, result.ExternalLinks = doc.findLinks(host)

	// Count words in body
	body := findFirstTag(doc, "body")
	if body != nil {
		result.WordCount = body.wordCount()
		if result.WordCount < 300 {
			result.Issues = append(result.Issues, fmt.Sprintf("Low word count (%d words, recommended: 300+)", result.WordCount))
			result.Score -= 5
		}
	}

	if result.Score < 0 {
		result.Score = 0
	}

	return result, nil
}

// AnalyzeWithReader performs SEO analysis on HTML content directly
func (a *Analyzer) AnalyzeWithReader(r io.Reader, url string) (*AnalysisResult, error) {
	doc, err := NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	result := &AnalysisResult{
		URL:    url,
		Issues: []string{},
		H1Tags: []string{},
		Score:  100,
	}

	// Analyze title
	titleNode := findFirstTag(doc, "title")
	var title string
	if titleNode != nil {
		title = strings.TrimSpace(titleNode.textContent())
	}
	result.Title = title
	result.TitleLength = len(title)
	if result.TitleLength == 0 {
		result.Issues = append(result.Issues, "Missing title tag")
		result.Score -= 15
	} else if result.TitleLength < 30 {
		result.Issues = append(result.Issues, "Title too short (less than 30 characters)")
		result.Score -= 10
	} else if result.TitleLength > 60 {
		result.Issues = append(result.Issues, "Title too long (more than 60 characters)")
		result.Score -= 10
	}

	// Analyze meta description
	desc := doc.findMetaByName("description")
	result.MetaDescription = desc
	result.DescriptionLength = len(desc)
	if result.DescriptionLength == 0 {
		result.Issues = append(result.Issues, "Missing meta description")
		result.Score -= 15
	} else if result.DescriptionLength < 120 {
		result.Issues = append(result.Issues, "Meta description too short (less than 120 characters)")
		result.Score -= 5
	} else if result.DescriptionLength > 160 {
		result.Issues = append(result.Issues, "Meta description too long (more than 160 characters)")
		result.Score -= 5
	}

	// Check viewport meta
	result.HasViewportMeta = doc.findMetaByName("viewport") != ""
	if !result.HasViewportMeta {
		result.Issues = append(result.Issues, "Missing viewport meta tag (important for mobile)")
		result.Score -= 10
	}

	// Check Open Graph
	result.HasOpenGraph = doc.findMetaByProperty("og:")
	if !result.HasOpenGraph {
		result.Issues = append(result.Issues, "Missing Open Graph tags")
		result.Score -= 5
	}

	// Check Twitter Card
	result.HasTwitterCard = doc.findMetaByName("twitter:card") != ""
	if !result.HasTwitterCard {
		result.Issues = append(result.Issues, "Missing Twitter Card meta tags")
		result.Score -= 3
	}

	// Check canonical URL
	result.CanonicalURL = doc.findLinkByRel("canonical")

	// Analyze H1 tags
	h1Elements := doc.findElements("h1")
	for _, h1 := range h1Elements {
		text := strings.TrimSpace(h1.textContent())
		if text != "" {
			result.H1Tags = append(result.H1Tags, text)
		}
	}
	if len(result.H1Tags) == 0 {
		result.Issues = append(result.Issues, "No H1 tag found")
		result.Score -= 15
	} else if len(result.H1Tags) > 1 {
		result.Issues = append(result.Issues, fmt.Sprintf("Multiple H1 tags found (%d)", len(result.H1Tags)))
		result.Score -= 10
	}

	// Analyze images
	result.ImageCount, result.ImagesWithAlt = doc.findImages()
	if result.ImageCount > 0 && result.ImagesWithAlt < result.ImageCount {
		missing := result.ImageCount - result.ImagesWithAlt
		result.Issues = append(result.Issues, fmt.Sprintf("%d images missing alt text", missing))
		result.Score -= 5
	}

	// Count words in body
	body := findFirstTag(doc, "body")
	if body != nil {
		result.WordCount = body.wordCount()
		if result.WordCount < 300 {
			result.Issues = append(result.Issues, fmt.Sprintf("Low word count (%d words, recommended: 300+)", result.WordCount))
			result.Score -= 5
		}
	}

	if result.Score < 0 {
		result.Score = 0
	}

	return result, nil
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
}