package app

import (
	"fmt"
	"strings"
	"time"

	"movie/internal/agent"
	"movie/internal/proxy"
)

func printBanner(opts Options, cfg agent.Config, docket proxy.Docket) {
	fmt.Println("Movie Finder — in-house Go agents")
	fmt.Printf("Config: %s (%d agents)\n", opts.Config, len(cfg.Agents))
	fmt.Printf("Workers: %d concurrent requests\n", opts.Workers)
	fmt.Printf("Timeout: %s (use -timeout to adjust)\n", opts.Timeout)

	docketProxies := proxy.EnabledProxies(docket)
	if len(docketProxies) > 0 {
		mode := "round-robin per agent"
		if strings.TrimSpace(docket.DefaultProxy) != "" {
			mode = fmt.Sprintf("fixed entry %q", docket.DefaultProxy)
		}
		fmt.Printf("Proxy docket: %s (%d enabled proxies, %s)\n", opts.Docket, len(docketProxies), mode)
	} else if strings.TrimSpace(opts.Docket) != "" {
		fmt.Printf("Proxy docket: %s (no enabled proxies — see config/PROXY-SETUP.md)\n", opts.Docket)
	}
	if len(cfg.Proxies) > 0 {
		fmt.Printf("Proxy pool: %d proxies (rotated per agent)\n", len(cfg.Proxies))
	} else if cfg.Proxy != "" || opts.Proxy != "" {
		fmt.Printf("Proxy: %s\n", proxy.Mask(proxy.FirstNonEmpty(cfg.Proxy, opts.Proxy, opts.EnvProxy())))
	} else {
		fmt.Println("Proxy: direct (your IP — use -proxy or config proxies to avoid bans)")
	}
	if opts.Offline {
		fmt.Println("Mode: offline (curated picks only)")
	} else if opts.Sequential {
		fmt.Println("Mode: scrape Reddit + Wikipedia (sequential agents)")
	} else {
		fmt.Println("Mode: scrape Reddit + Wikipedia (parallel agents)")
	}
	fmt.Println(strings.Repeat("=", 80))
}

func printResults(results []agent.Result, start time.Time, agentCount int) {
	total := 0
	for _, result := range results {
		a := result.Agent
		fmt.Printf("\n[%s]\n", a.Name)
		fmt.Printf("  Agent ID: %s\n", a.ID)
		fmt.Printf("  User-Agent: %s\n", a.UserAgent)
		fmt.Printf("  Proxy: %s\n", proxy.Mask(result.Proxy))

		if result.Error != "" && len(result.Movies) == 0 {
			fmt.Printf("  Error: %s\n", result.Error)
			continue
		}

		for i, movie := range result.Movies {
			source := movie.Source
			if movie.Score > 0 {
				source = fmt.Sprintf("%s, %d upvotes", source, movie.Score)
			}
			fmt.Printf("\n  %d. %s\n", i+1, agent.FormatTitle(movie.Title, movie.Year))
			fmt.Printf("     Source: %s\n", source)
			fmt.Printf("     Plot: %s\n", movie.Plot)
			total++
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 80))
	fmt.Printf("Done in %s (%d movies from %d agents)\n", time.Since(start).Round(time.Millisecond), total, agentCount)
}
