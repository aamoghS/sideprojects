# relay

gRPC pub/sub CLI: publish to topics, watch streams, or join bidirectional chat rooms.

```bash
cd relay
go mod tidy
go build -o relay.exe ./cmd/relay

./relay.exe serve
./relay.exe watch logs
./relay.exe publish logs "hello"
./relay.exe room hackathon --name alice
```

Proto regen:

```bash
protoc --go_out=. --go_opt=module=relay \
  --go-grpc_out=. --go-grpc_opt=module=relay \
  proto/relay/v1/relay.proto
```

Tests: `go test ./...`
