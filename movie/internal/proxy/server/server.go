package server

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
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
		logger:      log.New(os.Stdout, "[PROXY] ", log.Ldate|log.Ltime|log.Lmicroseconds),
		rateLimiter: newRateLimiter(100, 20), // 100 req/sec, burst of 20
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           http.HandlerFunc(p.dispatch),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	p.logger.Printf("Advanced Proxy listening on %s (Auth: %v, Throttling: Active, Spoofing: Active)", cfg.Addr, cfg.AuthRequired)

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

	p.logger.Println("Initiating graceful shutdown...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

// ---------------------------------------------------------
// Rate Limiter (Token Bucket per IP)
// ---------------------------------------------------------

type rateLimiter struct {
	sync.Mutex
	ips    map[string]*tokenBucket
	rate   float64
	burst  int
}

type tokenBucket struct {
	tokens     float64
	lastUpdate time.Time
}

func newRateLimiter(rate float64, burst int) *rateLimiter {
	return &rateLimiter{
		ips:   make(map[string]*tokenBucket),
		rate:  rate,
		burst: burst,
	}
}

func (rl *rateLimiter) Allow(ip string) bool {
	rl.Lock()
	defer rl.Unlock()

	now := time.Now()
	bucket, exists := rl.ips[ip]
	if !exists {
		bucket = &tokenBucket{tokens: float64(rl.burst), lastUpdate: now}
		rl.ips[ip] = bucket
	} else {
		elapsed := now.Sub(bucket.lastUpdate).Seconds()
		bucket.tokens += elapsed * rl.rate
		if bucket.tokens > float64(rl.burst) {
			bucket.tokens = float64(rl.burst)
		}
		bucket.lastUpdate = now
	}

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}
	return false
}

// ---------------------------------------------------------
// Bandwidth Throttling Connection Wrapper
// ---------------------------------------------------------

type throttledConn struct {
	net.Conn
	bpsLimit int64 // bytes per second limit
}

func (c *throttledConn) Read(b []byte) (int, error) {
	start := time.Now()
	n, err := c.Conn.Read(b)
	if n > 0 && c.bpsLimit > 0 {
		expectedDuration := time.Duration(float64(n)/float64(c.bpsLimit)*float64(time.Second))
		elapsed := time.Since(start)
		if expectedDuration > elapsed {
			time.Sleep(expectedDuration - elapsed)
		}
	}
	return n, err
}

func (c *throttledConn) Write(b []byte) (int, error) {
	start := time.Now()
	n, err := c.Conn.Write(b)
	if n > 0 && c.bpsLimit > 0 {
		expectedDuration := time.Duration(float64(n)/float64(c.bpsLimit)*float64(time.Second))
		elapsed := time.Since(start)
		if expectedDuration > elapsed {
			time.Sleep(expectedDuration - elapsed)
		}
	}
	return n, err
}

// ---------------------------------------------------------
// Core Proxy Handler
// ---------------------------------------------------------

type handler struct {
	user        string
	pass        string
	authEnabled bool
	logger      *log.Logger
	rateLimiter *rateLimiter
}

func (p *handler) dispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}

	clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if !p.rateLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	if p.authEnabled && !p.authenticated(r) {
		w.Header().Set("Proxy-Authenticate", `Basic realm="Advanced Proxy"`)
		http.Error(w, "Proxy authentication required", http.StatusProxyAuthRequired)
		return
	}

	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	p.logger.Printf("%s | %s | %s", clientIP, r.Method, host)

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
	destConn, err := net.DialTimeout("tcp", r.Host, 15*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("Upstream connection failed: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Apply Bandwidth Throttling (e.g. 5 MB/s limit)
	throttledDest := &throttledConn{Conn: destConn, bpsLimit: 5 * 1024 * 1024}

	w.WriteHeader(http.StatusOK)
	hj, ok := w.(http.Hijacker)
	if !ok {
		throttledDest.Close()
		http.Error(w, "Hijacking unsupported", http.StatusInternalServerError)
		return
	}

	srcConn, buf, err := hj.Hijack()
	if err != nil {
		throttledDest.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// WaitGroup ensures we clean up only when both directions close
	var wg sync.WaitGroup
	wg.Add(2)
	
	// Client -> Target
	go func() {
		defer wg.Done()
		if buf != nil && buf.Reader.Buffered() > 0 {
			_, _ = io.Copy(throttledDest, buf)
		}
		_, _ = io.Copy(throttledDest, srcConn)
		if cw, ok := throttledDest.Conn.(interface{ CloseWrite() error }); ok {
			cw.CloseWrite()
		}
	}()
	
	// Target -> Client
	go func() {
		defer wg.Done()
		_, _ = io.Copy(srcConn, throttledDest)
		if cw, ok := srcConn.(interface{ CloseWrite() error }); ok {
			cw.CloseWrite()
		}
	}()
	
	wg.Wait()
	srcConn.Close()
	throttledDest.Close()
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

	// Clone and filter headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	removeHopByHopHeaders(req.Header)
	
	// Privacy Injection: Spoof User-Agent if none exists or it looks suspicious
	ua := req.Header.Get("User-Agent")
	if ua == "" || strings.Contains(ua, "curl") || strings.Contains(ua, "Go-http-client") {
		req.Header.Set("User-Agent", spoofUserAgent())
	}
	
	// Standardize headers to mimic a normal browser request
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	
	// Strip tracking headers
	req.Header.Del("X-Forwarded-For")
	req.Header.Del("Via")

	// Custom HTTP client with timeout and connection pooling
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     false,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Upstream error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	removeHopByHopHeaders(resp.Header)
	copyHeader(w.Header(), resp.Header)
	
	// Add custom tracking header for debugging
	w.Header().Set("X-Proxy-Route", "Advanced-InHouse-V1")
	
	w.WriteHeader(resp.StatusCode)
	
	// Stream the response back
	_, _ = io.Copy(w, resp.Body)
}

func removeHopByHopHeaders(h http.Header) {
	hopHeaders := []string{
		"Proxy-Connection",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Connection",
		"Keep-Alive",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}
	for _, header := range hopHeaders {
		h.Del(header)
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func spoofUserAgent() string {
	uas := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/119.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	}
	return uas[rand.Intn(len(uas))]
}
