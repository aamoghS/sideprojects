package scraper

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const redditBase = "https://old.reddit.com"

var (
	threadHrefRE   = regexp.MustCompile(`(?i)/r/([a-zA-Z0-9_]+)/comments/([a-zA-Z0-9]+)/`)
	subredditHrefRE = regexp.MustCompile(`(?i)/r/([a-zA-Z0-9_]{2,21})(?:/|$|"|\?)`)
	commentScoreRE = regexp.MustCompile(`(?i)class="score[^"]*"[^>]*title="(\d+) points"`)
)

func parseThreadsFromHTML(page string) []redditThread {
	seen := make(map[string]bool)
	var threads []redditThread

	titleRE := regexp.MustCompile(`(?is)<a[^>]+class="[^"]*title[^"]*"[^>]+href="(/r/[^"]+/comments/[^"]+)"[^>]*>([^<]+)</a>`)
	for _, m := range titleRE.FindAllStringSubmatch(page, -1) {
		href := m[1]
		title := stripHTML(m[2])
		sub, id := parseThreadHref(href)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		if sub == "" {
			sub = threadSubreddit(href)
		}
		threads = append(threads, redditThread{ID: id, Title: title, Subreddit: normalizeSub(sub)})
	}

	if len(threads) > 0 {
		return threads
	}

	for _, m := range threadHrefRE.FindAllStringSubmatch(page, -1) {
		id := m[2]
		if seen[id] {
			continue
		}
		seen[id] = true
		threads = append(threads, redditThread{
			ID:        id,
			Subreddit: normalizeSub(m[1]),
		})
	}
	return threads
}

func parseThreadHref(href string) (sub, id string) {
	m := threadHrefRE.FindStringSubmatch(href)
	if len(m) < 3 {
		return "", ""
	}
	return m[1], m[2]
}

func threadSubreddit(href string) string {
	m := threadHrefRE.FindStringSubmatch(href)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func parseSubredditsFromHTML(page string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, m := range subredditHrefRE.FindAllStringSubmatch(page, -1) {
		sub := normalizeSub(m[1])
		if sub == "" || seen[sub] {
			continue
		}
		seen[sub] = true
		out = append(out, sub)
	}
	return out
}

func searchRedditSub(ctx context.Context, client *Client, userAgent, sub, query string) (redditThread, error) {
	if ctx.Err() != nil {
		return redditThread{}, ctx.Err()
	}

	pageURL := fmt.Sprintf(
		"%s/r/%s/search?q=%s&restrict_sr=on&sort=comments&t=all",
		redditBase, url.PathEscape(sub), url.QueryEscape(query),
	)
	page, err := fetchPage(ctx, client, userAgent, pageURL)
	if err != nil {
		return redditThread{}, err
	}

	for _, th := range parseThreadsFromHTML(page) {
		title := strings.ToLower(th.Title)
		if strings.Contains(title, "series") || strings.Contains(title, "tv show") {
			continue
		}
		if th.Subreddit == "" {
			th.Subreddit = normalizeSub(sub)
		}
		if th.Title != "" {
			return th, nil
		}
	}
	return redditThread{}, fmt.Errorf("no thread in r/%s for %q", sub, query)
}

func searchRedditGlobal(ctx context.Context, client *Client, userAgent, query string) ([]redditThread, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	pageURL := fmt.Sprintf("%s/search?q=%s&sort=comments&t=all", redditBase, url.QueryEscape(query))
	page, err := fetchPage(ctx, client, userAgent, pageURL)
	if err != nil {
		return nil, err
	}
	return parseThreadsFromHTML(page), nil
}

func listSubredditHot(ctx context.Context, client *Client, userAgent, sub string) ([]redditThread, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	pageURL := fmt.Sprintf("%s/r/%s/hot", redditBase, url.PathEscape(sub))
	page, err := fetchPage(ctx, client, userAgent, pageURL)
	if err != nil {
		return nil, err
	}

	var threads []redditThread
	for _, th := range parseThreadsFromHTML(page) {
		title := strings.ToLower(th.Title)
		if strings.Contains(title, "series") || strings.Contains(title, "tv show") {
			continue
		}
		if recommendationTitle(title) {
			threads = append(threads, th)
		}
	}
	return threads, nil
}

func discoverSubreddits(ctx context.Context, client *Client, userAgent string, terms []string, limit int) []string {
	if limit <= 0 {
		limit = 10
	}
	seen := make(map[string]bool)
	var out []string

	for _, term := range terms {
		if ctx.Err() != nil || len(out) >= limit {
			break
		}
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}

		pageURL := fmt.Sprintf("%s/subreddits/search?q=%s", redditBase, url.QueryEscape(term))
		page, err := fetchPage(ctx, client, userAgent, pageURL)
		if err != nil {
			continue
		}

		for _, sub := range parseSubredditsFromHTML(page) {
			if seen[sub] {
				continue
			}
			if !looksLikeMovieSub(sub, sub, term) {
				continue
			}
			seen[sub] = true
			out = append(out, sub)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func scrapeRedditThreadLearn(ctx context.Context, client *Client, userAgent, threadID string, limit int) ([]Movie, []string) {
	pageURL := fmt.Sprintf("%s/comments/%s/", redditBase, threadID)
	page, err := fetchPage(ctx, client, userAgent, pageURL)
	if err != nil {
		return nil, nil
	}

	scores := commentScoreRE.FindAllStringSubmatch(page, -1)
	bodies := extractMDBlocks(page)

	var movies []Movie
	var learned []string
	seenSub := make(map[string]bool)
	seenMovie := make(map[string]bool)

	addSubs := func(text string) {
		for _, sub := range extractSubredditsFromText(text) {
			if seenSub[sub] {
				continue
			}
			seenSub[sub] = true
			learned = append(learned, sub)
		}
	}

	for i, body := range bodies {
		if body == "" || body == "[deleted]" || body == "[removed]" {
			continue
		}
		addSubs(body)

		score := 5
		if i < len(scores) {
			if n, err := strconv.Atoi(scores[i][1]); err == nil {
				score = n
			}
		}
		if score < 3 {
			continue
		}

		for _, movie := range parseMoviesFromComment(body) {
			key := movieKey(movie.Title, movie.Year)
			if seenMovie[key] {
				continue
			}
			seenMovie[key] = true
			movie.Score = score
			movies = append(movies, movie)
			if len(movies) >= limit {
				return movies, learned
			}
		}
	}
	return movies, learned
}
