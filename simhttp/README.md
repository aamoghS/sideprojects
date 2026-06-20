# simhttp

Fake HTTP backend for load tests. Configurable latency, jitter, and error rate. Tracks p50/p99, error rate, inflight, and RPS.

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
