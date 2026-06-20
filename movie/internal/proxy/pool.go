package proxy

import (
	"sync"

	"movie/internal/scraper"
)

type ClientPool struct {
	workers int
	mu      sync.Mutex
	clients map[string]*scraper.Client
}

func NewClientPool(workers int) *ClientPool {
	return &ClientPool{
		workers: workers,
		clients: make(map[string]*scraper.Client),
	}
}

func (p *ClientPool) Get(proxyURL string) *scraper.Client {
	key := proxyURL
	if key == "" {
		key = "direct"
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if client, ok := p.clients[key]; ok {
		return client
	}

	transport, err := Transport(proxyURL, p.workers)
	if err != nil {
		panic("proxy transport: " + err.Error())
	}

	client := scraper.NewClient(p.workers, transport, proxyURL)
	p.clients[key] = client
	return client
}
