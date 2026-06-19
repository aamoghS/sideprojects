package seo

import (
	"fmt"
	"strings"
)

// RobotsRule represents a single rule in robots.txt
type RobotsRule struct {
	UserAgent  string
	Allow      string
	Disallow   string
	CrawlDelay string
}

// RobotsTxt represents a complete robots.txt file
type RobotsTxt struct {
	Rules   []RobotsRule
	Sitemap string
}

// NewRobotsTxt creates a new robots.txt generator
func NewRobotsTxt() *RobotsTxt {
	return &RobotsTxt{
		Rules: []RobotsRule{},
	}
}

// AddRule adds a rule to the robots.txt
func (r *RobotsTxt) AddRule(userAgent, allow, disallow string) *RobotsTxt {
	r.Rules = append(r.Rules, RobotsRule{
		UserAgent: userAgent,
		Allow:     allow,
		Disallow:  disallow,
	})
	return r
}

// AddSitemap adds the sitemap URL
func (r *RobotsTxt) AddSitemap(sitemapURL string) *RobotsTxt {
	r.Sitemap = sitemapURL
	return r
}

// SetCrawlDelay sets crawl-delay for a user agent
func (r *RobotsTxt) SetCrawlDelay(userAgent, delay string) *RobotsTxt {
	for i := range r.Rules {
		if r.Rules[i].UserAgent == userAgent {
			r.Rules[i].CrawlDelay = delay
			return r
		}
	}
	r.Rules = append(r.Rules, RobotsRule{
		UserAgent:  userAgent,
		CrawlDelay: delay,
	})
	return r
}

// Generate creates the robots.txt content
func (r *RobotsTxt) Generate() string {
	var sb strings.Builder

	currentAgent := ""

	for _, rule := range r.Rules {
		if rule.UserAgent != currentAgent {
			if currentAgent != "" {
				sb.WriteString("\n")
			}
			fmt.Fprintf(&sb, "User-agent: %s\n", rule.UserAgent)
			currentAgent = rule.UserAgent
		}

		if rule.Allow != "" {
			fmt.Fprintf(&sb, "Allow: %s\n", rule.Allow)
		}
		if rule.Disallow != "" {
			fmt.Fprintf(&sb, "Disallow: %s\n", rule.Disallow)
		}
		if rule.CrawlDelay != "" {
			fmt.Fprintf(&sb, "Crawl-delay: %s\n", rule.CrawlDelay)
		}
	}

	if r.Sitemap != "" {
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "Sitemap: %s\n", r.Sitemap)
	}

	return sb.String()
}

// CommonRobotsTxt generates a standard robots.txt for a site
func CommonRobotsTxt(baseURL string) string {
	robots := NewRobotsTxt().
		AddSitemap(baseURL+"/sitemap.xml").
		AddRule("*", "", "/").
		AddRule("Googlebot", "", "/").
		AddRule("Bingbot", "", "/").
		AddRule("Googlebot-Image", "", "/images/").
		SetCrawlDelay("*", "1")

	return robots.Generate()
}
