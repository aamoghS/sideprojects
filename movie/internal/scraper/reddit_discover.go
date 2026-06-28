package scraper

import (
	"context"
	"net/http"
	"regexp"
	"strings"
)

var subredditMention = regexp.MustCompile(`(?i)(?:/r/|(?:^|[\s(])r/)([a-zA-Z0-9_]{2,21})`)

var blockedSubreddits = map[string]bool{
	"all": true, "popular": true, "askreddit": true, "todayilearned": true,
	"news": true, "worldnews": true, "pics": true, "funny": true, "videos": true,
	"memes": true, "gaming": true, "technology": true, "science": true,
}

func redditBlocked(status int) bool {
	return status == http.StatusForbidden || status == http.StatusTooManyRequests
}

func normalizeSub(name string) string {
	name = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(name), "r/"))
	if name == "" || blockedSubreddits[name] {
		return ""
	}
	return name
}

func looksLikeMovieSub(name, title, desc string) bool {
	name = strings.ToLower(name)
	text := strings.ToLower(title + " " + desc)
	cues := []string{
		"github.com/aamoghS/sideprojects/movie", "film", "cinema", "horror", "scifi", "sci-fi", "action",
		"comedy", "drama", "thriller", "documentary", "criterion", "flicks",
		"boxoffice", "marvel", "netflix", "watch", "screen", "director",
	}
	for _, cue := range cues {
		if strings.Contains(name, cue) || strings.Contains(text, cue) {
			return true
		}
	}
	return false
}

func extractSubredditsFromText(text string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, m := range subredditMention.FindAllStringSubmatch(text, -1) {
		sub := normalizeSub(m[1])
		if sub == "" || seen[sub] {
			continue
		}
		seen[sub] = true
		out = append(out, sub)
	}
	return out
}

func recommendationTitle(title string) bool {
	cues := []string{
		"recommend", "suggestion", "suggest", "best ", "favorite", "favourite",
		"what to watch", "must see", "must-see", "underrated", "hidden gem",
		"movies like", "similar to", "top ", "greatest", "pick", "list",
	}
	for _, cue := range cues {
		if strings.Contains(title, cue) {
			return true
		}
	}
	return strings.Contains(title, "github.com/aamoghS/sideprojects/movie")
}

func scrapeRedditDiscover(ctx context.Context, client *Client, in RedditInput) []Movie {
	maxSubs := in.MaxSubreddits
	if maxSubs <= 0 {
		maxSubs = 20
	}
	maxThreads := in.MaxThreads
	if maxThreads <= 0 {
		maxThreads = 25
	}

	visitedSubs := make(map[string]bool)
	seenThreads := make(map[string]bool)
	seenMovies := make(map[string]bool)
	var frontier []string
	var pending []redditThread
	var movies []Movie

	enqueueSub := func(name string) {
		sub := normalizeSub(name)
		if sub == "" || visitedSubs[sub] {
			return
		}
		for _, existing := range frontier {
			if existing == sub {
				return
			}
		}
		if len(visitedSubs)+len(frontier) >= maxSubs {
			return
		}
		frontier = append(frontier, sub)
	}

	addThread := func(th redditThread) {
		if th.ID == "" || seenThreads[th.ID] {
			return
		}
		if len(seenThreads) >= maxThreads {
			return
		}
		seenThreads[th.ID] = true
		if th.Subreddit != "" {
			enqueueSub(th.Subreddit)
		}
		pending = append(pending, th)
	}

	scrapePending := func() {
		for len(pending) > 0 && len(movies) < in.Limit && ctx.Err() == nil {
			th := pending[0]
			pending = pending[1:]

			found, learned := scrapeRedditThreadLearn(ctx, client, in.UserAgent, th.ID, in.Limit*2)
			for _, sub := range learned {
				enqueueSub(sub)
			}
			for _, m := range found {
				key := movieKey(m.Title, m.Year)
				if seenMovies[key] {
					continue
				}
				seenMovies[key] = true
				m.Source = formatRedditSource(th.Title, th.Subreddit)
				movies = append(movies, m)
				if len(movies) >= in.Limit {
					return
				}
			}
		}
	}

	for _, sub := range in.Subreddits {
		enqueueSub(sub)
	}

	searchTerms := append([]string{}, in.Queries...)
	for _, q := range in.Queries {
		searchTerms = append(searchTerms, q+" movies")
	}
	for _, sub := range discoverSubreddits(ctx, client, in.UserAgent, searchTerms, maxSubs) {
		enqueueSub(sub)
	}

	for _, query := range in.Queries {
		if ctx.Err() != nil {
			break
		}
		found, err := searchRedditGlobal(ctx, client, in.UserAgent, query)
		if err != nil {
			if strings.Contains(err.Error(), "reddit blocked") {
				client.RedditBlocked.Store(true)
				scrapePending()
				return movies
			}
			continue
		}
		for _, th := range found {
			title := strings.ToLower(th.Title)
			if strings.Contains(title, "series") || strings.Contains(title, "tv show") {
				continue
			}
			addThread(th)
		}
	}
	scrapePending()

	for len(frontier) > 0 && len(movies) < in.Limit && ctx.Err() == nil {
		sub := frontier[0]
		frontier = frontier[1:]
		if visitedSubs[sub] {
			continue
		}
		visitedSubs[sub] = true

		if hot, err := listSubredditHot(ctx, client, in.UserAgent, sub); err == nil {
			for _, th := range hot {
				addThread(th)
			}
		} else if strings.Contains(err.Error(), "reddit blocked") {
			client.RedditBlocked.Store(true)
			break
		}

		for _, query := range in.Queries {
			if ctx.Err() != nil || len(seenThreads) >= maxThreads {
				break
			}
			th, err := searchRedditSubWithSub(ctx, client, in.UserAgent, sub, query)
			if err != nil {
				if strings.Contains(err.Error(), "reddit blocked") {
					client.RedditBlocked.Store(true)
					break
				}
				continue
			}
			addThread(th)
		}

		scrapePending()
	}

	scrapePending()
	return movies
}
