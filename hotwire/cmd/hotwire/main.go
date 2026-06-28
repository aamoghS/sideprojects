package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aamoghS/sideprojects/hotwire/internal/agent"
	"github.com/aamoghS/sideprojects/hotwire/internal/client"
	"github.com/aamoghS/sideprojects/hotwire/internal/control"
	"github.com/aamoghS/sideprojects/hotwire/internal/demo"
	"github.com/aamoghS/sideprojects/hotwire/internal/proxy"
	hotwirev1 "github.com/aamoghS/sideprojects/hotwire/proto/hotwire/v1"

	"google.golang.org/grpc"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "serve":
		err = cmdServe(os.Args[2:])
	case "backend":
		err = cmdBackend(os.Args[2:])
	case "watch":
		err = cmdWatch(os.Args[2:])
	case "proxy":
		err = cmdProxy(os.Args[2:])
	case "demo":
		err = cmdDemo(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`hotwire — live adaptive traffic steering control plane

usage:
  hotwire serve [--addr 127.0.0.1:50051] [--rebalance-ms 500]
  hotwire backend <name> [--control 127.0.0.1:50051] [--http :8081]
                  [--latency 20ms] [--jitter 5ms] [--error-rate 0.01]
  hotwire watch [--control 127.0.0.1:50051]
  hotwire proxy [--control 127.0.0.1:50051] [--port 8080]
                [--route name=http://127.0.0.1:8081] (repeatable)
  hotwire demo [--control 127.0.0.1:50051] [--proxy-port 8080]`)
}

func cmdServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:50051", "gRPC listen address")
	rebalanceMS := fs.Int("rebalance-ms", 500, "rebalance interval in milliseconds")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := signalContext()
	defer cancel()

	srv := control.NewServer(time.Duration(*rebalanceMS) * time.Millisecond)
	grpcSrv := grpc.NewServer()
	hotwirev1.RegisterControlPlaneServer(grpcSrv, srv)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}

	go srv.RunRebalanceLoop(ctx)
	fmt.Printf("control plane listening on %s (rebalance %dms)\n", *addr, *rebalanceMS)
	return grpcSrv.Serve(lis)
}

func cmdBackend(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("backend name required")
	}
	name := args[0]

	fs := flag.NewFlagSet("backend", flag.ContinueOnError)
	controlAddr := fs.String("control", "127.0.0.1:50051", "control plane address")
	httpAddr := fs.String("http", ":0", "fake backend HTTP listen address")
	latency := fs.Duration("latency", 20*time.Millisecond, "base latency")
	jitter := fs.Duration("jitter", 5*time.Millisecond, "latency jitter")
	errorRate := fs.Float64("error-rate", 0.01, "injected error rate 0.0-1.0")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	ctx, cancel := signalContext()
	defer cancel()

	return agent.Run(ctx, agent.Config{
		Name:        name,
		Control:     *controlAddr,
		HTTPAddr:    *httpAddr,
		Latency:     *latency,
		Jitter:      *jitter,
		ErrorRate:   *errorRate,
		ReportEvery: 250 * time.Millisecond,
	})
}

func cmdWatch(args []string) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	controlAddr := fs.String("control", "127.0.0.1:50051", "control plane address")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := signalContext()
	defer cancel()

	conn, cp, err := client.Dial(ctx, *controlAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	stream, err := client.SubscribeWeights(ctx, cp)
	if err != nil {
		return err
	}

	updates := make(chan *hotwirev1.WeightUpdate, 8)
	go func() {
		for {
			update, err := stream.Recv()
			if err != nil {
				return
			}
			updates <- update
		}
	}()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	var last *hotwirev1.WeightUpdate
	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			last = update
		case <-ticker.C:
			if last != nil {
				demo.RenderTable(last)
			}
		}
	}
}

func cmdProxy(args []string) error {
	fs := flag.NewFlagSet("proxy", flag.ContinueOnError)
	controlAddr := fs.String("control", "127.0.0.1:50051", "control plane address")
	port := fs.Int("port", 8080, "HTTP listen port")
	routes := routeFlag{}
	fs.Var(&routes, "route", "backend route name=url (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(routes) == 0 {
		return fmt.Errorf("at least one --route name=url required")
	}

	ctx, cancel := signalContext()
	defer cancel()

	p := proxy.New(*controlAddr, fmt.Sprintf("127.0.0.1:%d", *port), routes)
	return p.Run(ctx)
}

type routeFlag []proxy.Route

func (r *routeFlag) String() string {
	parts := make([]string, len(*r))
	for i, route := range *r {
		parts[i] = route.BackendID + "=" + route.URL
	}
	return strings.Join(parts, ",")
}

func (r *routeFlag) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid route %q, want name=url", value)
	}
	*r = append(*r, proxy.Route{BackendID: parts[0], URL: parts[1]})
	return nil
}

func cmdDemo(args []string) error {
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)
	controlAddr := fs.String("control", "127.0.0.1:50051", "control plane address")
	proxyPort := fs.Int("proxy-port", 8080, "demo proxy port")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := signalContext()
	defer cancel()

	return demo.Run(ctx, demo.Options{
		ControlAddr: *controlAddr,
		ProxyPort:   *proxyPort,
	})
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		select {
		case <-ch:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
