# AGENTS.md

## Project Overview

Browser Server is a Go-based REST API server with an Astro + Vue frontend and a companion browser extension. It manages personal data: todos, bookmarks, browsing history, a password wallet, screenshots, and domain usage analytics. Data is stored in SQLite databases under `.data/`.

It is a **pnpm workspace monorepo**: the Go backend lives at the root, while `frontend/`, `extension/`, and `shared/*` are TypeScript workspace packages.

## Sub-project guidance

This root `AGENTS.md` covers the Go backend and cross-cutting concerns. Each frontend project has its own `AGENTS.md` with details that take precedence within its directory:

- [`frontend/AGENTS.md`](frontend/AGENTS.md) — Astro + Vue web app
- [`extension/AGENTS.md`](extension/AGENTS.md) — Vite + Vue browser extension

## Tech Stack

- **Backend**: Go 1.25, gorilla/mux, mattn/go-sqlite3 (CGO required)
- **Frontend (web)**: Astro 6, Vue 3, TailwindCSS 4
- **Extension**: Vite 8, Vue 3, TailwindCSS 4, Manifest V3
- **Shared packages**: framework-free TypeScript (`shared/browser-types`, `shared/browser-client`, `shared/browser-utils`)
- **Package manager**: pnpm 11 (workspace defined in `pnpm-workspace.yaml`)
- **Build**: PowerShell script (`scripts/build.ps1`), `CGO_ENABLED=1` required
- **Auth**: opaque operator-level API token (Bearer header), generated via `server token generate`

## Project Structure

```
browser-server/
├── cmd/server/main.go          # Entry point — CLI subcommands, router setup, static serving
├── internal/
│   ├── auth/token.go           # API token: generate/refresh/load/validate (.server-token file)
│   ├── db/db.go                # SQLite connection management, schema init, sample data
│   ├── models/models.go        # Shared structs (Todo, Bookmark, History, WalletEntry, User, Route)
│   ├── helpers/helpers.go      # Query param parsing, path ID extraction, JSON tag conversion
│   ├── middleware/
│   │   ├── auth.go             # Bearer-token auth middleware (401/503)
│   │   ├── cors.go             # CORS middleware
│   │   └── logging.go          # Request logging middleware
│   └── handlers/
│       ├── health.go           # GET /health (public, no auth)
│       ├── routes.go           # POST /api/routes endpoint
│       ├── search.go           # GET /api/search/omnibox combined bookmark/history suggestions
│       ├── todos.go            # CRUD for /api/todos
│       ├── bookmarks.go        # CRUD for /api/bookmarks (with tag filtering)
│       ├── bookmark_import.go  # POST /api/bookmarks/import
│       ├── history.go          # CRUD for /api/history
│       ├── history_import.go   # POST /api/history/import
│       ├── wallet.go           # CRUD for /api/wallet (+ reveal)
│       ├── wallet_import.go    # POST /api/wallet/import
│       ├── screenshots.go      # Upload/serve todo screenshots
│       ├── analytics.go        # Domain usage upsert + summary
│       └── users.go            # Read/create for /api/users
├── frontend/                   # Astro + Vue web app (see frontend/AGENTS.md)
├── extension/                  # Vite + Vue browser extension (see extension/AGENTS.md)
├── shared/                     # Framework-free TS workspace packages
│   ├── browser-types/          # Domain models, DTOs, shared error/auth types
│   ├── browser-client/         # createBrowserServerClient() — the canonical API layer
│   └── browser-utils/          # Pure helpers (date/duration formatting, favicon, etc.)
├── scripts/build.ps1           # Full build: builds frontend, then Go binary, copies dist into bin/
├── bin/                        # Build output
├── pnpm-workspace.yaml         # pnpm workspace config
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

```powershell
# 1. Generate an API token (first run only; won't overwrite an existing one)
./bin/server.exe token generate

# 2. Start the server
./bin/server.exe
```

Serves on `:8080` by default. Override the port with `server --port 9090` or `PORT=9090 server`; the CLI flag takes precedence over the environment variable. All API endpoints live under `/api/` (todos, bookmarks, history, search, wallet, analytics, screenshots, users, routes) and require the API token. `/health` is public. Static frontend is served from `frontend/dist/` relative to the binary.

### Token CLI subcommands

- `server token generate` — create a random token, save to `.server-token` next to the binary (refuses to overwrite).
- `server token refresh` — regenerate (rotate) the token, overwriting the existing file.

## Authentication

Auth is a single **operator-level API token** — there is no user login/registration. See [`internal/auth/token.go`](internal/auth/token.go) and [`internal/middleware/auth.go`](internal/middleware/auth.go).

- The token is an opaque random hex string stored in `.server-token` alongside the binary (path overridable via `SERVER_TOKEN_PATH`).
- `auth.Load()` reads it into memory at startup; if missing, the server still starts but every `/api` request returns `503` until a token is generated.
- The `middleware.Auth` middleware is applied to the `/api` subrouter only. It accepts the token via `Authorization: Bearer <token>`, or via a `?token=` query param (needed for `<img>`-loaded screenshots that can't set headers). Comparison is constant-time.
- Responses: `401` for missing/invalid token, `503` when no token is configured. `/health` is intentionally left public.
- The multiple `users` records are data, **not** auth principals; `?user_id=` filtering is unchanged.
- Clients send the token through the shared client: `createBrowserServerClient(baseUrl, { getToken })`. The web app stores it in `localStorage` ([`frontend/src/lib/auth.ts`](frontend/src/lib/auth.ts)); the extension stores it in settings.

## Database Design

Each domain has its own SQLite database file in `.data/`:
- `users.db` — username, email
- `todos.db` — user_id, title, description, domain, screenshot_path, completed, timestamps
- `bookmarks.db` — user_id, title, url, description, tags (JSON string), folder_path, timestamps
- `history.db` — user_id, url, title, visited_at, duration
- `wallet.db` — user_id, username, password, website, description, timestamps
- `screenshots.db` — todo_id, filename, created_at (image files live in `.data/screenshots/`)
- `usage.db` — user_id, domain, date, total_seconds (unique per user/domain/date)

Bookmark tags are stored as JSON strings in SQLite and parsed/presented as `[]string` in API responses.

## Search / Omnibox

The extension's Chrome omnibox integration uses the keyword `bs` and calls `GET /api/search/omnibox` through the shared client. The endpoint combines:
- URL-grouped records from `history.db`, with `visit_count` showing how many times each URL was opened.
- Records from `bookmarks.db`, including bookmark metadata such as tags and folder path.

Results use a normalized `OmniboxSearchResult` shape in `internal/models` and `shared/browser-types`, with `source: "history"` or `source: "bookmark"` so clients can label suggestions clearly. The endpoint is token-protected like the rest of `/api`; the extension reads `apiBase`, `apiToken`, and `userId` from settings and passes the token via `createBrowserServerClient(..., { getToken })`.

When both sources have matches, the omnibox endpoint should preserve a balanced mix so bookmark suggestions are not crowded out by high-volume history matches. If one source has no matches, the other source can use the full result limit.

## How to Add a New Route

Adding a new API route involves touching **6 files** (plus `internal/db/db.go` for entirely new domains):

### 1. Define the model (`internal/models/models.go`)

Add your request/response structs with JSON tags. For import endpoints, create a dedicated result struct:

```go
type MyDomain struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
```

### 2. Create the handler (`internal/handlers/<domain>.go`)

Each handler file groups all CRUD functions for a domain. Handlers follow these conventions:
- Function signature: `func HandlerName(w http.ResponseWriter, r *http.Request)`
- Use `helpers.GetUserIDFromQuery(r)` for `?user_id=` filtering
- Use `helpers.GetIDFromPath(r)` for `{id}` path params
- Query the global DB var from `internal/db` (e.g., `db.HistoryDB`)
- Return JSON with `json.NewEncoder(w).Encode(...)`
- Set `w.Header().Set("Content-Type", "application/json")` before writing
- For file uploads, use `r.ParseMultipartForm()` + `r.FormFile("file")`
- Use `http.Error(w, "message", httpStatusCode)` for errors

### 3. Register the route (`cmd/server/main.go`)

API routes are registered on the auth-protected `api` subrouter (`api := r.PathPrefix("/api").Subrouter()`), so use **relative** paths (no `/api` prefix) — the subrouter adds it and `middleware.Auth` covers them automatically:

```go
api.HandleFunc("/mydomain", handlers.GetMyDomain).Methods("GET")
api.HandleFunc("/mydomain", handlers.CreateMyDomain).Methods("POST")
api.HandleFunc("/mydomain/{id}", handlers.GetMyDomainByID).Methods("GET")
api.HandleFunc("/mydomain/{id}", handlers.UpdateMyDomain).Methods("PUT")
api.HandleFunc("/mydomain/{id}", handlers.DeleteMyDomain).Methods("DELETE")
```

Only register on `r` directly for public, unauthenticated endpoints (like `/health`).

### 4. Add route description (`internal/handlers/routes.go`)

Add a `models.Route` entry so the `/api/routes` endpoint reflects the new route:

```go
{Method: "GET", Path: "/api/mydomain", Description: "Get all mydomain entries (filter: user_id)"},
```

### 5. Add the client method (`shared/browser-client/src/client.ts`)

Prefer adding the method to the **shared client** so both the web app and the extension can use it. The shared `apiFetch`/raw `fetch` calls must pass `getToken` so the Bearer header is attached:

```typescript
getMyDomain(userId?: number): Promise<MyDomain[]> {
  return apiFetch<MyDomain[]>(normalizedBaseUrl, 'GET', `/api/mydomain${buildQuery({ user_id: userId })}`, undefined, getToken)
}
```

Add any new types to `shared/browser-types/src/index.ts` (re-exported by `frontend/src/types.ts`). Then expose a thin wrapper in [`frontend/src/lib/api.ts`](frontend/src/lib/api.ts); any remaining raw `fetch` calls there must include `...authHeaders()` (from `frontend/src/lib/auth.ts`).

### Checklist

- [ ] Model struct in `internal/models/models.go`
- [ ] Handler functions in `internal/handlers/<domain>.go`
- [ ] Route registered on the `api` subrouter in `cmd/server/main.go`
- [ ] Route description in `internal/handlers/routes.go`
- [ ] Client method in `shared/browser-client/src/client.ts` (passes `getToken`)
- [ ] Types in `shared/browser-types/src/index.ts`
- [ ] Thin wrapper in `frontend/src/lib/api.ts` (raw fetches include `authHeaders()`)
- [ ] For new domains: SQLite DB init in `internal/db/db.go` (global var + `Init*DB()` + wire into `InitAll`/`CloseAll`)
- [ ] Go builds without errors (`go build ./cmd/server`)
- [ ] Web/extension components use the new client method as needed

For cross-domain search endpoints like `/api/search/omnibox`, keep the response type normalized and source-tagged rather than leaking raw domain models. Add the shared client method first and have the extension/frontend call that method instead of duplicating fetch logic.

## Key Conventions

- All handlers receive `(w http.ResponseWriter, r *http.Request)`
- Database connections are global vars exported from `internal/db`
- All `/api` routes are token-protected; only public endpoints (e.g. `/health`) go on the root router
- User filtering is done via `?user_id=` query parameter
- Cross-package struct literals use keyed fields (go vet compliance)
- Sample data is inserted on first run if tables are empty
- `DATA_PATH` env var overrides the default `.data/` location; `SERVER_TOKEN_PATH` overrides the `.server-token` location
