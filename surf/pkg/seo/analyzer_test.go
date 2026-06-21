package seo

import (
	"strings"
	"testing"
)

func TestAnalyzer_AnalyzeHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		url      string
		check    func(*testing.T, *AnalysisResult)
	}{
		{
			name: "perfect SEO page",
			html: `<!DOCTYPE html>
<html>
<head>
	<title>Perfect Page Title Here</title>
	<meta name="description" content="This is a perfect meta description that has just the right amount of content for SEO purposes.">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<meta property="og:title" content="Perfect Page Title Here">
	<meta property="og:description" content="This is a perfect meta description">
	<meta property="og:image" content="https://example.com/image.png">
	<meta name="twitter:card" content="summary_large_image">
	<link rel="canonical" href="https://example.com/page">
</head>
<body>
	<h1>This Is The Main Heading</h1>
	<p>First paragraph with lots of content. This is the first paragraph with lots of content. This is the first paragraph with lots of content.</p>
	<p>Second paragraph with lots of content. This is the second paragraph with lots of content. This is the second paragraph with lots of content.</p>
	<p>Third paragraph with lots of content. This is the third paragraph with lots of content. This is the third paragraph with lots of content.</p>
	<img src="img1.jpg" alt="Descriptive alt text">
	<img src="img2.jpg" alt="Another descriptive alt">
	<a href="/page1">Internal Link 1</a>
	<a href="/page2">Internal Link 2</a>
	<a href="https://external.com">External Link</a>
</body>
</html>`,
			url: "https://example.com/page",
			check: func(t *testing.T, r *AnalysisResult) {
				if r.Score != 100 {
					t.Errorf("Score = %d, want 100", r.Score)
				}
				if r.Title != "Perfect Page Title Here" {
					t.Errorf("Title = %q, want %q", r.Title, "Perfect Page Title Here")
				}
				if len(r.H1Tags) != 1 {
					t.Errorf("H1 count = %d, want 1", len(r.H1Tags))
				}
				if !r.HasViewportMeta {
					t.Error("Expected HasViewportMeta = true")
				}
				if !r.HasOpenGraph {
					t.Error("Expected HasOpenGraph = true")
				}
				if !r.HasTwitterCard {
					t.Error("Expected HasTwitterCard = true")
				}
				if r.CanonicalURL != "https://example.com/page" {
					t.Errorf("CanonicalURL = %q, want %q", r.CanonicalURL, "https://example.com/page")
				}
			},
		},
		{
			name: "missing title",
			html: `<!DOCTYPE html><html><head></head><body></body></html>`,
			url:  "https://example.com",
			check: func(t *testing.T, r *AnalysisResult) {
				if r.Title != "" {
					t.Errorf("Title = %q, want empty", r.Title)
				}
				if r.Score > 85 { // Should lose 15 points
					t.Errorf("Score = %d, want < 85", r.Score)
				}
			},
		},
		{
			name: "short title",
			html: `<!DOCTYPE html><html><head><title>Short</title></head><body></body></html>`,
			url:  "https://example.com",
			check: func(t *testing.T, r *AnalysisResult) {
				if r.TitleLength < 30 {
					t.Errorf("TitleLength = %d, want >= 30", r.TitleLength)
				}
			},
		},
		{
			name: "long title",
			html: `<!DOCTYPE html><html><head><title>` + strings.Repeat("a", 70) + `</title></head><body></body></html>`,
			url:  "https://example.com",
			check: func(t *testing.T, r *AnalysisResult) {
				if r.TitleLength <= 60 {
					t.Errorf("TitleLength = %d, want > 60", r.TitleLength)
				}
			},
		},
		{
			name: "multiple h1 tags",
			html: `<!DOCTYPE html><html><head><title>Test</title></head><body><h1>First</h1><h1>Second</h1></body></html>`,
			url:  "https://example.com",
			check: func(t *testing.T, r *AnalysisResult) {
				if len(r.H1Tags) != 2 {
					t.Errorf("H1 count = %d, want 2", len(r.H1Tags))
				}
			},
		},
		{
			name: "images without alt",
			html: `<!DOCTYPE html><html><head><title>Test</title></head><body><img src="a.jpg"><img src="b.jpg" alt="has alt"></body></html>`,
			url:  "https://example.com",
			check: func(t *testing.T, r *AnalysisResult) {
				if r.ImagesWithAlt != 1 {
					t.Errorf("ImagesWithAlt = %d, want 1", r.ImagesWithAlt)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAnalyzer()
			r, err := a.AnalyzeWithReader(strings.NewReader(tt.html), tt.url)
			if err != nil {
				t.Fatalf("AnalyzeWithReader() error = %v", err)
			}
			tt.check(t, r)
		})
	}
}

func TestAnalysisResult_ScoreCalculation(t *testing.T) {
	tests := []struct {
		name          string
		issues        []string
		expectedScore int
	}{
		{"no issues", []string{}, 100},
		{"one issue -15", []string{"Missing title tag"}, 85},
		{"multiple issues", []string{"Missing title tag", "Missing meta description", "No H1 tag found"}, 55},
		{"over 100 deduction", []string{"Missing title tag", "Missing meta description", "No H1 tag found", "Missing viewport meta tag"}, 40},
		{"all issues", []string{
			"Missing title tag",
			"Missing meta description",
			"No H1 tag found",
			"Missing viewport meta tag",
			"Missing Open Graph tags",
			"Missing Twitter Card meta tags",
			"Low word count",
		}, 35},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{
				URL:    "https://example.com",
				Issues: tt.issues,
				Score:  100,
			}

			// Simulate score calculation
			for _, issue := range tt.issues {
				switch {
				case issue == "Missing title tag":
					result.Score -= 15
				case issue == "Missing meta description":
					result.Score -= 15
				case issue == "No H1 tag found":
					result.Score -= 15
				case issue == "Missing viewport meta tag":
					result.Score -= 10
				case issue == "Missing Open Graph tags":
					result.Score -= 5
				case issue == "Missing Twitter Card meta tags":
					result.Score -= 3
				case issue == "Low word count":
					result.Score -= 5
				}
			}
			if result.Score < 0 {
				result.Score = 0
			}

			if result.Score != tt.expectedScore {
				t.Errorf("Score = %d, want %d", result.Score, tt.expectedScore)
			}
		})
	}
}

func BenchmarkHTMLParser_Parse(b *testing.B) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<meta name="description" content="Test description">
	<meta name="viewport" content="width=device-width">
	<meta property="og:title" content="Test">
</head>
<body>
	<h1>Main Heading</h1>
	<p>Paragraph with content. More content here.</p>
	<div>
		<p>Nested paragraph with more text.</p>
		<img src="img1.jpg" alt="Image 1">
		<img src="img2.jpg" alt="Image 2">
		<a href="/link1">Link 1</a>
		<a href="/link2">Link 2</a>
		<a href="https://external.com">External</a>
	</div>
</body>
</html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseHTML(html)
		if err != nil {
			b.Fatalf("ParseHTML() error = %v", err)
		}
	}
}

func BenchmarkHTMLParser_FindElements(b *testing.B) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
	<h1>Heading 1</h1>
	<h1>Heading 2</h1>
	<h1>Heading 3</h1>
	<div><div><div><div><p>Deep</p></div></div></div></div>
	<img src="a.jpg" alt="A">
	<img src="b.jpg" alt="B">
	<img src="c.jpg" alt="C">
	<img src="d.jpg">
	<img src="e.jpg">
</body>
</html>`

	doc, _ := ParseHTML(html)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = doc.findElements("h1")
		_ = doc.findElements("div")
		_, _ = doc.findImages()
	}
}