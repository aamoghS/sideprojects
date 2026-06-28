package agent

import (
	"context"

	"github.com/aamoghS/sideprojects/movie/internal/scraper"
)

func Run(ctx context.Context, client *scraper.Client, a Agent) Result {
	limit := a.Limit
	seen := make(map[string]bool)
	var movies []Movie

	found := scraper.ScrapeReddit(ctx, client, scraper.RedditInput{
		UserAgent:     a.UserAgent,
		Subreddits:    a.Subreddits,
		Queries:       a.Queries,
		Limit:         limit,
		Discover:      a.Discover != nil && *a.Discover,
		MaxSubreddits: a.MaxSubreddits,
		MaxThreads:    a.MaxThreads,
	})
	for _, m := range found {
		if ctx.Err() != nil {
			break
		}
		key := MovieKey(m.Title, m.Year)
		if seen[key] {
			continue
		}
		seen[key] = true
		movies = append(movies, Movie{
			Title:  m.Title,
			Year:   m.Year,
			Score:  m.Score,
			Source: m.Source,
		})
		if len(movies) >= limit {
			break
		}
	}

	for _, pick := range a.Picks {
		if len(movies) >= limit || ctx.Err() != nil {
			break
		}
		key := MovieKey(pick.Title, pick.Year)
		if seen[key] {
			continue
		}
		seen[key] = true
		movies = append(movies, PickToMovie(pick, "curated fallback"))
	}

	result := Result{Agent: a, Movies: movies}
	if len(movies) == 0 {
		result.Error = "no movies found from reddit or fallbacks"
	}
	return result
}
