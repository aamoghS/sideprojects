# minstd

Small stdlib shims used by `simhttp`. Each package covers one gap; nothing here is meant to be a full replacement.

Packages: `atomic`, `sync`, `math`, `chrono`, `strings`, `io`, `bufio`, `net`, `strconv`, `errors`, `http`.

Local monorepo:

```go
replace minstd => ../minstd
```

TCP in `net/` uses syscalls on Windows/Linux. Tests: `go test ./...`.
