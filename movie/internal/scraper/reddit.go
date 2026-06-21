package scraper

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type redditThread struct {
	ID        string
	Title     string
	Subreddit string
}

type RedditInput struct {
	UserAgent     string
	Subreddits    []string
	Queries       []string
	Limit         int
	Discover      bool
	MaxSubreddits int
	MaxThreads    int
}

func ScrapeReddit(ctx context.Context, client *Client, in RedditInput) []Movie {
	if client.RedditBlocked.Load() || ctx.Err() != nil {
		return nil
	}
	if in.Discover {
		return scrapeRedditDiscover(ctx, client, in)
	}
	return scrapeRedditFixed(ctx, client, in)
}

func searchRedditSubWithSub(ctx context.Context, client *Client, userAgent, sub, query string) (redditThread, error) {
	th, err := searchRedditSub(ctx, client, userAgent, sub, query)
	if err != nil {
		return th, err
	}
	if th.Subreddit == "" {
		th.Subreddit = normalizeSub(sub)
	}
	return th, nil
}

func scrapeRedditFixed(ctx context.Context, client *Client, in RedditInput) []Movie {
	type searchTask struct {
		sub   string
		query string
	}
	type searchResult struct {
		thread redditThread
		err    error
	}

	tasks := make([]searchTask, 0, len(in.Queries)*len(in.Subreddits))
	for _, query := range in.Queries {
		for _, sub := range in.Subreddits {
			tasks = append(tasks, searchTask{sub: sub, query: query})
		}
	}
	if len(tasks) == 0 {
		return nil
	}

	searchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan searchResult, len(tasks))
	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(t searchTask) {
			defer wg.Done()
			if searchCtx.Err() != nil {
				return
			}

			thread, err := searchRedditSubWithSub(searchCtx, client, in.UserAgent, t.sub, t.query)
			if err != nil {
				if strings.Contains(err.Error(), "reddit blocked") {
					client.RedditBlocked.Store(true)
					cancel()
				}
				results <- searchResult{err: err}
				return
			}
			results <- searchResult{thread: thread}
		}(task)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	threads := make([]redditThread, 0, len(tasks))
	seenThread := make(map[string]bool)
	for r := range results {
		if r.err != nil || r.thread.ID == "" || seenThread[r.thread.ID] {
			continue
		}
		seenThread[r.thread.ID] = true
		threads = append(threads, r.thread)
	}
	if len(threads) == 0 || ctx.Err() != nil {
		return nil
	}

	type threadResult struct {
		movies    []Movie
		title     string
		subreddit string
	}
	threadCh := make(chan threadResult, len(threads))

	var scrapeWg sync.WaitGroup
	for _, thread := range threads {
		scrapeWg.Add(1)
		go func(th redditThread) {
			defer scrapeWg.Done()
			found, _ := scrapeRedditThreadLearn(ctx, client, in.UserAgent, th.ID, in.Limit*2)
			threadCh <- threadResult{movies: found, title: th.Title, subreddit: th.Subreddit}
		}(thread)
	}

	go func() {
		scrapeWg.Wait()
		close(threadCh)
	}()

	seen := make(map[string]bool)
	var movies []Movie
	for tr := range threadCh {
		for _, m := range tr.movies {
			key := movieKey(m.Title, m.Year)
			if seen[key] {
				continue
			}
			seen[key] = true
			m.Source = formatRedditSource(tr.title, tr.subreddit)
			movies = append(movies, m)
			if len(movies) >= in.Limit {
				return movies
			}
		}
	}
	return movies
}

func scrapeRedditThread(ctx context.Context, client *Client, userAgent, threadID string, limit int) []Movie {
	movies, _ := scrapeRedditThreadLearn(ctx, client, userAgent, threadID, limit)
	return movies
}

func formatRedditSource(threadTitle, subreddit string) string {
	if subreddit != "" {
		return fmt.Sprintf("reddit (r/%s: %s)", subreddit, threadTitle)
	}
	return fmt.Sprintf("reddit (%s)", threadTitle)
}
