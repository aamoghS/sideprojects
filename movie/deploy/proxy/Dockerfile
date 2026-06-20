FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY internal/proxy/server ./internal/proxy/server
COPY cmd/edgeproxy ./cmd/edgeproxy

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /edgeproxy ./cmd/edgeproxy

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /edgeproxy /app/edgeproxy

EXPOSE 8888

ENV PROXY_ADDR=0.0.0.0:8888

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8888/health || exit 1

ENTRYPOINT ["/app/edgeproxy"]
