package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aamoghS/sideprojects/surf/pkg/seo"
)

func main() {
	analyzeCmd := flag.NewFlagSet("analyze", flag.ExitOnError)
	analyzeURL := analyzeCmd.String("url", "", "URL to analyze")

	sitemapCmd := flag.NewFlagSet("sitemap", flag.ExitOnError)
	sitemapURL := sitemapCmd.String("url", "", "Base URL for sitemap")
	sitemapDepth := sitemapCmd.Int("depth", 2, "Crawl depth")

	robotsCmd := flag.NewFlagSet("robots", flag.ExitOnError)
	robotsURL := robotsCmd.String("url", "", "Base URL for robots.txt")

	batchCmd := flag.NewFlagSet("batch", flag.ExitOnError)
	batchURLs := batchCmd.String("urls-file", "", "File with one URL per line")
	batchOut := batchCmd.String("output", "results.jsonl", "JSONL output path")
	batchShard := batchCmd.Int("shard-index", -1, "Shard index for manual sharding")
	batchShards := batchCmd.Int("shard-total", 1, "Total shards")

	auditCmd := flag.NewFlagSet("audit", flag.ExitOnError)
	auditURL := auditCmd.String("url", "", "Base URL to crawl and audit")
	auditDepth := auditCmd.Int("depth", 2, "Crawl depth")
	auditOut := auditCmd.String("output", "audit.jsonl", "JSONL audit output")
	auditPlan := auditCmd.String("plan", "", "Optional fix plan output path")

	planCmd := flag.NewFlagSet("plan", flag.ExitOnError)
	planInput := planCmd.String("input", "", "Audit JSONL from audit or batch")
	planOut := planCmd.String("output", "", "Write plan to file (default: stdout)")

	diffCmd := flag.NewFlagSet("diff", flag.ExitOnError)
	diffBefore := diffCmd.String("before", "", "Audit JSONL before fixes")
	diffAfter := diffCmd.String("after", "", "Audit JSONL after fixes")
	diffOut := diffCmd.String("output", "", "Write diff to file (default: stdout)")

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

	case "batch":
		batchCmd.Parse(os.Args[2:])
		if *batchURLs == "" {
			fmt.Println("Error: --urls-file is required")
			os.Exit(1)
		}
		if err := seo.RunBatch(seo.BatchOptions{
			URLsFile:   *batchURLs,
			Output:     *batchOut,
			ShardIdx:   *batchShard,
			ShardTotal: *batchShards,
		}); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "audit":
		auditCmd.Parse(os.Args[2:])
		if *auditURL == "" {
			fmt.Println("Error: --url is required")
			os.Exit(1)
		}
		if err := seo.RunAudit(seo.AuditOptions{
			BaseURL:  *auditURL,
			Depth:    *auditDepth,
			Output:   *auditOut,
			PlanPath: *auditPlan,
		}); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Wrote audit to %s\n", *auditOut)
		if *auditPlan != "" {
			fmt.Printf("Wrote fix plan to %s\n", *auditPlan)
		}

	case "plan":
		planCmd.Parse(os.Args[2:])
		if *planInput == "" {
			fmt.Println("Error: --input is required")
			os.Exit(1)
		}
		if *planOut == "" {
			records, err := seo.LoadAuditRecords(*planInput)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			seo.PrintFixPlan(records, os.Stdout)
		} else if err := seo.WriteFixPlan(*planInput, *planOut); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "diff":
		diffCmd.Parse(os.Args[2:])
		if *diffBefore == "" || *diffAfter == "" {
			fmt.Println("Error: --before and --after are required")
			os.Exit(1)
		}
		if *diffOut == "" {
			if err := seo.RunDiff(*diffBefore, *diffAfter, os.Stdout); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		} else if err := seo.WriteDiff(*diffBefore, *diffAfter, *diffOut); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

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
  audit     Crawl a site and write JSONL audit + optional fix plan
  batch     Analyze many URLs (JSONL out, K8s shard friendly)
  plan      Build prioritized fix plan from audit JSONL
  diff      Compare before/after audits (score deltas, cleared fixes)
  sitemap   Generate sitemap for a site
  robots    Generate robots.txt for a site
  help      Show this help message

Examples:
  seotool analyze --url https://example.com
  seotool audit --url https://example.com --output audit.jsonl --plan fixes.txt
  seotool plan --input audit.jsonl
  seotool diff --before audit-before.jsonl --after audit-after.jsonl
  seotool batch --urls-file urls.txt --output results.jsonl
  seotool sitemap --url https://example.com --depth 3
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