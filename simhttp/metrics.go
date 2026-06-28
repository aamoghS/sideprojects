package simhttp

import (
	"github.com/aamoghS/sideprojects/minstd/atomic"
	"github.com/aamoghS/sideprojects/minstd/chrono"
	"github.com/aamoghS/sideprojects/minstd/math"
	"github.com/aamoghS/sideprojects/minstd/sync"
)

const maxLatencySamples = 256

type Snapshot struct {
	P50Ms     float64
	P99Ms     float64
	ErrorRate float64
	Inflight  int32
	RPS       float64
}

type Metrics struct {
	baseLatencyNs atomic.Int64
	window        chrono.Duration

	mu             sync.Mutex
	latencySamples []float64
	inflight       int32
	requests       uint64
	errors         uint64
}

func NewMetrics(baseLatency, window chrono.Duration) *Metrics {
	m := &Metrics{window: window}
	m.baseLatencyNs.Store(baseLatency.Nanoseconds())
	return m
}

func (m *Metrics) BeginRequest() func(recordMs float64, failed bool) {
	atomic.AddInt32(&m.inflight, 1)
	return func(recordMs float64, failed bool) {
		atomic.AddUint64(&m.requests, 1)
		if failed {
			atomic.AddUint64(&m.errors, 1)
		}
		m.recordLatency(recordMs)
		atomic.AddInt32(&m.inflight, -1)
	}
}

func (m *Metrics) recordLatency(ms float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latencySamples = append(m.latencySamples, ms)
	if len(m.latencySamples) > maxLatencySamples {
		m.latencySamples = m.latencySamples[len(m.latencySamples)-maxLatencySamples:]
	}
}

func (m *Metrics) Snapshot() Snapshot {
	m.mu.Lock()
	samples := append([]float64(nil), m.latencySamples...)
	m.mu.Unlock()

	s := Snapshot{Inflight: atomic.LoadInt32(&m.inflight)}
	reqs := atomic.LoadUint64(&m.requests)
	errs := atomic.LoadUint64(&m.errors)
	if reqs > 0 {
		s.ErrorRate = float64(errs) / float64(reqs)
	}
	if len(samples) == 0 {
		s.P50Ms = float64(m.baseLatencyNs.Load()) / float64(chrono.Millisecond)
		s.P99Ms = s.P50Ms * 1.5
	} else {
		s.P50Ms = Percentile(samples, 0.50)
		s.P99Ms = Percentile(samples, 0.99)
	}
	windowSec := m.window.Seconds()
	if windowSec > 0 {
		s.RPS = float64(len(samples)) / windowSec
	}
	return s
}

func Percentile(samples []float64, p float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	cp := append([]float64(nil), samples...)
	for i := 0; i < len(cp); i++ {
		minIdx := i
		for j := i + 1; j < len(cp); j++ {
			if cp[j] < cp[minIdx] {
				minIdx = j
			}
		}
		cp[i], cp[minIdx] = cp[minIdx], cp[i]
	}
	idx := math.Ceil(p*float64(len(cp))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}
