package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aamogh/seotool/pkg/seo"
)

func main() {
	analyzeCmd := flag.NewFlagSet("analyze", flag.ExitOnError)
	analyzeURL := analyzeCmd.String("url", "", "URL to analyze")

	sitemapCmd := flag.NewFlagSet("sitemap", flag.ExitOnError)
	sitemapURL := sitemapCmd.String("url", "", "Base URL for sitemap")
	sitemapDepth := sitemapCmd.Int("depth", 2, "Crawl depth")

	robotsCmd := flag.NewFlagSet("robots", flag.ExitOnError)
	robotsURL := robotsCmd.String("url", "", "Base URL for robots.txt")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "analyze":
		analyzeCmd.Parse(os.Args[2:])
		if *analyzeURL == "" {
			fmt.Println("Error: --url is required")
			os.Exit(1)
		}
		runAnalyze(*analyzeURL)

	case "sitemap":
		sitemapCmd.Parse(os.Args[2:])
		if *sitemapURL == "" {
			fmt.Println("Error: --url is required")
			os.Exit(1)
		}
		runSitemap(*sitemapURL, *sitemapDepth)

	case "robots":
		robotsCmd.Parse(os.Args[2:])
		if *robotsURL == "" {
			fmt.Println("Error: --url is required")
			os.Exit(1)
		}
		runRobots(*robotsURL)

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`SEO Tool - WebBot for SEO Optimization

Usage:
  seotool <command> [options]

Commands:
  analyze   Analyze a URL for SEO issues
  sitemap   Generate sitemap for a site
  robots    Generate robots.txt for a site
  help      Show this help message

Examples:
  seotool analyze --url https://aamogh.vercel.app
  seotool sitemap --url https://aamogh.vercel.app --depth 3
  seotool robots --url https://aamogh.vercel.app`)
}

func runAnalyze(url string) {
	fmt.Printf("Analyzing: %s\n", url)

	analyzer := seo.NewAnalyzer()
	result, err := analyzer.Analyze(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	seo.PrintResult(result)
}

func runSitemap(baseURL string, depth int) {
	fmt.Printf("Generating sitemap for: %s (depth: %d)\n", baseURL, depth)

	gen, err := seo.NewSitemapGenerator(baseURL, depth)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Crawling site...")
	if err := gen.Crawl(); err != nil {
		fmt.Printf("Error during crawl: %v\n", err)
		os.Exit(1)
	}

	urls := gen.GetURLs()
	fmt.Printf("Found %d URLs\n", len(urls))

	fmt.Println("\n--- sitemap.xml ---")
	gen.GenerateSitemap(os.Stdout)
}

func runRobots(baseURL string) {
	fmt.Printf("Generating robots.txt for: %s\n", baseURL)

	content := seo.CommonRobotsTxt(baseURL)

	fmt.Println("\n--- robots.txt ---")
	fmt.Print(content)
}