package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func Transport(proxyURL string, workers int) (*http.Transport, error) {
	baseDialer := &net.Dialer{
		Timeout:   4 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext:           baseDialer.DialContext,
		MaxIdleConns:          workers * 2,
		MaxIdleConnsPerHost:   workers,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if proxyURL == "" {
		return transport, nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(u.Scheme, "socks5") {
		dialer, err := socks5Dialer(u, baseDialer)
		if err != nil {
			return nil, err
		}
		transport.DialContext = dialer.DialContext
		return transport, nil
	}

	transport.Proxy = http.ProxyURL(u)
	return transport, nil
}

func Mask(raw string) string {
	if raw == "" {
		return "direct"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.User != nil {
		u.User = url.UserPassword("***", "***")
	}
	return u.String()
}

func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid proxy URL %q: %w", Mask(raw), err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "socks5":
	default:
		return fmt.Errorf("unsupported proxy scheme %q (use http, https, or socks5)", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("proxy URL missing host: %s", Mask(raw))
	}
	return nil
}
