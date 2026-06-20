package control

import (
	"math"
	"sort"
	"time"
)

type ScoredBackend struct {
	BackendID string
	Weight    float64
	Score     float64
	P99Ms     float64
	P50Ms     float64
	ErrorRate float64
	Inflight  int32
	RPS       float64
	LastReport time.Time
	Stale     bool
}

type ScoreResult struct {
	Backends []ScoredBackend
	Reason   string
}

func Score(registry *Registry, now time.Time) ScoreResult {
	agents := registry.Snapshot(now)

	type raw struct {
		agent AgentMetrics
		score float64
		stale bool
	}
	raws := make([]raw, 0, len(agents))
	totalScore := 0.0
	staleCount := 0

	for _, a := range agents {
		stale := registry.IsStale(a, now)
		if stale {
			staleCount++
			raws = append(raws, raw{agent: a, stale: true})
			continue
		}
		p99 := math.Max(a.P99Ms, 1.0)
		errPenalty := 1.0 - math.Min(a.ErrorRate, 1.0)
		score := (1.0 / p99) * errPenalty
		totalScore += score
		raws = append(raws, raw{agent: a, score: score})
	}

	sort.Slice(raws, func(i, j int) bool {
		return raws[i].agent.BackendID < raws[j].agent.BackendID
	})

	out := make([]ScoredBackend, 0, len(raws))
	for _, r := range raws {
		weight := 0.0
		if !r.stale && totalScore > 0 {
			weight = r.score / totalScore
		}
		out = append(out, ScoredBackend{
			BackendID:  r.agent.BackendID,
			Weight:     weight,
			Score:      r.score,
			P99Ms:      r.agent.P99Ms,
			P50Ms:      r.agent.P50Ms,
			ErrorRate:  r.agent.ErrorRate,
			Inflight:   r.agent.Inflight,
			RPS:        r.agent.RPS,
			LastReport: r.agent.LastReport,
			Stale:      r.stale,
		})
	}

	reason := "rebalance"
	if staleCount > 0 {
		reason = "rebalance_with_stale"
	}
	if len(out) == 0 {
		reason = "no_backends"
	}

	return ScoreResult{Backends: out, Reason: reason}
}
