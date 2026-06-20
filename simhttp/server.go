package simhttp

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"minstd/chrono"
	"minstd/http"
)

type Server struct {
	name      string
	addr      string
	errorRate float64
	latency   *Latency
	metrics   *Metrics
}

func NewServer(name, addr string, errorRate float64, latency *Latency, metrics *Metrics) *Server {
	return &Server{
		name:      name,
		addr:      addr,
		errorRate: errorRate,
		latency:   latency,
		metrics:   metrics,
	}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	srv := &http.Server{Addr: s.addr, Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("[%s] http listening on %s", s.name, s.addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		_ = srv.ShutdownChrono(context.Background(), 2*chrono.Second)
		return ctx.Err()
	case err := <-errCh:
		_ = srv.ShutdownChrono(context.Background(), 2*chrono.Second)
		return err
	}
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	done := s.metrics.BeginRequest()
	lat := s.latency.Sample()
	chrono.Sleep(lat)
	ms := float64(lat.Milliseconds())

	if rand.Float64() < s.errorRate {
		done(ms, true)
		http.Error(w, "injected error", http.StatusInternalServerError)
		return
	}

	done(ms, false)
	fmt.Fprintf(w, "ok from %s in %s\n", s.name, lat)
}
