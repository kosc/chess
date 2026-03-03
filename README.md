# ChessWeb (Go backend, React frontend)

A web chess app: *human vs bot*.  
Backend is written in *Go* (stdlib `net/http`), frontend is *React 18* (Vite).  
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

## Repository structure

- `cmd/server` — Go HTTP server entrypoint
- `internal/chess` — chess rules/logic, FEN, move generation, validation
- `internal/engine` — bot engine (minimax)
- `internal/server` — HTTP API handlers, in-memory game store
- `web/` — React frontend (Vite)

> If your frontend folder name differs (e.g. `chessweb-frontend/`), adjust the commands below accordingly.

## Requirements

- Go 1.25.x
- Node.js 18+ (recommended)
- npm

## Run backend

From repository root:

```bash
go run ./cmd/server
```

Health check:
```bash
curl -s http://localhost:8080/heathz
```

## Run frontend

```bash
cd web
npm install
npm run dev
```

Open the URL printed by Vite (usually `http://localhost:5173`).

### API base URL

Frontend uses an API base URL (defaults to `http://localhost:8080`).  

