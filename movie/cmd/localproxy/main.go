// Local HTTP forward proxy for development and movie-finder scraping.
// Handles plain HTTP requests and HTTPS via CONNECT tunneling.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"movie/internal/proxy/server"
)

func main() {
	addr := flag.String("addr", envOr("LOCAL_PROXY_ADDR", "127.0.0.1:8888"), "Listen address (host:port)")
	user := flag.String("user", os.Getenv("LOCAL_PROXY_USER"), "Basic auth username (empty = disabled)")
	pass := flag.String("pass", os.Getenv("LOCAL_PROXY_PASS"), "Basic auth password")
	flag.Parse()

	cfg, err := server.LocalConfig(*addr, *user, *pass)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := server.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
