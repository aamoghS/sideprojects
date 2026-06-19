package seo

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
	"unicode"
)

// NodeType represents the type of an HTML node
type NodeType int

const (
	ErrorNode NodeType = iota
	TextNode
	ElementNode
)

// Node represents an HTML node (element or text)
type Node struct {
	Type     NodeType
	Data     string
	Attr     []Attribute
	FirstChild, LastChild, NextSibling, PrevSibling *Node
}

// Attribute represents an HTML attribute
type Attribute struct {
	Key, Val string
}

// HTMLParser is a minimal HTML parser using only standard library
type HTMLParser struct {
	s   string
	pos int
}

// NewHTMLParser creates a new HTML parser
func NewHTMLParser(s string) *HTMLParser {
	return &HTMLParser{s: s, pos: 0}
}

// Parse parses HTML into a Node tree
func (p *HTMLParser) Parse() (*Node, error) {
	root := &Node{Type: ElementNode, Data: "root"}
	p.skipWhitespace()

	for p.pos < len(p.s) {
		if p.match("<!DOCTYPE") || p.match("<!doctype") {
			p.skipUntil('>')
			p.pos++
			p.skipWhitespace()
			continue
		}
		if p.match("<!--") {
			if err := p.parseComment(); err != nil {
				return nil, err
			}
			continue
		}
		if p.match("<") {
			node, err := p.parseElement()
			if err != nil {
				return nil, err
			}
			root.appendChild(node)
			p.skipWhitespace()
			continue
		}
		// Text content
		text := p.parseText()
		if text != "" {
			root.appendChild(&Node{Type: TextNode, Data: text})
		}
	}
	return root, nil
}

func (p *HTMLParser) match(s string) bool {
	return strings.HasPrefix(p.s[p.pos:], s)
}

func (p *HTMLParser) skipWhitespace() {
	for p.pos < len(p.s) && unicode.IsSpace(rune(p.s[p.pos])) {
		p.pos++
	}
}

func (p *HTMLParser) parseElement() (*Node, error) {
	p.pos++ // skip '<'

	// Check for closing tag
	if p.pos < len(p.s) && p.s[p.pos] == '/' {
		p.pos++
		tagName := p.parseTagName()
		p.skipUntil('>')
		p.pos++
		return &Node{Type: ElementNode, Data: tagName, Attr: []Attribute{{Key: "/", Val: ""}}}, nil
	}

	tagName := p.parseTagName()
	node := &Node{Type: ElementNode, Data: strings.ToLower(tagName)}

	// Parse attributes
	for p.pos < len(p.s) && p.s[p.pos] != '>' && p.s[p.pos] != '/' {
		p.skipWhitespace()
		if p.pos >= len(p.s) || p.s[p.pos] == '>' || p.s[p.pos] == '/' {
			break
		}
		attrKey := p.parseAttributeName()
		var attrVal string
		if p.pos < len(p.s) && p.s[p.pos] == '=' {
			p.pos++
			attrVal = p.parseAttributeValue()
		}
		node.Attr = append(node.Attr, Attribute{Key: strings.ToLower(attrKey), Val: attrVal})
	}

	// Self-closing tag
	if p.pos < len(p.s) && p.s[p.pos] == '/' {
		p.pos++
	}

	if p.pos < len(p.s) && p.s[p.pos] == '>' {
		p.pos++
	}

	// Check for void elements (self-closing)
	voidElements := map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"embed": true, "hr": true, "img": true, "input": true,
		"link": true, "meta": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}

	if !voidElements[node.Data] {
		// Parse children
		for p.pos < len(p.s) && !p.match("</"+node.Data) {
			p.skipWhitespace()
			if p.match("<!--") {
				p.parseComment()
				continue
			}
			if p.match("<") {
				child, err := p.parseElement()
				if err != nil {
					return nil, err
				}
				if child.hasAttr("/") {
					// This was actually a closing tag
					break
				}
				node.appendChild(child)
				p.skipWhitespace()
				continue
			}
			text := p.parseText()
			if text != "" {
				node.appendChild(&Node{Type: TextNode, Data: text})
			}
		}

		// Skip closing tag
		if p.match("</") {
			p.pos += 2
			p.skipUntil('>')
			p.pos++
		}
	}

	return node, nil
}

func (p *HTMLParser) parseTagName() string {
	start := p.pos
	for p.pos < len(p.s) && (unicode.IsLetter(rune(p.s[p.pos])) || unicode.IsDigit(rune(p.s[p.pos])) || p.s[p.pos] == '-' || p.s[p.pos] == '_') {
		p.pos++
	}
	return p.s[start:p.pos]
}

func (p *HTMLParser) parseAttributeName() string {
	start := p.pos
	for p.pos < len(p.s) && !unicode.IsSpace(rune(p.s[p.pos])) && p.s[p.pos] != '=' && p.s[p.pos] != '>' {
		p.pos++
	}
	return p.s[start:p.pos]
}

func (p *HTMLParser) parseAttributeValue() string {
	p.skipWhitespace()
	if p.pos >= len(p.s) {
		return ""
	}

	quote := p.s[p.pos]
	if quote == '"' || quote == '\'' {
		p.pos++
		start := p.pos
		for p.pos < len(p.s) && p.s[p.pos] != quote {
			p.pos++
		}
		val := p.s[start:p.pos]
		if p.pos < len(p.s) {
			p.pos++
		}
		return val
	}

	// Unquoted value
	start := p.pos
	for p.pos < len(p.s) && !unicode.IsSpace(rune(p.s[p.pos])) && p.s[p.pos] != '>' {
		p.pos++
	}
	return p.s[start:p.pos]
}

func (p *HTMLParser) parseText() string {
	start := p.pos
	for p.pos < len(p.s) && p.s[p.pos] != '<' {
		p.pos++
	}
	return p.s[start:p.pos]
}

func (p *HTMLParser) skipUntil(ch byte) {
	for p.pos < len(p.s) && p.s[p.pos] != ch {
		p.pos++
	}
}

func (p *HTMLParser) parseComment() error {
	p.pos += 4 // skip <!--
	for p.pos < len(p.s)-2 {
		if p.s[p.pos] == '-' && p.s[p.pos+1] == '-' && p.s[p.pos+2] == '>' {
			p.pos += 3
			return nil
		}
		p.pos++
	}
	p.pos = len(p.s)
	return errors.New("unclosed comment")
}

func (n *Node) appendChild(child *Node) {
	if n.LastChild != nil {
		n.LastChild.NextSibling = child
		child.PrevSibling = n.LastChild
	} else {
		n.FirstChild = child
	}
	n.LastChild = child
}

func (n *Node) hasAttr(key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func (n *Node) getAttr(key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// textContent returns all text content from a node and its children
func (n *Node) textContent() string {
	if n.Type == TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(c.textContent())
	}
	return sb.String()
}

// findElements finds all elements with the given tag name
func (n *Node) findElements(tagName string) []*Node {
	tagName = strings.ToLower(tagName)
	var results []*Node
	if n.Type == ElementNode && n.Data == tagName {
		results = append(results, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		results = append(results, c.findElements(tagName)...)
	}
	return results
}

// findMetaByName finds meta tag by name attribute
func (n *Node) findMetaByName(name string) string {
	if n.Type == ElementNode && n.Data == "meta" {
		nameAttr := n.getAttr("name")
		if strings.EqualFold(nameAttr, name) {
			return n.getAttr("content")
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := c.findMetaByName(name); result != "" {
			return result
		}
	}
	return ""
}

// findMetaByProperty finds meta tag by property attribute
func (n *Node) findMetaByProperty(prop string) bool {
	if n.Type == ElementNode && n.Data == "meta" {
		propAttr := n.getAttr("property")
		if strings.HasPrefix(propAttr, prop) {
			return true
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.findMetaByProperty(prop) {
			return true
		}
	}
	return false
}

// findLinkByRel finds link tag by rel attribute
func (n *Node) findLinkByRel(rel string) string {
	if n.Type == ElementNode && n.Data == "link" {
		relAttr := n.getAttr("rel")
		if strings.EqualFold(relAttr, rel) {
			return n.getAttr("href")
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := c.findLinkByRel(rel); result != "" {
			return result
		}
	}
	return ""
}

// findImages finds all img elements and returns count and count with alt
func (n *Node) findImages() (count, withAlt int) {
	if n.Type == ElementNode && n.Data == "img" {
		count = 1
		if n.getAttr("alt") != "" {
			withAlt = 1
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		c, a := c.findImages()
		count += c
		withAlt += a
	}
	return
}

// findLinks finds all anchor elements and returns internal and external counts
func (n *Node) findLinks(baseHost string) (internal, external int) {
	if n.Type == ElementNode && n.Data == "a" {
		href := n.getAttr("href")
		if href != "" {
			if strings.HasPrefix(href, "http") {
				linkHost := strings.TrimPrefix(strings.TrimPrefix(href, "https://"), "http://")
				if idx := strings.Index(linkHost, "/"); idx > 0 {
					linkHost = linkHost[:idx]
				}
				if linkHost == baseHost {
					internal = 1
				} else {
					external = 1
				}
			} else if strings.HasPrefix(href, "/") || href == "" {
				internal = 1
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		i, e := c.findLinks(baseHost)
		internal += i
		external += e
	}
	return
}

// wordCount counts words in text content
func (n *Node) wordCount() int {
	text := n.textContent()
	words := strings.Fields(text)
	return len(words)
}

// ParseHTML parses HTML string into a Node tree
func ParseHTML(htmlStr string) (*Node, error) {
	parser := NewHTMLParser(htmlStr)
	return parser.Parse()
}

// ParseHTMLFromReader parses HTML from an io.Reader
func ParseHTMLFromReader(r io.Reader) (*Node, error) {
	// Read all content
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		return nil, err
	}
	return ParseHTML(buf.String())
}

// NewDocumentFromReader creates a document from a reader (goquery-compatible interface)
func NewDocumentFromReader(r io.Reader) (*Node, error) {
	return ParseHTMLFromReader(bufio.NewReader(r))
}