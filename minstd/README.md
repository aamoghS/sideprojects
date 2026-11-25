# minstd

Small stdlib shims used by `simhttp`. Each package covers one gap in the standard library — `atomic`, `sync`, `math`, `chrono`, `strings`, `io`, `bufio`, `net`, `strconv`, `errors`, `http` — without trying to be a full replacement.

Started as copy-paste from experiments that needed slightly different HTTP or timing behavior than `net/http` gives you out of the box.

Local monorepo:

```go
replace minstd => ../minstd
```

TCP in `net/` uses syscalls on Windows/Linux. Tests: `go test ./...`.

## System design

How a request or a run moves through the code, then the choices behind that shape. In the decision charts, a **solid** box is the route this repo takes and a **dashed** box is the alternative it turns down; the edge label is the tradeoff.

### One package per gap

http has a request, a response, a status, and a server. net splits Unix and Windows sockets. Each package is a hole from the simhttp work, tested with go test ./... .

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  gap["A hole hit while writing simhttp"]
  pkg["One package"]
  test["go test ./..."]
  gap --> pkg --> test
  class gap,pkg,test chosen
```

### A shim, not a port

The packages are not drop-in replacements for the standard library. A shim that almost matches is worse than no shim, because the missing method is the one you call. These tests cover what the packages claim.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  classDef rejected fill:#f5f5f5,stroke:#9e9e9e,stroke-width:1px,stroke-dasharray:5 5,color:#616161
  q{How much of std to copy}
  q -->|"the piece the HTTP stack needed"| shim["One shim"]
  q -->|"a second standard library"| port["A full port"]
  class shim chosen
  class port rejected
```

Longer write-up: [aamogh.vercel.app/projects/minstd](https://aamogh.vercel.app/projects/minstd.html)
