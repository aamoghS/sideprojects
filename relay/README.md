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

## System design

How a request or a run moves through the code, then the choices behind that shape. In the decision charts, a **solid** box is the route this repo takes and a **dashed** box is the alternative it turns down; the edge label is the tradeoff.

### Three RPC shapes

Publish is unary. Watch is a server stream. Chat is bidirectional. The server keeps topics in memory. There is no disk.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  cli[CLI]
  server["In-memory server"]
  once["Publish, unary"]
  watch["Watch, server stream"]
  chat["Chat, bidirectional"]
  cli --> server
  server --> once
  server --> watch
  server --> chat
  class cli,server,once,watch,chat chosen
```

### Streams without a score

Hotwire uses the same three RPC shapes under a scorer. Relay is those streams on their own. Restarting the server drops the topics. A second node is a different program.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  classDef rejected fill:#f5f5f5,stroke:#9e9e9e,stroke-width:1px,stroke-dasharray:5 5,color:#616161
  q{What is being learned}
  q -->|"a broken stream is just a broken stream"| relay["gRPC pub sub"]
  q -->|"the pattern stays buried under weights"| buried["Only inside hotwire"]
  store{Where rooms live}
  store -->|"one process"| mem["Memory"]
  store -->|"restart keeps the topics"| disk["Disk and a second node"]
  class relay,mem chosen
  class buried,disk rejected
```

Longer write-up: [aamogh.vercel.app/projects/relay](https://aamogh.vercel.app/projects/relay.html)
