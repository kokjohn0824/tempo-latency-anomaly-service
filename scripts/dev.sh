#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
COMPOSE_FILE="$ROOT_DIR/docker/compose.yml"

usage() {
  cat <<EOF
Usage: $(basename "$0") <command>

Commands:
  up         Build and start services (detached)
  down       Stop and remove services
  restart    Restart the service
  logs       Tail service logs
  build      Force rebuild images

Examples:
  $0 up
  $0 logs
EOF
}

cmd=${1:-}
case "$cmd" in
  up)
    docker compose -f "$COMPOSE_FILE" up -d --build
    ;;
  down)
    docker compose -f "$COMPOSE_FILE" down -v
    ;;
  restart)
    docker compose -f "$COMPOSE_FILE" restart service
    ;;
  logs)
    docker compose -f "$COMPOSE_FILE" logs -f --tail=200
    ;;
  build)
    docker compose -f "$COMPOSE_FILE" build --no-cache
    ;;
  *)
    usage
    exit 1
    ;;
esac
