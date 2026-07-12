# minstd

Small stdlib shims used by `simhttp`. Each package covers one gap in the standard library — `atomic`, `sync`, `math`, `chrono`, `strings`, `io`, `bufio`, `net`, `strconv`, `errors`, `http` — without trying to be a full replacement.

Started as copy-paste from experiments that needed slightly different HTTP or timing behavior than `net/http` gives you out of the box.

Local monorepo:

```go
replace minstd => ../minstd
```

TCP in `net/` uses syscalls on Windows/Linux. Tests: `go test ./...`.
