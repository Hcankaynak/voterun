# VoteRun load test (k6)

Simulates ~150 concurrent VoteRun participants against a single shared board.

## What it does

- **Setup (once):** a single facilitator account registers (the only registered
  user), creates one board, and seeds a few cards.
- **150 virtual users (anonymous):** each one "opens the link"
  (`GET /api/boards/:id` + WebSocket), submits a few answer cards, and a subset
  (`VOTE_RATIO`) upvotes random cards. Board APIs are unauthenticated, so the
  VUs never register or log in.
- Measures HTTP latency per action, WebSocket connect time, message throughput,
  and approximate write -> broadcast propagation latency.

## Install k6

```bash
# macOS
brew install k6
```

Other platforms: https://grafana.com/docs/k6/latest/set-up/install-k6/

## Run

Always start with a local dry run, then a prod smoke test, then the full run.

```bash
# 1. Local dry run (start backend via root docker-compose first; backend on :8081)
SMOKE=1 k6 run -e BASE_URL=http://localhost:8081 -e WS_URL=ws://localhost:8081 loadtest/script.js

# 2. Prod smoke test (5 VUs)
SMOKE=1 k6 run loadtest/script.js

# 3. Prod smoke test with 30 VUs
SMOKE=1 SMOKE_VUS=30 k6 run loadtest/script.js

# 4. Full prod run (150 VUs, default target https://api.voterun.app)
k6 run loadtest/script.js
```

## Environment variables

| Var | Default | Purpose |
|-----|---------|---------|
| `BASE_URL` | `https://api.voterun.app` | REST base URL |
| `WS_URL` | derived from `BASE_URL` (`http`->`ws`) | WebSocket base URL |
| `VUS` | `150` | Concurrent virtual users (full run) |
| `SMOKE` | unset | `1` -> small smoke profile |
| `SMOKE_VUS` | `5` | Concurrent virtual users in smoke profile |
| `SESSION` | `120` (`30` in smoke) | Seconds each VU holds a session before reconnecting |
| `CARDS_PER_USER` | `3` | Answer cards each user submits |
| `VOTE_RATIO` | `0.55` | Fraction of users that also upvote |
| `SEED_CARDS` | `3` | Cards created in setup so early upvotes have a target |
| `ACTION_INTERVAL` | `4000` | Base ms between a user's actions |
| `ACTION_JITTER` | `2000` | Random extra ms added per user |
| `CLEANUP` | `1` | Delete all board cards in teardown (`0` to keep) |

Examples:

```bash
k6 run -e VUS=200 -e CARDS_PER_USER=5 -e VOTE_RATIO=0.7 loadtest/script.js
k6 run -e BASE_URL=https://new.voterun.app -e WS_URL=wss://new.voterun.app loadtest/script.js
```

## Reading the results

A JSON report is written to `loadtest/main-summary.json`; a summary also prints to
the console. Key metrics:

- `http_req_duration` (p95/p99) — REST latency. Filter by action tag
  (`open`, `submit`, `vote`) to see which calls are slow.
- `http_req_failed` — error rate; threshold fails the run above 2%.
- `ws_connect_time` — time to establish each WebSocket.
- `ws_propagation_ms` — approximate time from a mutation to the next board
  snapshot received over WebSocket (the live-sync latency users feel).
- `ws_messages_received` / `ws_sessions` / `ws_errors` — WS throughput and health.

### Expected bottlenecks

This backend is a **single instance** with **single-writer SQLite**
(`MaxOpenConns(1)` in `backend/store.go`) and an **in-memory hub that
re-broadcasts the full board on every mutation** (`backend/handlers.go`).
Under load expect:

- Rising `submit`/`vote` p95/p99 as writes serialize on the one SQLite connection.
- `ws_propagation_ms` growth as full-board JSON is marshaled and fanned out to
  all ~150 clients on every change.
- Slow clients may silently miss updates (server drops messages when a client's
  16-slot send buffer is full).

## Notes / cautions

- **Production target writes real data.** Run off-hours and start with the smoke
  test. The test board persists: there is no board-delete endpoint, so cleanup
  only removes cards (enabled by default via `CLEANUP=1`).
- A single machine easily generates 150 WebSocket clients; no distributed
  runners are needed for this scale.
