# VoteRun v2

A real-time retrospective app for agile teams. Create boards, add cards across columns, vote on what matters, and watch everything update live across every participant's screen.

## Features

- **Real-time collaboration** — cards, votes, and edits sync instantly to all connected clients via WebSockets.
- **Accounts** — register/login with email + password (bcrypt-hashed, JWT sessions). Boards are owned by their creator; invited participants join via the shared link.
- **Retro boards** — organize feedback into columns (e.g. _What went well_, _What didn't_, _Action items_).
- **Voting** — participants vote to surface the team's top priorities.
- **Lightweight & self-hostable** — single Go binary backend with an embedded SQLite database; no external services required.

## Tech Stack

| Layer       | Technology                          |
| ----------- | ----------------------------------- |
| Frontend    | React + Vite                        |
| Backend     | Go ([Gin](https://gin-gonic.com/))  |
| Realtime    | WebSockets                          |
| Database    | SQLite                              |

## Project Structure

```
voterun/
├── backend/        # Gin (Go) API + WebSocket server, SQLite storage
└── frontend/       # React + Vite single-page app
```

## Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) 1.22+
- [Node.js](https://nodejs.org/) 20+
- npm (or pnpm / yarn)

### Quick start (both servers)

From the repo root, run the helper script to start the backend and frontend together (it installs dependencies on first run):

```bash
./dev.sh
```

Then open `http://localhost:5173`. Press `Ctrl+C` to stop both. To run them individually instead, follow the steps below.

### Backend

```bash
cd backend
go mod download
go run .
```

The API server starts on `http://localhost:8080` and creates a local SQLite database file (e.g. `voterun.db`) on first run.

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The Vite dev server starts on `http://localhost:5173` and proxies API/WebSocket requests to the backend.

## Configuration

The backend can be configured via environment variables:

| Variable      | Description                          | Default       |
| ------------- | ------------------------------------ | ------------- |
| `PORT`        | Port the API server listens on       | `8080`        |
| `DB_PATH`     | Path to the SQLite database file      | `voterun.db`  |
| `CORS_ORIGIN` | Allowed origin for the frontend       | `http://localhost:5173` |
| `JWT_SECRET`  | Secret used to sign auth tokens (set a strong value in production) | `dev-insecure-secret-change-me` |

## Run with Docker

The repo ships with Dockerfiles for both apps and a Compose file for spinning up the full stack:

```bash
docker compose up --build
```

- Frontend (nginx serving the SPA + proxying the API/WebSocket): `http://localhost:3000`
- Backend API (exposed for direct testing): `http://localhost:8081`

The SQLite database is stored in a named Docker volume (`voterun-data`), so your retros **persist across restarts and redeploys**. To wipe all data, remove the volume:

```bash
docker compose down -v
```

## Deploying to a server

For production deployment behind custom domains (`new.voterun.app` → frontend,
`api.voterun.app` → backend) with a host nginx reverse proxy and HTTPS, see
[`deploy/README.md`](./deploy/README.md). Images are published to GHCR by the
GitHub Actions workflow in [`.github/workflows`](./.github/workflows).

## Build for Production

```bash
# Build the frontend
cd frontend
npm run build

# Build the backend binary
cd ../backend
go build -o voterun .
```

## License

MIT
