package control_test

import (
	"testing"
	"time"

	"github.com/aamoghS/sideprojects/hotwire/internal/control"
)

func TestRegistryStaleDetection(t *testing.T) {
	reg := control.NewRegistry(3 * time.Second)
	now := time.Unix(1000, 0)

	reg.Update(control.AgentMetrics{
		BackendID:  "a",
		P99Ms:      10,
		LastReport: now.Add(-2 * time.Second),
	})
	reg.Update(control.AgentMetrics{
		BackendID:  "b",
		P99Ms:      10,
		LastReport: now.Add(-4 * time.Second),
	})

	byID := map[string]control.AgentMetrics{}
	for _, a := range reg.Snapshot(now) {
		byID[a.BackendID] = a
	}
	if reg.IsStale(byID["a"], now) {
		t.Fatal("expected backend a to be fresh")
	}
	if !reg.IsStale(byID["b"], now) {
		t.Fatal("expected backend b to be stale")
	}

	reg.Remove("a")
	if len(reg.Snapshot(now)) != 1 {
		t.Fatalf("expected one backend after remove, got %d", len(reg.Snapshot(now)))
	}
}

func TestScoreWeightsFavorLowLatency(t *testing.T) {
	reg := control.NewRegistry(3 * time.Second)
	now := time.Unix(2000, 0)

	reg.Update(control.AgentMetrics{BackendID: "fast", P99Ms: 10, ErrorRate: 0, LastReport: now})
	reg.Update(control.AgentMetrics{BackendID: "slow", P99Ms: 100, ErrorRate: 0, LastReport: now})

	result := control.Score(reg, now)
	weights := map[string]float64{}
	for _, b := range result.Backends {
		weights[b.BackendID] = b.Weight
	}

	if weights["fast"] <= weights["slow"] {
		t.Fatalf("fast=%f slow=%f, expected fast weight > slow", weights["fast"], weights["slow"])
	}

	sum := weights["fast"] + weights["slow"]
	if sum < 0.999 || sum > 1.001 {
		t.Fatalf("weights sum=%f, want 1.0", sum)
	}
}

func TestScoreStaleBackendGetsZeroWeight(t *testing.T) {
	reg := control.NewRegistry(3 * time.Second)
	now := time.Unix(3000, 0)

	reg.Update(control.AgentMetrics{BackendID: "live", P99Ms: 20, LastReport: now})
	reg.Update(control.AgentMetrics{BackendID: "dead", P99Ms: 20, LastReport: now.Add(-10 * time.Second)})

	result := control.Score(reg, now)
	for _, b := range result.Backends {
		if b.BackendID == "dead" && b.Weight != 0 {
			t.Fatalf("dead weight=%f, want 0", b.Weight)
		}
		if b.BackendID == "dead" && !b.Stale {
			t.Fatal("dead backend should be marked stale")
		}
	}
}

func TestScoreErrorRatePenalizesBackend(t *testing.T) {
	reg := control.NewRegistry(3 * time.Second)
	now := time.Unix(4000, 0)

	reg.Update(control.AgentMetrics{BackendID: "clean", P99Ms: 30, ErrorRate: 0, LastReport: now})
	reg.Update(control.AgentMetrics{BackendID: "noisy", P99Ms: 30, ErrorRate: 0.5, LastReport: now})

	result := control.Score(reg, now)
	weights := map[string]float64{}
	for _, b := range result.Backends {
		weights[b.BackendID] = b.Weight
	}
	if weights["clean"] <= weights["noisy"] {
		t.Fatalf("clean=%f noisy=%f, expected clean weight > noisy", weights["clean"], weights["noisy"])
	}
}
