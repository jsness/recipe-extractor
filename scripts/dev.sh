#!/usr/bin/env zsh
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="$ROOT/.env"

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
    if docker compose ps postgres 2>/dev/null | grep -q 'healthy'; then
      return 0
    fi
    sleep 2
  done
  echo 'Postgres did not become healthy in time.' >&2
  return 1
}

load_env

export DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5433/recipes?sslmode=disable}"
if [[ -z "${DATABASE_URL+set}" ]] || [[ "$DATABASE_URL" == "postgres://postgres:postgres@localhost:5433/recipes?sslmode=disable" ]]; then
  echo 'DATABASE_URL not set, defaulting to localhost:5433/recipes'
fi

export FRONTEND_DEV_PROXY_URL='http://localhost:5173'

echo 'Stopping app container if it is already running...'
(cd "$ROOT" && docker compose stop app) > /dev/null

echo 'Starting Postgres container...'
(cd "$ROOT" && docker compose up -d postgres)

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
