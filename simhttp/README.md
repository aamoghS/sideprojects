# simhttp

Fake HTTP backend for load tests. You configure latency, jitter, and error rate per instance; it serves a simple handler and tracks p50/p99, error rate, inflight requests, and RPS. `hotwire` backends wrap this and stream the snapshots to the control plane.

Depends on `minstd` for HTTP and timing primitives.

```go
backend, err := simhttp.New(simhttp.Config{
    Name:      "alpha",
    Addr:      "127.0.0.1:8081",
    Latency:   simhttp.FromDuration(20 * time.Millisecond),
    Jitter:    simhttp.FromDuration(5 * time.Millisecond),
    ErrorRate: 0.01,
})
snap := backend.Snapshot()
err = backend.Run(ctx)
```

Monorepo:

```go
replace simhttp => ../simhttp
replace minstd => ../minstd
```
