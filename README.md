# VoteRun v2

A real-time retrospective app for agile teams. Create boards, add cards across columns, vote on what matters, and watch everything update live across every participant's screen.

## Features

- **Real-time collaboration** — cards, votes, and edits sync instantly to all connected clients via WebSockets.
- **Retro boards** — organize feedback into columns (e.g. _What went well_, _What didn't_, _Action items_).
- **Voting** — give each participant a limited number of votes to surface the team's top priorities.
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
