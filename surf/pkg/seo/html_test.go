package seo

import (
	"strings"
	"testing"
)

func TestHTMLParser_ParseBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantText string
		wantErr  bool
	}{
		{
			name:     "simple text",
			input:    "Hello World",
			wantText: "Hello World",
			wantErr:  false,
		},
		{
			name:     "basic HTML",
			input:    "<html><body>Test</body></html>",
			wantText: "Test",
			wantErr:  false,
		},
		{
			name:     "nested elements",
			input:    "<div><span><p>Nested</p></span></div>",
			wantText: "Nested",
			wantErr:  false,
		},
		{
			name:     "multiple text nodes",
			input:    "Hello <p>world</p> how <strong>are</strong> you",
			wantText: "Hello world how are you",
			wantErr:  false,
		},
		{
			name:     "self-closing tags",
			input:    "Text<br/>more<br>end",
			wantText: "Textmoreend",
			wantErr:  false,
		},
		{
			name:     "script and style ignored",
			input:    "Visible<script>document.write('hidden')</script><style>.x{display:none}</style>Text",
			wantText: "VisibleText",
			wantErr:  false,
		},
		{
			name:     "comments ignored",
			input:    "Before<!-- comment -->After",
			wantText: "BeforeAfter",
			wantErr:  false,
		},
		{
			name:     "doctype ignored",
			input:    "<!DOCTYPE html><html><body>Content</body></html>",
			wantText: "Content",
			wantErr:  false,
		},
		{
			name:     "attributes parsed",
			input:    `<a href="http://example.com" title="Link">Link</a>`,
			wantText: "Link",
			wantErr:  false,
		},
		{
			name:     "empty input",
			input:    "",
			wantText: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseHTML(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				got := strings.TrimSpace(doc.textContent())
				if got != tt.wantText {
					t.Errorf("textContent() = %q, want %q", got, tt.wantText)
				}
			}
		})
	}
}

func TestHTMLParser_FindElements(t *testing.T) {
	input := `<html>
		<body>
			<h1>Title</h1>
			<div>
				<h1>Subtitle</h1>
			</div>
			<h1>Another</h1>
		</body>
	</html>`

	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	h1s := doc.findElements("h1")
	if len(h1s) != 3 {
		t.Errorf("findElements(h1) = %d, want 3", len(h1s))
	}

	divs := doc.findElements("div")
	if len(divs) != 1 {
		t.Errorf("findElements(div) = %d, want 1", len(divs))
	}
}

func TestHTMLParser_FindMetaByName(t *testing.T) {
	input := `<html>
		<head>
			<meta name="description" content="Test description">
			<meta name="keywords" content="go,golang">
			<meta name="viewport" content="width=device-width">
			<title>Test</title>
		</head>
	</html>`

	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	tests := []struct {
		name   string
		want   string
	}{
		{"description", "Test description"},
		{"keywords", "go,golang"},
		{"viewport", "width=device-width"},
		{"author", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := doc.findMetaByName(tt.name)
			if got != tt.want {
				t.Errorf("findMetaByName(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestHTMLParser_FindMetaByProperty(t *testing.T) {
	input := `<html>
		<head>
			<meta property="og:title" content="Test">
			<meta property="og:image" content="/img.png">
			<meta property="twitter:card" content="summary">
		</head>
	</html>`

	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	tests := []struct {
		prefix string
		want   bool
	}{
		{"og:", true},
		{"og:image", true},
		{"twitter:", true},
		{"fb:", false},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			got := doc.findMetaByProperty(tt.prefix)
			if got != tt.want {
				t.Errorf("findMetaByProperty(%q) = %v, want %v", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestHTMLParser_FindLinkByRel(t *testing.T) {
	input := `<html>
		<head>
			<link rel="canonical" href="https://example.com/page">
			<link rel="alternate" hreflang="es" href="https://example.com/es">
			<link rel="stylesheet" href="style.css">
		</head>
	</html>`

	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	tests := []struct {
		rel  string
		want string
	}{
		{"canonical", "https://example.com/page"},
		{"alternate", "https://example.com/es"},
		{"stylesheet", "style.css"},
		{"icon", ""},
	}

	for _, tt := range tests {
		t.Run(tt.rel, func(t *testing.T) {
			got := doc.findLinkByRel(tt.rel)
			if got != tt.want {
				t.Errorf("findLinkByRel(%q) = %q, want %q", tt.rel, got, tt.want)
			}
		})
	}
}

func TestHTMLParser_Images(t *testing.T) {
	input := `<html>
		<body>
			<img src="a.jpg" alt="Image A">
			<img src="b.jpg" alt="Image B">
			<img src="c.jpg">
			<img src="d.jpg" alt="">
		</body>
	</html>`

	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	count, withAlt := doc.findImages()
	if count != 4 {
		t.Errorf("findImages() count = %d, want 4", count)
	}
	if withAlt != 2 {
		t.Errorf("findImages() withAlt = %d, want 2", withAlt)
	}
}

func TestHTMLParser_Links(t *testing.T) {
	input := `<html>
		<body>
			<a href="/page1">Internal 1</a>
			<a href="/page2">Internal 2</a>
			<a href="https://external.com">External</a>
			<a href="https://example.com">Same domain</a>
			<a href="">Empty</a>
		</body>
	</html>`

	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	internal, external := doc.findLinks("example.com")
	if internal != 3 {
		t.Errorf("findLinks() internal = %d, want 3", internal)
	}
	if external != 1 {
		t.Errorf("findLinks() external = %d, want 1", external)
	}
}

func TestHTMLParser_WordCount(t *testing.T) {
	input := `<html>
		<body>
			<p>This is a test paragraph with several words.</p>
			<div>
				<p>Another paragraph with more content here.</p>
			</div>
		</body>
	</html>`

	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	body := doc.findElements("body")[0]
	count := body.wordCount()
	// "This is a test paragraph with several words. Another paragraph with more content here."
	// = 14 words
	if count != 14 {
		t.Errorf("wordCount() = %d, want 14", count)
	}
}

func TestHTMLParser_SelfClosingTags(t *testing.T) {
	input := `Text<br><br/>more<hr>end`

	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	text := doc.textContent()
	if text != "Textmoreend" {
		t.Errorf("textContent() = %q, want %q", text, "Textmoreend")
	}
}

func TestHTMLParser_UnclosedTags(t *testing.T) {
	input := `<div><p>Unclosed paragraph</div>`

	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	text := strings.TrimSpace(doc.textContent())
	if text != "Unclosed paragraph" {
		t.Errorf("textContent() = %q, want %q", text, "Unclosed paragraph")
	}
}

func TestHTMLParser_MalformedHTML(t *testing.T) {
	tests := []string{
		`<div><p>Mixed<div>`,
		`<p>Starting without<html>`,
		`<<<<>`,
		`<div attr=without quotes>test</div>`,
		`<img src=x title='mixed"quotes'>`,
	}

	for _, input := range tests {
		t.Run(input[:min(20, len(input))], func(t *testing.T) {
			doc, err := ParseHTML(input)
			if err != nil {
				t.Logf("ParseHTML() error = %v (may be expected)", err)
				return
			}
			if doc == nil {
				t.Error("ParseHTML() returned nil doc")
			}
		})
	}
}

func TestHTMLParser_DeeplyNested(t *testing.T) {
	// Create deeply nested structure
	input := strings.Repeat("<div>", 1000) + "deep" + strings.Repeat("</div>", 1000)

	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	text := strings.TrimSpace(doc.textContent())
	if text != "deep" {
		t.Errorf("textContent() = %q, want %q", text, "deep")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}