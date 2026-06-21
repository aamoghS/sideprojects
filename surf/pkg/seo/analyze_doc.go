package seo

import (
	"fmt"
	"strings"
)

func analyzeDocument(doc *Node, pageURL string) *AnalysisResult {
	result := &AnalysisResult{
		URL:    pageURL,
		Issues: []string{},
		H1Tags: []string{},
		Score:  100,
	}

	titleNode := findFirstTag(doc, "title")
	title := ""
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

	result.HasViewportMeta = doc.findMetaByName("viewport") != ""
	if !result.HasViewportMeta {
		result.Issues = append(result.Issues, "Missing viewport meta tag (important for mobile)")
		result.Score -= 10
	}

	result.HasOpenGraph = doc.findMetaByProperty("og:")
	if !result.HasOpenGraph {
		result.Issues = append(result.Issues, "Missing Open Graph tags")
		result.Score -= 5
	}

	result.HasTwitterCard = doc.findMetaByName("twitter:card") != ""
	if !result.HasTwitterCard {
		result.Issues = append(result.Issues, "Missing Twitter Card meta tags")
		result.Score -= 3
	}

	result.CanonicalURL = doc.findLinkByRel("canonical")

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

	result.ImageCount, result.ImagesWithAlt = doc.findImages()
	if result.ImageCount > 0 && result.ImagesWithAlt < result.ImageCount {
		missing := result.ImageCount - result.ImagesWithAlt
		result.Issues = append(result.Issues, fmt.Sprintf("%d images missing alt text", missing))
		result.Score -= 5
	}

	host := pageURL
	if idx := strings.Index(pageURL, "://"); idx > 0 {
		host = pageURL[idx+3:]
	}
	if idx := strings.Index(host, "/"); idx > 0 {
		host = host[:idx]
	}
	result.InternalLinks, result.ExternalLinks = doc.findLinks(host)

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
	result.Fixes = BuildFixes(result)
	return result
}
