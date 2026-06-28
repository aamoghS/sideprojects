package simhttp

import (
	"context"
	"fmt"
	"time"

	"github.com/aamoghS/sideprojects/minstd/chrono"
)

type Duration = chrono.Duration

const (
	Millisecond = chrono.Millisecond
	Second      = chrono.Second
)

func FromDuration(d time.Duration) Duration {
	return chrono.FromStd(d)
}

type Config struct {
	Name          string
	Addr          string
	Latency       Duration
	Jitter        Duration
	ErrorRate     float64
	MetricsWindow Duration
}

type Backend struct {
	metrics *Metrics
	server  *Server
}

func New(cfg Config) (*Backend, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	if cfg.Addr == "" {
		cfg.Addr = ":0"
	}
	if cfg.MetricsWindow <= 0 {
		cfg.MetricsWindow = 250 * Millisecond
	}

	metrics := NewMetrics(cfg.Latency, cfg.MetricsWindow)
	latency := NewLatency(cfg.Latency, cfg.Jitter)
	server := NewServer(cfg.Name, cfg.Addr, cfg.ErrorRate, latency, metrics)

	return &Backend{metrics: metrics, server: server}, nil
}

func (b *Backend) Metrics() *Metrics {
	return b.metrics
}

func (b *Backend) Snapshot() Snapshot {
	return b.metrics.Snapshot()
}

func (b *Backend) Run(ctx context.Context) error {
	return b.server.Run(ctx)
}
