# Contributing to VoteRun

Thanks for taking the time to contribute! This document describes how we work in this repository: how to set up your environment, how to commit, branch, and open pull requests, and the basic rules we expect everyone to follow.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Project Layout](#project-layout)
- [Branching Strategy](#branching-strategy)
- [Commit Messages](#commit-messages)
- [Pull Requests](#pull-requests)
- [Coding Standards](#coding-standards)
- [Repository Rules](#repository-rules)

## Code of Conduct

We are committed to a welcoming, harassment-free experience for everyone. By participating in this project you agree to:

- Be respectful, inclusive, and considerate in all communication.
- Assume good intent and give constructive, actionable feedback.
- Accept that decisions are made through discussion and consensus.
- Not tolerate harassment, discrimination, or personal attacks of any kind.

Report unacceptable behavior to the project maintainers privately. Maintainers may remove, edit, or reject contributions and comments that violate these principles.

## Getting Started

1. **Fork** the repository (external contributors) or create a branch (maintainers).
2. **Clone** your fork and install dependencies.
3. Run both apps locally with the helper script:

```bash
./dev.sh
```

   Or run them individually — see the [README](./README.md) for backend and frontend steps.

4. Make your change, test it, and open a pull request.

## Project Layout

```
voterun/
├── backend/        # Gin (Go) API + WebSocket server, SQLite storage
├── frontend/       # React + Vite single-page app
├── dev.sh          # Runs backend + frontend together
└── README.md
```

Keep changes scoped to the relevant area. Avoid mixing backend and frontend refactors in the same commit unless they are part of the same feature.

## Branching Strategy

- `main` is always **deployable**. Never commit directly to `main`.
- Create a descriptive branch off `main` for every change:

| Type        | Prefix      | Example                          |
| ----------- | ----------- | -------------------------------- |
| Feature     | `feat/`     | `feat/board-export`              |
| Bug fix     | `fix/`      | `fix/websocket-reconnect`        |
| Refactor    | `refactor/` | `refactor/store-queries`         |
| Docs        | `docs/`     | `docs/contributing-guide`        |
| Chore/tooling | `chore/`  | `chore/bump-vite`                |

- Keep branches short-lived and rebase on the latest `main` before opening a PR.

## Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<optional scope>): <short summary>

<optional body explaining the "why">

<optional footer, e.g. "Closes #123">
```

**Allowed types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `build`, `ci`.

**Rules:**

- Use the imperative mood: "add", not "added" or "adds".
- Keep the summary line under ~72 characters and don't end it with a period.
- Explain the *why* in the body, not just the *what*.
- One logical change per commit; keep commits focused and atomic.

**Examples:**

```
feat(board): add per-participant vote limit
fix(ws): reconnect after the server drops the connection
docs: document the dev.sh quick-start script
refactor(store): extract board loading into helper
```

## Pull Requests

Before opening a PR:

- [ ] Branch is rebased on the latest `main`.
- [ ] Backend builds: `cd backend && go build ./... && go vet ./...`
- [ ] Frontend builds: `cd frontend && npm run build`
- [ ] Code is formatted (`gofmt` for Go; consistent formatting for JS/JSX).
- [ ] You manually tested the change (note how in the PR description).

In the PR description:

- Summarize **what** changed and **why**.
- Link any related issues (e.g. `Closes #42`).
- Include screenshots or a short clip for UI changes.

PRs require at least **one approving review** and a **green CI** before merging. Prefer **squash merge** so `main` keeps a clean, linear history.

## Coding Standards

### Backend (Go)

- Run `gofmt`/`go fmt ./...` before committing; CI assumes formatted code.
- Keep handlers thin; put data access in `store.go`.
- Return meaningful HTTP status codes and JSON error bodies.
- Don't commit the SQLite database file (`voterun.db`) — it's git-ignored.

### Frontend (React + Vite)

- Functional components and hooks only.
- Keep components small and focused; colocate component-specific logic.
- Use the existing API wrapper (`src/lib/api.js`) for network calls.
- Keep all styling in `src/styles.css` and reuse the existing theme variables.

## Repository Rules

- **Never force-push to `main`** or any shared branch.
- **Do not commit secrets** (`.env`, API keys, credentials). They belong in environment variables.
- **Do not commit build artifacts or dependencies** (`node_modules/`, `dist/`, compiled binaries, `*.db`). See [`.gitignore`](./.gitignore).
- Keep PRs small and reviewable; split large efforts into multiple PRs.
- Discuss significant architectural changes in an issue before implementing.
- Be kind in code review — critique the code, not the person.

Happy contributing!
