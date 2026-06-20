package proxy

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

func socks5Dialer(proxyURL *url.URL, base *net.Dialer) (proxy.ContextDialer, error) {
	var auth *proxy.Auth
	if proxyURL.User != nil {
		password, _ := proxyURL.User.Password()
		auth = &proxy.Auth{
			User:     proxyURL.User.Username(),
			Password: password,
		}
	}

	host := proxyURL.Host
	if !strings.Contains(host, ":") {
		host += ":1080"
	}

	dialer, err := proxy.SOCKS5("tcp", host, auth, base)
	if err != nil {
		return nil, fmt.Errorf("socks5 dialer: %w", err)
	}

	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return &contextDialerAdapter{dialer: dialer, timeout: base.Timeout}, nil
	}
	return contextDialer, nil
}

type contextDialerAdapter struct {
	dialer  proxy.Dialer
	timeout time.Duration
}

func (a *contextDialerAdapter) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	timeout := a.timeout
	if timeout <= 0 {
		timeout = 4 * time.Second
	}

	type result struct {
		conn net.Conn
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		conn, err := a.dialer.Dial(network, addr)
		ch <- result{conn, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("socks5 dial timed out after %s", timeout)
	case r := <-ch:
		return r.conn, r.err
	}
}
