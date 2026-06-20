package control

import (
	"sync"
	"time"
)

const DefaultStaleThreshold = 3 * time.Second

type AgentMetrics struct {
	BackendID string
	P50Ms     float64
	P99Ms     float64
	ErrorRate float64
	Inflight  int32
	RPS       float64
	LastReport time.Time
}

type Registry struct {
	mu              sync.RWMutex
	agents          map[string]*AgentMetrics
	staleThreshold  time.Duration
}

func NewRegistry(staleThreshold time.Duration) *Registry {
	if staleThreshold <= 0 {
		staleThreshold = DefaultStaleThreshold
	}
	return &Registry{
		agents:         make(map[string]*AgentMetrics),
		staleThreshold: staleThreshold,
	}
}

func (r *Registry) Update(m AgentMetrics) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m.LastReport.IsZero() {
		m.LastReport = time.Now()
	}
	cp := m
	r.agents[m.BackendID] = &cp
}

func (r *Registry) Snapshot(now time.Time) []AgentMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]AgentMetrics, 0, len(r.agents))
	for _, a := range r.agents {
		out = append(out, *a)
	}
	return out
}

func (r *Registry) IsStale(a AgentMetrics, now time.Time) bool {
	if a.LastReport.IsZero() {
		return true
	}
	return now.Sub(a.LastReport) > r.staleThreshold
}

func (r *Registry) StaleThreshold() time.Duration {
	return r.staleThreshold
}

func (r *Registry) Remove(backendID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, backendID)
}
