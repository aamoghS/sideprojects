# relay

gRPC pub/sub CLI. Three modes: fire-and-forget publish to a topic, server-streaming watch on a topic, and bidirectional room chat where everyone in a room sees each other's messages.

The server keeps an in-memory store — fine for local dev and demos, not for production persistence.

## Build

```bash
cd relay
go mod tidy
go build -o relay.exe ./cmd/relay
```

## Usage

```bash
./relay.exe serve
./relay.exe watch logs
./relay.exe publish logs "hello"
./relay.exe room hackathon --name alice
```

Proto regen:

```bash
protoc --go_out=. --go_opt=module=github.com/aamoghS/sideprojects/relay \
  --go-grpc_out=. --go-grpc_opt=module=github.com/aamoghS/sideprojects/relay \
  proto/relay/v1/relay.proto
```

Tests: `go test ./...`
