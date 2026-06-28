// Production HTTP forward proxy for VPS deployment.
// Handles plain HTTP and HTTPS via CONNECT tunneling with required basic auth.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aamoghS/sideprojects/movie/internal/proxy/server"
)

func main() {
	cfg, err := server.ConfigFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := server.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
