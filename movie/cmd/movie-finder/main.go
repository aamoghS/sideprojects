package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aamoghS/sideprojects/movie/internal/app"
)

func main() {
	opts := app.ParseFlags()

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	if err := app.Run(ctx, opts); err != nil {
		if opts.TestProxies {
			fmt.Fprintf(os.Stderr, "Proxy test failed: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
