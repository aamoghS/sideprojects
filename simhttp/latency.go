package simhttp

import (
	"math/rand"

	"minstd/atomic"
	"minstd/chrono"
)

type Latency struct {
	base   atomic.Int64
	jitter atomic.Int64
}

func NewLatency(base, jitter chrono.Duration) *Latency {
	l := &Latency{}
	l.base.Store(base.Nanoseconds())
	l.jitter.Store(jitter.Nanoseconds())
	return l
}

func (l *Latency) Set(base, jitter chrono.Duration) {
	l.base.Store(base.Nanoseconds())
	l.jitter.Store(jitter.Nanoseconds())
}

func (l *Latency) Sample() chrono.Duration {
	base := chrono.Duration(l.base.Load())
	jitter := chrono.Duration(l.jitter.Load())
	if jitter <= 0 {
		return base
	}
	delta := chrono.Duration(rand.Int63n(int64(jitter)*2+1)) - jitter
	if delta < 0 && base < -delta {
		return 0
	}
	return base + delta
}
