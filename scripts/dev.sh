#!/usr/bin/env zsh
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="$ROOT/.env"
DEV_COMPOSE_FILE="$ROOT/compose.dev.yml"
DEV_COMPOSE_PROJECT="${DEV_COMPOSE_PROJECT:-recipe-extractor-local}"

load_env() {
  [[ -f "$ENV_FILE" ]] || return 0
  while IFS= read -r line; do
    line="${line#$'\xef\xbb\xbf'}"  # strip UTF-8 BOM if present
    [[ "$line" =~ ^\s*# ]] && continue
    [[ "$line" =~ ^\s*$ ]] && continue
    [[ "$line" != *=* ]] && continue
    export "${line%%=*}"="${line#*=}"
  done < "$ENV_FILE"
}

wait_for_postgres() {
  local deadline=$(( SECONDS + 60 ))
  while (( SECONDS < deadline )); do
    if docker compose -p "$DEV_COMPOSE_PROJECT" -f "$DEV_COMPOSE_FILE" ps postgres 2>/dev/null | grep -q 'healthy'; then
      return 0
    fi
    sleep 2
  done
  echo 'Postgres did not become healthy in time.' >&2
  return 1
}

load_env

export DEV_HTTP_ADDR="${DEV_HTTP_ADDR:-:8081}"
export DEV_VITE_PORT="${DEV_VITE_PORT:-5174}"
export DEV_POSTGRES_PORT="${DEV_POSTGRES_PORT:-5434}"

export DATABASE_URL="${DEV_DATABASE_URL:-postgres://postgres:postgres@localhost:${DEV_POSTGRES_PORT}/recipes?sslmode=disable}"
echo "Using dev database at ${DATABASE_URL}"

export HTTP_ADDR="$DEV_HTTP_ADDR"
export FRONTEND_DEV_PROXY_URL="http://localhost:${DEV_VITE_PORT}"
export VITE_DEV_PORT="$DEV_VITE_PORT"
export VITE_API_PROXY_TARGET="http://localhost${DEV_HTTP_ADDR}"

echo "Starting isolated dev Postgres container on localhost:${DEV_POSTGRES_PORT}..."
(cd "$ROOT" && docker compose -p "$DEV_COMPOSE_PROJECT" -f "$DEV_COMPOSE_FILE" up -d postgres)

echo 'Waiting for Postgres to become healthy...'
(cd "$ROOT" && wait_for_postgres)

cleanup() {
  echo ''
  echo 'Shutting down...'
  kill "$SERVER_PID" "$WEB_PID" 2>/dev/null
  wait "$SERVER_PID" "$WEB_PID" 2>/dev/null
}
trap cleanup INT TERM

echo 'Starting Go server...'
(cd "$ROOT/server" && go run ./cmd/server) &
SERVER_PID=$!

echo 'Starting Vite dev server...'
(cd "$ROOT/web" && npm run dev) &
WEB_PID=$!

wait "$SERVER_PID" "$WEB_PID"
