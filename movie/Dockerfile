FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /movie-finder ./cmd/movie-finder

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /movie-finder /app/movie-finder
COPY config/ ./config/

ENTRYPOINT ["/app/movie-finder"]
CMD []
