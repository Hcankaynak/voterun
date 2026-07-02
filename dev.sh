#!/usr/bin/env bash
#
# Starts the VoteRun backend (Gin) and frontend (Vite) together.
# Press Ctrl+C to stop both.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"

PIDS=()

# Recursively kill a process and all its descendants. `go run` and `npm run dev`
# each spawn grandchildren (the compiled binary, vite/esbuild) with their own
# PIDs, so killing just the direct child leaves those grandchildren holding the
# ports. Walk the tree children-first to shut everything down cleanly.
kill_tree() {
  local pid=$1 child
  for child in $(pgrep -P "$pid" 2>/dev/null); do
    kill_tree "$child"
  done
  kill -TERM "$pid" 2>/dev/null || true
}

cleanup() {
  # Run once, even though the trap fires for both the signal and EXIT.
  trap - INT TERM EXIT
  echo ""
  echo "Shutting down VoteRun…"
  for pid in "${PIDS[@]}"; do
    kill_tree "$pid"
  done
  wait 2>/dev/null || true
  exit 0
}
trap cleanup INT TERM EXIT

# --- Dependency checks ---
command -v go >/dev/null 2>&1 || { echo "Error: Go is not installed."; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "Error: npm is not installed."; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "Error: Docker is not installed (needed for the dev database)."; exit 1; }

# --- Start development infrastructure (Postgres, Adminer) ---
INFRA_COMPOSE="$ROOT_DIR/docker-compose.dev.yml"
echo "Starting dev infrastructure (Postgres, Adminer)…"
docker compose -f "$INFRA_COMPOSE" up -d --wait

# Point the host-run backend at the containerized Postgres unless overridden.
export DATABASE_URL="${DATABASE_URL:-postgres://voterun:voterun@localhost:5432/voterun?sslmode=disable}"

# Allow any browser origin in local dev so you can reach the app via localhost,
# 127.0.0.1, or a LAN IP without CORS 403s. "*" tells the backend to reflect
# whatever origin calls it (dev only; production sets an explicit CORS_ORIGIN in
# deploy/.env).
export CORS_ORIGIN="${CORS_ORIGIN:-*}"

echo "Preparing backend dependencies…"
(cd "$BACKEND_DIR" && go mod download)

if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
  echo "Installing frontend dependencies…"
  (cd "$FRONTEND_DIR" && npm install)
fi

# --- Start backend ---
echo "Starting backend on http://localhost:${PORT:-8080} …"
(cd "$BACKEND_DIR" && go run .) &
PIDS+=($!)

# --- Start frontend ---
echo "Starting frontend on http://localhost:5173 …"
(cd "$FRONTEND_DIR" && npm run dev) &
PIDS+=($!)

echo ""
echo "VoteRun is running. Open http://localhost:5173 — press Ctrl+C to stop."
echo "Database UI (Adminer): http://localhost:8082"
echo "Dev infra stays up after Ctrl+C. Stop it with:"
echo "  docker compose -f docker-compose.dev.yml down       # keep data"
echo "  docker compose -f docker-compose.dev.yml down -v    # wipe data"

# Wait until any tracked process exits, then let the trap clean up the rest.
# (macOS ships Bash 3.2, which has no `wait -n`, so poll the PIDs instead.)
while :; do
  for pid in "${PIDS[@]}"; do
    kill -0 "$pid" 2>/dev/null || break 2
  done
  sleep 1
done
