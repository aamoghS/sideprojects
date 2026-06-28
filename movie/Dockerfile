FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /edgeproxy ./cmd/edgeproxy
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /localproxy ./cmd/localproxy
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /movie-ctl ./cmd/movie-ctl
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /movie-finder ./cmd/movie-finder
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /orchestrator ./cmd/orchestrator

FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /edgeproxy /app/edgeproxy
COPY --from=builder /localproxy /app/localproxy
COPY --from=builder /movie-ctl /app/movie-ctl
COPY --from=builder /movie-finder /app/movie-finder
COPY --from=builder /orchestrator /app/orchestrator
COPY config/ ./config/

ENV PATH="/app:${PATH}"

# Default command can be anything, but we usually override it in Kubernetes.
CMD ["/app/movie-finder"]
