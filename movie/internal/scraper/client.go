package scraper

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

const defaultHTTPTimeout = 5 * time.Second

type Client struct {
	http          *http.Client
	sem           chan struct{}
	RedditBlocked atomic.Bool
	proxy         string
}

func NewClient(workers int, transport *http.Transport, proxyURL string) *Client {
	if workers <= 0 {
		workers = 16
	}
	if transport == nil {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}

	return &Client{
		http: &http.Client{
			Timeout:   defaultHTTPTimeout,
			Transport: transport,
		},
		sem:   make(chan struct{}, workers),
		proxy: proxyURL,
	}
}

func (c *Client) Get(ctx context.Context, userAgent, rawURL string) (*http.Response, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	c.sem <- struct{}{}
	defer func() { <-c.sem }()

	reqCtx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return c.http.Do(req)
}
