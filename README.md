# ChessWeb (Go backend, React frontend)

A web chess app: *human vs bot*.  
Backend is written in *Go* (stdlib `net/http`), frontend is *React 19* (Vite).
The server is the source of truth: it validates moves, applies rules, and returns updated positions as *FEN*.

## Features (v1)

- Human vs Bot
  - Bot: simple *minimax* (2–4 ply) with *material-only* evaluation and a think-time limit
- Full chess rules supported by backend
  - Castling
  - En passant
  - Promotion (supports `...q/r/b/n`; if frontend sends no suffix, server defaults to queen when applicable)
- Game end detection
  - Check, checkmate, stalemate
  - Draw: threefold repetition, 50-move rule, insufficient material
- UI support endpoints
  - Request legal moves for a selected square (current side to move only)
  - Server returns updated FEN after each move (bot replies within the same `/move` request)
- Optional clocks through the API (`clockEnabled: true`, `initialSeconds`)
  - Elapsed time retains sub-second precision; response seconds are rounded up, including explicit zero on expiry.
  - Moves, game reads and legal-move requests update the clock. Timed games are polled by the frontend once per second.
  - Timeout returns `status: "timeout"` and `winner: "white" | "black"`.
  - Insufficient opposing mating material returns `status: "draw"`, `drawReason: "timeout_insufficient_material"`.
    Material detection includes minor pieces and bishop square colors, but does not solve position-dependent dead positions
    (the same limitation is documented for [material-based detection in python-chess](https://python-chess.readthedocs.io/en/latest/core.html#chess.Board.has_insufficient_material)).

## Repository structure

- `cmd/server` — Go HTTP server entrypoint
- `internal/chess` — chess rules/logic, FEN, move generation, validation
- `internal/engine` — bot engine (minimax)
- `internal/server` — HTTP API handlers, in-memory game store
- `web/` — React frontend (Vite)

## Run with Docker Compose

Requires Docker Engine and the Docker Compose plugin. From the repository root:

```bash
docker compose up --build -d --wait
```

Open http://localhost:8080. Swagger is at http://localhost:8080/swagger/index.html,
and the health check is at http://localhost:8080/healthz.

Compose builds two services: `api` (Go) and `web` (Nginx serving the React build).
Only Nginx publishes a host port; it forwards API requests to the Go service.
The frontend waits for the API health check using
[Compose's service_healthy dependency](https://docs.docker.com/compose/how-tos/startup-order/).
Build tools and source code are excluded from the runtime images.

To use a different host port:

```bash
PORT=8081 docker compose up --build -d --wait
```

View status and logs, or stop and remove the containers:

```bash
docker compose ps
docker compose logs -f
docker compose down
```

Games are stored in API memory and are lost when that container restarts.
There is no database or persistent volume.

## Requirements

- Go 1.25 or newer (Docker uses Go 1.27)
- Node.js 24 (also used in Docker)
- npm

## Run backend

From repository root:

```bash
go run ./cmd/server
```

Health check:
```bash
curl -s http://localhost:8080/healthz
```

## Run frontend

```bash
cd web
npm ci
npm run dev
```

Open the URL printed by Vite (usually `http://localhost:5173`).

The browser saves the current game ID in local storage and restores it after reload,
including completed games. Use **New game** to start again. If the API no longer has
the saved game (for example, after a container restart), a new game is created.
Network errors leave the saved ID intact and offer a retry.

Frontend checks (from `web/`, using Node.js 24):

```bash
npm test
npm run build
npm run lint
```

### API base URL

Frontend requests use relative `/api/v1/...` URLs. Nginx proxies them in Docker;
the Vite development server proxies them to `http://localhost:8080` during local development.


## Documentaion

### Re-generate docs
```bash
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```
