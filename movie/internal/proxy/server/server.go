package server

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Run starts the forward proxy and blocks until ctx is cancelled or a signal arrives.
func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	p := &handler{
		user:        cfg.User,
		pass:        cfg.Pass,
		authEnabled: cfg.AuthRequired,
		logger:      log.New(os.Stdout, "", log.LstdFlags),
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           http.HandlerFunc(p.dispatch),
		ReadHeaderTimeout: 10 * time.Second,
	}

	p.logger.Printf("proxy listening on %s (auth: %v)", cfg.Addr, cfg.AuthRequired)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	case <-stop:
	}

	p.logger.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

type handler struct {
	user        string
	pass        string
	authEnabled bool
	logger      *log.Logger
}

func (p *handler) dispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}

	if p.authEnabled && !p.authenticated(r) {
		w.Header().Set("Proxy-Authenticate", `Basic realm="proxy"`)
		http.Error(w, "proxy authentication required", http.StatusProxyAuthRequired)
		return
	}

	host := r.Host
	if host == "" {
		host = r.URL.Host
	}
	p.logger.Printf("%s %s %s", r.RemoteAddr, r.Method, host)

	if r.Method == http.MethodConnect {
		p.handleTunnel(w, r)
		return
	}
	p.handleHTTP(w, r)
}

func (p *handler) authenticated(r *http.Request) bool {
	user, pass, ok := proxyBasicAuth(r)
	if !ok {
		return false
	}
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(p.user)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(p.pass)) == 1
	return userOK && passOK
}

func proxyBasicAuth(r *http.Request) (username, password string, ok bool) {
	if username, password, ok = r.BasicAuth(); ok {
		return username, password, true
	}

	const prefix = "Basic "
	header := r.Header.Get("Proxy-Authorization")
	if !strings.HasPrefix(header, prefix) {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(header[len(prefix):])
	if err != nil {
		return "", "", false
	}

	pair := string(decoded)
	colon := strings.IndexByte(pair, ':')
	if colon < 0 {
		return pair, "", true
	}
	return pair[:colon], pair[colon+1:], true
}

func (p *handler) handleTunnel(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout("tcp", r.Host, 30*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)

	hj, ok := w.(http.Hijacker)
	if !ok {
		destConn.Close()
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	srcConn, buf, err := hj.Hijack()
	if err != nil {
		destConn.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if buf != nil && buf.Reader.Buffered() > 0 {
			if _, err := io.Copy(destConn, buf); err != nil {
				return
			}
		}
		_, _ = io.Copy(destConn, srcConn)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(srcConn, destConn)
	}()
	wg.Wait()

	srcConn.Close()
	destConn.Close()
}

func (p *handler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.String()
	if r.URL.Scheme == "" {
		targetURL = "http://" + r.Host + r.URL.RequestURI()
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	removeHopByHopHeaders(req.Header)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	removeHopByHopHeaders(resp.Header)
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func removeHopByHopHeaders(h http.Header) {
	h.Del("Proxy-Connection")
	h.Del("Proxy-Authenticate")
	h.Del("Proxy-Authorization")
	h.Del("Connection")
	h.Del("Keep-Alive")
	h.Del("Te")
	h.Del("Trailer")
	h.Del("Transfer-Encoding")
	h.Del("Upgrade")
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
