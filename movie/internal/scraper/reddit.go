package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type redditSearchResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditCommentsResponse []struct {
	Data struct {
		Children []struct {
			Data struct {
				Body  string `json:"body"`
				Score int    `json:"score"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditThread struct {
	ID    string
	Title string
}

func redditBlocked(status int) bool {
	return status == http.StatusForbidden || status == http.StatusTooManyRequests
}

func searchRedditSub(ctx context.Context, client *Client, userAgent, sub, query string) (redditThread, error) {
	if ctx.Err() != nil {
		return redditThread{}, ctx.Err()
	}

	searchURL := fmt.Sprintf(
		"https://www.reddit.com/r/%s/search.json?q=%s&restrict_sr=on&sort=comments&t=all&limit=5",
		sub, url.QueryEscape(query),
	)

	resp, err := client.Get(ctx, userAgent, searchURL)
	if err != nil {
		return redditThread{}, err
	}
	defer resp.Body.Close()

	if redditBlocked(resp.StatusCode) {
		return redditThread{}, fmt.Errorf("reddit blocked (%d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return redditThread{}, fmt.Errorf("reddit status %d", resp.StatusCode)
	}

	var result redditSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return redditThread{}, err
	}

	for _, child := range result.Data.Children {
		title := strings.ToLower(child.Data.Title)
		if strings.Contains(title, "series") || strings.Contains(title, "tv show") {
			continue
		}
		return redditThread{ID: child.Data.ID, Title: child.Data.Title}, nil
	}
	return redditThread{}, fmt.Errorf("no thread in r/%s for %q", sub, query)
}

type RedditInput struct {
	UserAgent  string
	Subreddits []string
	Queries    []string
	Limit      int
}

func ScrapeReddit(ctx context.Context, client *Client, in RedditInput) []Movie {
	if client.RedditBlocked.Load() || ctx.Err() != nil {
		return nil
	}

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

			thread, err := searchRedditSub(searchCtx, client, in.UserAgent, t.sub, t.query)
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
		movies []Movie
		title  string
	}
	threadCh := make(chan threadResult, len(threads))

	var scrapeWg sync.WaitGroup
	for _, thread := range threads {
		scrapeWg.Add(1)
		go func(th redditThread) {
			defer scrapeWg.Done()
			found := scrapeRedditThread(ctx, client, in.UserAgent, th.ID, in.Limit*2)
			threadCh <- threadResult{movies: found, title: th.Title}
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
			m.Source = fmt.Sprintf("reddit (%s)", tr.title)
			movies = append(movies, m)
			if len(movies) >= in.Limit {
				return movies
			}
		}
	}
	return movies
}

func scrapeRedditThread(ctx context.Context, client *Client, userAgent, threadID string, limit int) []Movie {
	commentsURL := fmt.Sprintf("https://www.reddit.com/comments/%s.json?sort=top&limit=40", threadID)
	resp, err := client.Get(ctx, userAgent, commentsURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var comments redditCommentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil || len(comments) < 2 {
		return nil
	}

	var movies []Movie
	seen := make(map[string]bool)

	for _, child := range comments[1].Data.Children {
		body := strings.TrimSpace(child.Data.Body)
		score := child.Data.Score
		if body == "" || body == "[deleted]" || body == "[removed]" || score < 3 {
			continue
		}

		for _, movie := range parseMoviesFromComment(body) {
			key := movieKey(movie.Title, movie.Year)
			if seen[key] {
				continue
			}
			seen[key] = true
			movie.Score = score
			movies = append(movies, movie)
			if len(movies) >= limit {
				return movies
			}
		}
	}
	return movies
}
