package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aamoghS/sideprojects/movie/internal/agent"
	"github.com/aamoghS/sideprojects/movie/internal/proxy"
	"github.com/aamoghS/sideprojects/movie/internal/scraper"
)

func Run(ctx context.Context, opts Options) error {
	cfg, err := agent.LoadConfig(opts.Config)
	if err != nil {
		return err
	}
	cfg, err = selectAgents(cfg, opts)
	if err != nil {
		return err
	}

	docket, err := proxy.LoadDocket(opts.Docket)
	if err != nil && strings.TrimSpace(opts.Docket) != "" {
		return err
	}

	pool := proxy.NewClientPool(opts.Workers)

	if opts.TestProxies {
		if err := proxy.TestProxies(ctx, pool, docket, opts.Proxy, opts.EnvProxy()); err != nil {
			return err
		}
		return nil
	}

	printBanner(opts, cfg, docket)

	start := time.Now()
	results, err := runAllAgents(ctx, opts, pool, cfg, docket)
	if err != nil {
		return err
	}

	if !opts.Offline {
		enrichPlots(ctx, pool, results)
	}

	if opts.Output != "" {
		if err := writeJSONResults(opts.Output, results, start); err != nil {
			return err
		}
	} else {
		printResults(results, start, len(cfg.Agents))
	}
	return nil
}

func selectAgents(cfg agent.Config, opts Options) (agent.Config, error) {
	if opts.AgentID != "" {
		return agent.FilterByID(cfg, opts.AgentID)
	}
	idx := opts.AgentIndex
	if idx < 0 {
		if v := strings.TrimSpace(os.Getenv("JOB_COMPLETION_INDEX")); v != "" {
			var err error
			idx, err = strconv.Atoi(v)
			if err != nil {
				return agent.Config{}, fmt.Errorf("JOB_COMPLETION_INDEX: %w", err)
			}
		}
	}
	if idx >= 0 {
		return agent.FilterByIndex(cfg, idx)
	}
	return cfg, nil
}

func runAllAgents(ctx context.Context, opts Options, pool *proxy.ClientPool, cfg agent.Config, docket proxy.Docket) ([]agent.Result, error) {
	if opts.Sequential {
		results := make([]agent.Result, 0, len(cfg.Agents))
		for i, a := range cfg.Agents {
			if ctx.Err() != nil {
				break
			}
			result, err := runOneAgent(ctx, opts, pool, cfg, docket, i, a)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return results, nil
	}

	results := make([]agent.Result, len(cfg.Agents))
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	for i, a := range cfg.Agents {
		wg.Add(1)
		go func(idx int, ag agent.Agent) {
			defer wg.Done()
			result, err := runOneAgent(ctx, opts, pool, cfg, docket, idx, ag)
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			results[idx] = result
		}(i, a)
	}

	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
		return results, nil
	}
}

func runOneAgent(ctx context.Context, opts Options, pool *proxy.ClientPool, cfg agent.Config, docket proxy.Docket, agentIdx int, a agent.Agent) (agent.Result, error) {
	proxyURL, err := proxy.Resolve(a, agentIdx, cfg, docket, opts.Proxy)
	if err != nil {
		return agent.Result{}, err
	}

	if opts.Offline {
		limit := a.Limit
		var movies []agent.Movie
		for _, pick := range a.Picks {
			movies = append(movies, agent.PickToMovie(pick, "curated"))
			if len(movies) >= limit {
				break
			}
		}
		return agent.Result{Agent: a, Movies: movies, Proxy: proxyURL}, nil
	}

	fmt.Printf("  [%s] starting...\n", a.Name)
	client := pool.Get(proxyURL)
	result := agent.Run(ctx, client, a)
	result.Proxy = proxyURL
	fmt.Printf("  [%s] done (%d movies)\n", a.Name, len(result.Movies))
	return result, nil
}

type plotJob struct {
	client    *scraper.Client
	agentIdx  int
	movieIdx  int
	wikiPage  string
	userAgent string
	title     string
	year      int
}

func enrichPlots(ctx context.Context, pool *proxy.ClientPool, results []agent.Result) {
	pickPlots := make(map[int]map[string]string)
	pickWiki := make(map[int]map[string]string)
	for i, r := range results {
		pickPlots[i] = make(map[string]string)
		pickWiki[i] = make(map[string]string)
		for _, p := range r.Agent.Picks {
			key := agent.MovieKey(p.Title, p.Year)
			if p.Plot != "" {
				pickPlots[i][key] = p.Plot
			}
			if p.WikiPage != "" {
				pickWiki[i][key] = p.WikiPage
			}
		}
	}

	var jobs []plotJob
	for ai, r := range results {
		client := pool.Get(r.Proxy)
		for mi, movie := range r.Movies {
			if movie.Plot != "" {
				continue
			}
			key := agent.MovieKey(movie.Title, movie.Year)
			if plot, ok := pickPlots[ai][key]; ok {
				results[ai].Movies[mi].Plot = plot
				continue
			}
			jobs = append(jobs, plotJob{
				client:    client,
				agentIdx:  ai,
				movieIdx:  mi,
				wikiPage:  pickWiki[ai][key],
				userAgent: r.Agent.UserAgent,
				title:     movie.Title,
				year:      movie.Year,
			})
		}
	}

	if len(jobs) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, job := range jobs {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(j plotJob) {
			defer wg.Done()
			plot := scraper.ScrapeWikiPlot(ctx, j.client, j.userAgent, j.title, j.year, j.wikiPage)
			if plot == "" {
				plot = "Plot unavailable."
			}
			results[j.agentIdx].Movies[j.movieIdx].Plot = plot
		}(job)
	}
	wg.Wait()
}
