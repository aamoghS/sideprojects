package demo

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aamoghS/sideprojects/hotwire/internal/agent"
	"github.com/aamoghS/sideprojects/hotwire/internal/client"
	"github.com/aamoghS/sideprojects/hotwire/internal/control"
	"github.com/aamoghS/sideprojects/hotwire/internal/proxy"
	hotwirev1 "github.com/aamoghS/sideprojects/hotwire/proto/hotwire/v1"

	"google.golang.org/grpc"
)

type Options struct {
	ControlAddr string
	ProxyPort   int
}

func Run(ctx context.Context, opts Options) error {
	if opts.ControlAddr == "" {
		opts.ControlAddr = "127.0.0.1:50051"
	}
	if opts.ProxyPort == 0 {
		opts.ProxyPort = 8080
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		cancel()
	}()

	srv := control.NewServer(500 * time.Millisecond)
	grpcSrv := grpc.NewServer()
	hotwirev1.RegisterControlPlaneServer(grpcSrv, srv)

	lis, err := net.Listen("tcp", opts.ControlAddr)
	if err != nil {
		return err
	}
	go srv.RunRebalanceLoop(ctx)
	go func() {
		_ = grpcSrv.Serve(lis)
	}()
	defer grpcSrv.Stop()

	type spec struct {
		name    string
		latency time.Duration
		jitter  time.Duration
	}
	backends := []spec{
		{name: "alpha", latency: 15 * time.Millisecond, jitter: 3 * time.Millisecond},
		{name: "beta", latency: 45 * time.Millisecond, jitter: 8 * time.Millisecond},
		{name: "gamma", latency: 120 * time.Millisecond, jitter: 15 * time.Millisecond},
	}

	routes := make([]proxy.Route, 0, len(backends))
	for i, b := range backends {
		port := 18081 + i
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		routes = append(routes, proxy.Route{
			BackendID: b.name,
			URL:       fmt.Sprintf("http://%s", addr),
		})
		go func(b spec, listenAddr string) {
			_ = agent.Run(ctx, agent.Config{
				Name:        b.name,
				Control:     opts.ControlAddr,
				HTTPAddr:    listenAddr,
				Latency:     b.latency,
				Jitter:      b.jitter,
				ErrorRate:   0.005,
				ReportEvery: 200 * time.Millisecond,
			})
		}(b, addr)
	}

	go func() {
		p := proxy.New(opts.ControlAddr, fmt.Sprintf("127.0.0.1:%d", opts.ProxyPort), routes)
		_ = p.Run(ctx)
	}()

	time.Sleep(900 * time.Millisecond)

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", opts.ProxyPort)
	go loadGenerator(ctx, proxyURL)

	conn, cp, err := client.Dial(ctx, opts.ControlAddr)
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

	fmt.Println("hotwire demo — live adaptive load steering")
	fmt.Printf("control %s  proxy %s  traffic -> proxy\n", opts.ControlAddr, proxyURL)
	fmt.Println("backends: alpha ~15ms, beta ~45ms, gamma ~120ms")
	fmt.Println("try: hotwire backend gamma --latency 200ms (in another terminal) to watch weights collapse")
	fmt.Println("press Ctrl+C to exit")
	fmt.Println()

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
				RenderTable(last)
			}
		}
	}
}

func RenderTable(update *hotwirev1.WeightUpdate) {
	fmt.Print("\033[H\033[2J")
	ts := time.UnixMilli(update.GetTimestampUnixMs()).Format("15:04:05.000")
	fmt.Printf("hotwire weights  reason=%s  updated=%s\n\n", update.GetReason(), ts)
	fmt.Printf("%-8s %8s %8s %8s %s\n", "backend", "p99_ms", "err", "weight", "bar")
	fmt.Println(strings.Repeat("-", 56))
	for _, w := range update.GetWeights() {
		bar := weightBar(w.GetWeight(), 24)
		fmt.Printf("%-8s %8.1f %7.2f%% %8.3f %s\n",
			w.GetBackendId(),
			w.GetP99Ms(),
			w.GetErrorRate()*100,
			w.GetWeight(),
			bar,
		)
	}
}

func weightBar(weight float64, width int) string {
	if width <= 0 {
		return ""
	}
	filled := int(weight * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func loadGenerator(ctx context.Context, proxyURL string) {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			go func() {
				resp, err := client.Get(proxyURL)
				if err != nil {
					return
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}()
		}
	}
}
