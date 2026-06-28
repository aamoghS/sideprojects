package simhttp

import (
	"testing"

	"github.com/aamoghS/sideprojects/minstd/chrono"
)

func TestPercentile(t *testing.T) {
	samples := []float64{10, 20, 30, 40, 100}
	if got := Percentile(samples, 0.50); got != 30 {
		t.Fatalf("p50 = %v, want 30", got)
	}
	if got := Percentile(samples, 0.99); got != 100 {
		t.Fatalf("p99 = %v, want 100", got)
	}
}

func TestMetricsSnapshotEmpty(t *testing.T) {
	m := NewMetrics(20*chrono.Millisecond, 250*chrono.Millisecond)
	s := m.Snapshot()
	if s.P50Ms != 20 {
		t.Fatalf("p50 = %v, want 20", s.P50Ms)
	}
	if s.P99Ms != 30 {
		t.Fatalf("p99 = %v, want 30", s.P99Ms)
	}
	if s.ErrorRate != 0 || s.Inflight != 0 || s.RPS != 0 {
		t.Fatalf("unexpected snapshot: %+v", s)
	}
}
