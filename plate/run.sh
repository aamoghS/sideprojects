#!/usr/bin/env bash
# plate — run this from the project root every time
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

if [[ -f "$ROOT/plate.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/plate.env"
  set +a
fi

PLATE_PROVIDER="${PLATE_PROVIDER:-docker}"
PLATE_LISTEN="${PLATE_LISTEN:-:8080}"
PLATE_DATA="${PLATE_DATA:-.plate}"
PLATE_API="${PLATE_API:-http://127.0.0.1:8080}"
PLATE_DOCKER_IMAGE="${PLATE_DOCKER_IMAGE:-ubuntu:22.04}"
BIN="$ROOT/plate"

build() {
  go build -o "$BIN" ./cmd/plate
}

usage() {
  cat <<EOF
Usage: ./run.sh [command] [options]

Commands:
  serve          build + start control plane (default)
  build          compile ./plate binary only
  test           go test ./...
  plans          list plans from running API
  list           list VMs
  create         create VM (--name required)
  start          start VM (--id required)
  stop           stop VM (--id required)
  delete         delete VM (--id required)
  help           show this help

Config (env or plate.env):
  PLATE_PROVIDER       docker | proxmox   (default: docker)
  PLATE_LISTEN         e.g. :8080
  PLATE_DATA             state dir
  PLATE_API              CLI target URL
  PLATE_DOCKER_IMAGE     default container image
  PLATE_PROXMOX_*        see README for proxmox vars

Examples:
  ./run.sh
  ./run.sh serve
  ./run.sh create --name web-1 --plan medium
  ./run.sh list
  PLATE_PROVIDER=proxmox ./run.sh serve
EOF
}

CMD="${1:-serve}"
if [[ "$CMD" == "help" || "$CMD" == "-h" || "$CMD" == "--help" ]]; then
  usage
  exit 0
fi
shift || true

case "$CMD" in
  build)
    build
    echo "built $BIN"
    ;;
  test)
    go test ./...
    ;;
  serve)
    build
    exec "$BIN" serve \
      --provider "$PLATE_PROVIDER" \
      --listen "$PLATE_LISTEN" \
      --data "$PLATE_DATA" \
      --docker-image "$PLATE_DOCKER_IMAGE" \
      "$@"
    ;;
  plans|list|create|start|stop|delete)
    build
    exec "$BIN" "$CMD" --api "$PLATE_API" "$@"
    ;;
  *)
    build
    exec "$BIN" "$CMD" "$@"
    ;;
esac
