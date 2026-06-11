# AGENTS.md

## Project Overview

Browser Server is a Go-based REST API server with an Astro + Vue frontend. It manages personal data: todos, bookmarks, browsing history, and a password wallet. Data is stored in SQLite databases under `.data/`.

## Tech Stack

- **Backend**: Go 1.24, gorilla/mux, mattn/go-sqlite3 (CGO required)
- **Frontend**: Astro 5, Vue 3, TailwindCSS 4
- **Build**: PowerShell script (`scripts/build.ps1`), `CGO_ENABLED=1` required

## Project Structure

```
browser-server/
├── cmd/server/main.go          # Entry point — router setup, static file serving
├── internal/
│   ├── db/db.go                # SQLite connection management, schema init, sample data
│   ├── models/models.go        # Shared structs (Todo, Bookmark, History, WalletEntry, User, Route)
│   ├── helpers/helpers.go      # Query param parsing, path ID extraction, JSON tag conversion
│   └── handlers/
│       ├── routes.go           # GET /routes endpoint
│       ├── todos.go            # CRUD for /todos
│       ├── bookmarks.go        # CRUD for /bookmarks (with tag filtering)
│       ├── history.go          # CRUD for /history
│       ├── wallet.go           # CRUD for /wallet
│       └── users.go            # Read/create for /users
├── frontend/                   # Astro + Vue + TailwindCSS frontend (built to dist/)
├── scripts/build.ps1           # Full build: builds frontend, then Go binary, copies dist into bin/
├── bin/                        # Build output
├── go.mod / go.sum
├── PRD.md                      # Product requirements and API documentation
├── AGENTS.md                   # This file
└── ROADMAP.md                  # What's done and what's next
```

## Building

```powershell
# Full build (requires bun or npm + CGO_ENABLED=1)
./scripts/build.ps1

# Go-only build
go build -o bin/server.exe ./cmd/server
```

Requires `CGO_ENABLED=1` for SQLite. Set it persistently in PowerShell:
```powershell
[System.Environment]::SetEnvironmentVariable("CGO_ENABLED", "1", "User")
```

## Running

```
./bin/server.exe
```

Serves on `:8080`. API endpoints are under `/todos`, `/bookmarks`, `/history`, `/wallet`, `/users`. Static frontend from `frontend/dist/` relative to the binary.

## Database Design

Each domain has its own SQLite database file in `.data/`:
- `users.db` — username, email
- `todos.db` — user_id, title, description, completed, timestamps
- `bookmarks.db` — user_id, title, url, description, tags (JSON string), timestamps
- `history.db` — user_id, url, title, visited_at, duration
- `wallet.db` — user_id, username, password, website, description, timestamps

Bookmark tags are stored as JSON strings in SQLite and parsed/presented as `[]string` in API responses.

## Key Conventions

- All handlers receive `(w http.ResponseWriter, r *http.Request)`
- Database connections are global vars exported from `internal/db`
- User filtering is done via `?user_id=` query parameter
- Cross-package struct literals use keyed fields (go vet compliance)
- Sample data is inserted on first run if tables are empty
- `DATA_PATH` env var overrides the default `.data/` location
