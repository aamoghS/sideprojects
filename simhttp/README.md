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

## System design

How a request or a run moves through the code, then the choices behind that shape. In the decision charts, a **solid** box is the route this repo takes and a **dashed** box is the alternative it turns down; the edge label is the tradeoff.

### A dishonest backend

Flags set latency, jitter, and error rate. The library serves HTTP and snapshots p50, p99, error rate, inflight, and RPS. A hotwire backend is a thin wrapper that streams that snapshot.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  flags["Latency, jitter, and error flags"]
  lib["simhttp library"]
  snap["Snapshot"]
  wrap["Hotwire backend wrapper"]
  flags --> lib --> snap --> wrap
  class flags,lib,snap,wrap chosen
```

### A library, not a flag

The fake server is its own module so go test can check latency without gRPC. Hotwire’s score reads p99 and error rate. The other fields stay on the snapshot for a later formula, without a new RPC.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  classDef rejected fill:#f5f5f5,stroke:#9e9e9e,stroke-width:1px,stroke-dasharray:5 5,color:#616161
  q{Where the fake server lives}
  q -->|"tests without a gRPC server"| lib["A library"]
  q -->|"a demo switch inside hotwire"| flag["A demo flag"]
  use{Fields on the snapshot}
  use -->|"the formula today"| used["p99 and error rate"]
  use -->|"present, not read"| later["p50, inflight, and RPS"]
  class lib,used,later chosen
  class flag rejected
```

Longer write-up: [aamogh.vercel.app/projects/simhttp](https://aamogh.vercel.app/projects/simhttp.html)
