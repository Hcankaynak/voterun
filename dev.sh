#!/usr/bin/env bash
#
# Starts the VoteRun backend (Gin) and frontend (Vite) together.
# Press Ctrl+C to stop both.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"

PIDS=()

cleanup() {
  echo ""
  echo "Shutting down VoteRun…"
  for pid in "${PIDS[@]}"; do
    # Kill the whole process group so child processes exit too.
    kill -- -"$pid" 2>/dev/null || kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup INT TERM EXIT

# --- Dependency checks ---
command -v go >/dev/null 2>&1 || { echo "Error: Go is not installed."; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "Error: npm is not installed."; exit 1; }

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

# Wait for any process to exit, then cleanup runs via the trap.
wait -n
