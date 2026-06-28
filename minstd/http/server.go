package http

import (
	"context"

	"github.com/aamoghS/sideprojects/minstd/chrono"
	"github.com/aamoghS/sideprojects/minstd/errors"
	"github.com/aamoghS/sideprojects/minstd/net"
)

var ErrServerClosed = errors.New("http: server closed")

type Handler interface {
	ServeHTTP(w ResponseWriter, r *Request)
}

type HandlerFunc func(w ResponseWriter, r *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}

type ServeMux struct {
	routes map[string]Handler
}

func NewServeMux() *ServeMux {
	return &ServeMux{routes: make(map[string]Handler)}
}

func (m *ServeMux) HandleFunc(pattern string, fn HandlerFunc) {
	m.routes[pattern] = fn
}

func (m *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	if h, ok := m.routes[r.Path]; ok {
		h.ServeHTTP(w, r)
		return
	}
	if h, ok := m.routes["/"]; ok {
		h.ServeHTTP(w, r)
		return
	}
	Error(w, "not found", StatusNotFound)
}

type Server struct {
	Addr    string
	Handler Handler

	ln net.Listener
}

func (s *Server) ListenAndServe() error {
	if s.Handler == nil {
		return errors.New("http: nil handler")
	}

	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return ErrServerClosed
			}
			return err
		}
		go s.serveConn(conn)
	}
}

func (s *Server) serveConn(conn net.Conn) {
	defer conn.Close()

	req, err := readRequest(conn)
	if err != nil {
		return
	}

	w := newResponseWriter(conn)
	s.Handler.ServeHTTP(w, req)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.ln == nil {
		return nil
	}
	err := s.ln.Close()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) ShutdownChrono(ctx context.Context, timeout chrono.Duration) error {
	shutdownCtx, cancel := chrono.WithTimeout(ctx, timeout)
	defer cancel()
	return s.Shutdown(shutdownCtx)
}
