package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type state struct {
	Tokens float64 `json:"tokens"`
	At     int64   `json:"at_unix_nano"`
}

type Bucket struct {
	mu     sync.Mutex
	rate   float64 // tokens per second
	burst  float64
	file   string
	tokens float64
	last   time.Time
}

func Open(path string, perMinute float64, burst float64) (*Bucket, error) {
	b := &Bucket{
		rate:  perMinute / 60,
		burst: burst,
		file:  path,
	}
	if err := b.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if b.last.IsZero() {
		b.tokens = burst
		b.last = time.Now()
	}
	return b, nil
}

func (b *Bucket) Take(n float64) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.last = now

	if b.tokens >= n {
		b.tokens -= n
		_ = b.saveLocked()
		return 0
	}

	need := n - b.tokens
	wait := time.Duration(need/b.rate*1e9) * time.Nanosecond
	b.tokens = 0
	b.last = now.Add(wait)
	_ = b.saveLocked()
	return wait
}

func (b *Bucket) load() error {
	raw, err := os.ReadFile(b.file)
	if err != nil {
		return err
	}
	var s state
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	b.tokens = s.Tokens
	b.last = time.Unix(0, s.At)
	return nil
}

func (b *Bucket) saveLocked() error {
	if b.file == "" {
		return nil
	}
	s := state{Tokens: b.tokens, At: b.last.UnixNano()}
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp := b.file + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, b.file)
}

func (b *Bucket) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return fmt.Sprintf("%.1f / %.0f tokens", b.tokens, b.burst)
}
