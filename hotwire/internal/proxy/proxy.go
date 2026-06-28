package proxy

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/aamoghS/sideprojects/hotwire/internal/client"
	hotwirev1 "github.com/aamoghS/sideprojects/hotwire/proto/hotwire/v1"
)

type Route struct {
	BackendID string
	URL       string
}

type Server struct {
	controlAddr string
	routes      []Route
	listen      string

	mu      sync.RWMutex
	weights map[string]float64
	proxies map[string]*httputil.ReverseProxy
}

func New(controlAddr, listen string, routes []Route) *Server {
	proxies := make(map[string]*httputil.ReverseProxy, len(routes))
	for _, r := range routes {
		u, err := url.Parse(r.URL)
		if err != nil {
			continue
		}
		proxies[r.BackendID] = httputil.NewSingleHostReverseProxy(u)
	}
	return &Server{
		controlAddr: controlAddr,
		routes:      routes,
		listen:      listen,
		weights:     make(map[string]float64),
		proxies:     proxies,
	}
}

func (s *Server) Run(ctx context.Context) error {
	conn, cp, err := client.Dial(ctx, s.controlAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	stream, err := client.SubscribeWeights(ctx, cp)
	if err != nil {
		return err
	}

	go func() {
		for {
			update, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
			s.applyUpdate(update)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)

	srv := &http.Server{Addr: s.listen, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Printf("proxy listening on %s (control %s)\n", s.listen, s.controlAddr)
	return srv.ListenAndServe()
}

func (s *Server) applyUpdate(update *hotwirev1.WeightUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range update.GetWeights() {
		s.weights[w.GetBackendId()] = w.GetWeight()
	}
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	id := s.pick()
	if id == "" {
		http.Error(w, "no healthy backends", http.StatusServiceUnavailable)
		return
	}
	s.mu.RLock()
	p := s.proxies[id]
	s.mu.RUnlock()
	if p == nil {
		http.Error(w, "backend not configured", http.StatusBadGateway)
		return
	}
	p.ServeHTTP(w, r)
}

func (s *Server) pick() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0.0
	ids := make([]string, 0, len(s.routes))
	cum := make([]float64, 0, len(s.routes))
	for _, route := range s.routes {
		w := s.weights[route.BackendID]
		if w <= 0 {
			continue
		}
		total += w
		ids = append(ids, route.BackendID)
		cum = append(cum, total)
	}
	if total <= 0 || len(ids) == 0 {
		return ""
	}

	roll := rand.Float64() * total
	for i, bound := range cum {
		if roll <= bound {
			return ids[i]
		}
	}
	return ids[len(ids)-1]
}
