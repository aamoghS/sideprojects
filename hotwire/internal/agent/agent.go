package agent

import (
	"context"
	"fmt"
	"io"
	"time"

	"hotwire/internal/client"
	hotwirev1 "hotwire/proto/hotwire/v1"
	"simhttp"
)

type Config struct {
	Name        string
	Control     string
	HTTPAddr    string
	Latency     time.Duration
	Jitter      time.Duration
	ErrorRate   float64
	ReportEvery time.Duration
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.Name == "" {
		return fmt.Errorf("backend name required")
	}
	if cfg.Control == "" {
		cfg.Control = "127.0.0.1:50051"
	}
	if cfg.ReportEvery <= 0 {
		cfg.ReportEvery = 250 * time.Millisecond
	}

	backend, err := simhttp.New(simhttp.Config{
		Name:          cfg.Name,
		Addr:          cfg.HTTPAddr,
		Latency:       simhttp.FromDuration(cfg.Latency),
		Jitter:        simhttp.FromDuration(cfg.Jitter),
		ErrorRate:     cfg.ErrorRate,
		MetricsWindow: simhttp.FromDuration(cfg.ReportEvery),
	})
	if err != nil {
		return err
	}

	errCh := make(chan error, 2)
	go func() { errCh <- backend.Run(ctx) }()
	go func() { errCh <- reportLoop(ctx, cfg, backend) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func reportLoop(ctx context.Context, cfg Config, backend *simhttp.Backend) error {
	conn, cp, err := client.Dial(ctx, cfg.Control)
	if err != nil {
		return err
	}
	defer conn.Close()

	stream, err := client.ReportMetrics(ctx, cp)
	if err != nil {
		return err
	}

	recvDone := make(chan struct{})
	go func() {
		defer close(recvDone)
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(cfg.ReportEvery)
	defer ticker.Stop()

	for {
		snap := backend.Snapshot()
		report := &hotwirev1.MetricReport{
			BackendId:       cfg.Name,
			P50Ms:           snap.P50Ms,
			P99Ms:           snap.P99Ms,
			ErrorRate:       snap.ErrorRate,
			Inflight:        snap.Inflight,
			Rps:             snap.RPS,
			TimestampUnixMs: time.Now().UnixMilli(),
		}
		if err := stream.Send(report); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			_ = stream.CloseSend()
			<-recvDone
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
