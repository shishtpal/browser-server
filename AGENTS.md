# AGENTS.md

## Project Overview

Browser Server is a Go-based REST API server with an Astro + Vue frontend and Chromium and Firefox browser extensions. It manages personal data: todos, bookmarks, browsing history, a password wallet, screenshots, domain usage analytics, and AI-powered chat conversations. Data is stored in SQLite databases under `.data/`.

It is a **pnpm workspace monorepo**: the Go backend lives at the root, while `frontend/`, `extension/`, `extension-firefox/`, and `shared/*` are TypeScript workspace packages.

## Sub-project guidance

This root `AGENTS.md` covers the Go backend and cross-cutting concerns. Each frontend project has its own `AGENTS.md` with details that take precedence within its directory:

- [`frontend/AGENTS.md`](frontend/AGENTS.md) — Astro + Vue web app
- [`extension/AGENTS.md`](extension/AGENTS.md) — Vite + Vue browser extension

## Git Repository Workflow

- Canonical repository: `https://github.com/shishtpal/browser-server.git`.
- Start work from an up-to-date `main`: fetch the remote, then create a focused branch such as `feat/short-description`, `fix/short-description`, or `docs/short-description`.
- Before editing, inspect `git status` and preserve unrelated user or agent changes. Never discard, overwrite, or stage files outside the requested work.
- Keep commits focused and use the repository's Conventional Commit-style subjects: `feat(scope): ...`, `fix(scope): ...`, `docs(scope): ...`, `refactor(scope): ...`, or `chore(scope): ...`.
- Run checks appropriate to the changed area before committing. At minimum, use `go test ./...` and `go vet ./...` for backend changes, the package build for frontend changes, and `type-check` plus the package build for extension changes.
- Review both `git diff` and `git diff --cached` so commits contain only intended changes.
- Never commit generated output, dependencies, runtime data, or secrets: `bin/`, `dist/`, `node_modules/`, `.data/`, `.bs-token`, `.env`, and local logs.
- Push feature branches and open pull requests against `main`; do not force-push or rewrite shared branch history.
- For fork-based work, use the fork as `origin` and add this repository as `upstream`: `git remote add upstream https://github.com/shishtpal/browser-server.git`.

## Tech Stack

- **Backend**: Go 1.25, gorilla/mux, mattn/go-sqlite3 (CGO required)
- **Frontend (web)**: Astro 6, Vue 3, TailwindCSS 4
- **Extensions**: Vite 8, Vue 3, TailwindCSS 4, Manifest V3 (Chromium and Firefox wrappers)
- **Shared packages**: framework-free API types/client/utilities plus shared Vue extension code in `shared/browser-extension-core`
- **Package manager**: pnpm 11 (workspace defined in `pnpm-workspace.yaml`)
- **Build**: PowerShell script (`scripts/build.ps1`), `CGO_ENABLED=1` required
- **Auth**: opaque operator-level API token (Bearer header), generated via `server token generate`

## Project Structure

```
browser-server/
├── cmd/server/main.go          # Entry point — CLI subcommands, router setup, static serving
├── internal/
│   ├── auth/token.go           # API token: generate/refresh/load/validate (.bs-token file)
│   ├── db/db.go                # SQLite connection management, schema init, sample data
│   ├── models/models.go        # Shared structs (Todo, Bookmark, History, WalletEntry, User, Route)
│   ├── helpers/helpers.go      # Query param parsing, path ID extraction, JSON tag conversion
│   ├── middleware/
│   │   ├── auth.go             # Bearer-token auth middleware (401/503)
│   │   ├── cors.go             # CORS middleware
│   │   └── logging.go          # Request logging middleware
│   ├── ai/                     # AI chat module (self-contained, registered via aiapi.Init())
│   │   ├── api/handlers.go    # HTTP handlers for /api/ai/* + module init & route registration
│   │   ├── chat/service.go    # Conversation orchestration, streaming, tool-call loops
│   │   ├── config/config.go   # Load & parse bs-ai-config.json
│   │   ├── provider/          # LLM provider abstraction (OpenAI-compatible endpoint)
│   │   ├── store/store.go     # SQLite persistence for conversations & messages (.data/bs-ai.db)
│   │   └── tools/registry.go  # Server-side tool definitions and execution
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
├── extension/                  # Chromium extension wrapper (see extension/AGENTS.md)
├── extension-firefox/          # Firefox extension wrapper
├── shared/                     # Shared TypeScript workspace packages
│   ├── browser-types/          # Domain models, DTOs, shared error/auth types
│   ├── browser-client/         # createBrowserServerClient() — the canonical API layer
│   ├── browser-utils/          # Pure helpers (date/duration formatting, favicon, etc.)
│   └── browser-extension-core/ # Shared Vue extension UI and runtime logic
├── scripts/build.ps1           # Full build: builds frontend, then Go binary, copies dist into bin/
├── bin/                        # Build output
├── pnpm-workspace.yaml         # pnpm workspace config
├── go.mod / go.sum
├── bs-ai-config.json           # AI chat configuration (providers, models, tools, logging)
├── README.md                   # Setup, usage, extension, and development guide
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

Serves on `:9191` by default. Override the port with `server --port 9090` or `PORT=9090 server`; the CLI flag takes precedence over the environment variable. All API endpoints live under `/api/` (todos, bookmarks, history, search, wallet, analytics, screenshots, users, routes) and require the API token. `/health` is public. Static frontend is served from `frontend/dist/` relative to the binary.

### Token CLI subcommands

- `server token generate` — create a random token, save to `.bs-token` next to the binary (refuses to overwrite).
- `server token refresh` — regenerate (rotate) the token, overwriting the existing file.

## Authentication

Auth is a single **operator-level API token** — there is no user login/registration. See [`internal/auth/token.go`](internal/auth/token.go) and [`internal/middleware/auth.go`](internal/middleware/auth.go).

- The token is an opaque random hex string stored in `.bs-token` alongside the binary (path overridable via `SERVER_TOKEN_PATH`).
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
- `bs-ai.db` — AI conversations, messages, and tool-call logs (managed by `internal/ai/store`)

Bookmark tags are stored as JSON strings in SQLite and parsed/presented as `[]string` in API responses.

## AI Chat

The AI chat module lives in `internal/ai/` and is self-contained — it manages its own config, database, providers, and tools. It is initialized in `cmd/server/main.go` via `aiapi.Init()` and registers its routes on the authenticated `/api` subrouter.

### Configuration (`bs-ai-config.json`)

Place `bs-ai-config.json` next to the server binary. The module reads it at startup; if the file is missing or has `"enabled": false` the feature is reported as disabled to the frontend. Key sections:

```jsonc
{
  "default_provider": "openrouter",
  "providers": {
    "<name>": {
      "type": "openai_compatible",
      "base_url": "https://...",
      "api_key": "env:ENV_VAR_NAME",   // resolved from environment at runtime
      "request_timeout_seconds": 120,
      "retry_attempts": 10,
      "retry_delay_seconds": 5,
      "models": [
        { "id": "openai/gpt-4o-mini", "label": "GPT-4o Mini", "supports_tools": true, "default": true }
      ]
    }
  },
  "tools": { "enabled": true, "allowed": ["get_current_time", "search_bookmarks"], "max_iterations": 5 },
  "chat": { "system_prompt": "...", "max_history_messages": 30, "stream": true, "temperature": 0.7 },
  "logging": { "enabled": true, "db_path": ".data/bs-ai.db", "retention_days": 60 }
}
```

API keys that start with `env:` are resolved from the corresponding environment variable.

Provider requests retry transient failures (network errors, timeouts, HTTP `429`/`5xx`, and malformed provider responses). `retry_attempts` is the number of retries after the initial request (`0` disables retries; valid range `0`-`20`), and `retry_delay_seconds` is the fixed delay between attempts (valid range `1`-`300`). Both regular and streaming completions use this policy, but a stream is never retried after it has emitted an event because doing so could duplicate output. Retry waits honor context cancellation.

### Architecture

| Package | Responsibility |
|---------|---------------|
| `ai/config` | Parses `bs-ai-config.json`, resolves env-based keys, exposes typed config |
| `ai/provider` | LLM abstraction; currently supports OpenAI-compatible (OpenRouter, OpenAI, etc.) |
| `ai/store` | SQLite persistence for conversations + messages |
| `ai/tools` | Registry of server-side tools the model can invoke (e.g. `get_current_time`, `search_bookmarks`) |
| `ai/chat` | Orchestration: builds prompts, streams completions, handles multi-turn tool-call loops |
| `ai/api` | HTTP handlers for all `/api/ai/*` routes + the `Init()` / `Register()` / `Close()` lifecycle |

### API endpoints

All under `/api/ai/` (token-protected):

| Method | Path | Description |
|--------|------|-------------|
| GET | `/ai/config` | Sanitized config (no secrets) + model catalog for the frontend |
| GET | `/ai/conversations` | List conversations (params: `q`, `limit`) |
| POST | `/ai/conversations` | Create a conversation |
| GET | `/ai/conversations/{id}` | Get conversation with all messages |
| PATCH | `/ai/conversations/{id}` | Rename or change model |
| DELETE | `/ai/conversations/{id}` | Delete conversation |
| POST | `/ai/conversations/{id}/messages` | Send a user message (supports SSE streaming) |
| POST | `/ai/conversations/{id}/tool-calls/{callID}` | Approve or reject a pending tool call |
| POST | `/ai/conversations/{id}/stop` | Cancel active generation |
| POST | `/ai/conversations/{id}/regenerate` | Supersede last response and regenerate |

The `/messages` endpoint supports both synchronous JSON responses and SSE streaming (controlled by the `stream` body field). During streaming, tool calls that require user approval pause the stream until the frontend submits a decision via the `/tool-calls/{callID}` endpoint.

### Frontend integration

The web app's chat page is at `frontend/src/components/ChatPage.vue` and is composed of:
- `chat/ChatTopBar.vue` — provider/model selects, YOLO mode toggle
- `chat/ChatSidebar.vue` — conversation list with search
- `chat/ChatMessageList.vue` — scrollable messages with empty-state suggestions
- `chat/ChatBubble.vue` — renders user, assistant (markdown), and tool messages
- `chat/ChatInput.vue` — auto-resizing textarea with send/stop
- `chat/ChatRegenerateButton.vue` — regenerate the last assistant response
- `chat/ChatMobileDrawer.vue` — mobile conversation list
- `chat/ChatDisabledState.vue` — placeholder when AI is not configured
- `chat/ChatCopyToast.vue` — clipboard feedback toast
- `chat/composables/useChatConfig.ts` — config, provider/model state, YOLO mode
- `chat/composables/useChatConversations.ts` — conversation CRUD, rename/delete
- `chat/composables/useChatMessaging.ts` — send, stream, tool decisions, regenerate, stop

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

## How to Add an AI Tool

AI tools are server-side functions the LLM can call during chat conversations. They live in `internal/ai/tools/` and are registered in the tool registry. Adding a new tool involves **3 files**:

### 1. Implement the tool function (`internal/ai/tools/<domain>.go`)

Group related tools in a single file (e.g. `git.go` for all git tools, `filesystem.go` for file ops). Each tool function has the same signature:

```go
func myTool(ctx context.Context, raw json.RawMessage) (any, error) {
    var a struct {
        Param1 string `json:"param1"`
        Param2 int    `json:"param2"`
    }
    if err := strict(raw, &a, map[string]bool{"param1": true, "param2": true}); err != nil {
        return nil, err
    }
    // Validate inputs
    if a.Param1 == "" {
        return nil, fmt.Errorf("param1 is required")
    }
    // Do work...
    return map[string]any{"result": "value"}, nil
}
```

Conventions:
- Use `strict(raw, &a, allowedKeys)` to validate and parse JSON arguments (rejects unknown fields)
- Return `(any, error)` — the `any` value is JSON-marshaled and sent back to the model
- Use `context.Context` for timeouts and cancellation
- Keep output under `maxOutput` (32 KiB) to avoid blowing up model context
- For shell/exec tools, use `exec.Command` with discrete args (never string interpolation) to prevent injection
- Validate that user-supplied ref names, paths, or identifiers don't start with `-` to prevent flag confusion

### 2. Register the tool (`internal/ai/tools/registry.go`)

Add an `r.add(Tool{...})` call inside the `New()` function:

```go
r.add(Tool{
    Name:        "my_tool",
    Description: "Short description of what the tool does",
    Schema:      json.RawMessage(`{"type":"object","properties":{"param1":{"type":"string","description":"..."},"param2":{"type":"integer"}},"required":["param1"],"additionalProperties":false}`),
    Execute:     myTool,
})
```

- `Name` — snake_case identifier used by the model to invoke the tool
- `Description` — helps the model decide when to use the tool; be concise but specific
- `Schema` — JSON Schema for the tool's parameters (the model sees this)
- `Execute` — the function from step 1

### 3. Whitelist in config validation AND `bs-ai-config.json`

**`internal/ai/config/config.go`** — add the tool name to the `known` map in the validation function:

```go
known := map[string]bool{
    // ... existing tools ...
    "my_tool": true,
}
```

**`bs-ai-config.json`** — add the tool name to `tools.allowed[]`:

```json
"tools": {
    "allowed": ["get_current_time", "search_bookmarks", "...", "my_tool"]
}
```

### Checklist

- [ ] Tool function in `internal/ai/tools/<domain>.go` (uses `strict()`, returns `(any, error)`)
- [ ] `r.add(Tool{...})` in `internal/ai/tools/registry.go` `New()` function
- [ ] Tool name added to `known` map in `internal/ai/config/config.go`
- [ ] Tool name added to `tools.allowed[]` in `bs-ai-config.json`
- [ ] `go build ./cmd/server` passes
- [ ] `go vet ./...` passes
- [ ] Server starts without "unknown tool" errors

### Existing tools

| Tool | File | Description |
|------|------|-------------|
| `get_current_time` | `registry.go` | Get server time in a timezone |
| `search_bookmarks` | `registry.go` | Search bookmark database |
| `execute_command` | `shell.go` | Run a shell command (30s timeout) |
| `read_file` | `filesystem.go` | Read a UTF-8 file (32 KiB max) |
| `write_file` | `filesystem.go` | Create/overwrite a file |
| `list_directory` | `filesystem.go` | List directory contents |
| `delete_file` | `filesystem.go` | Delete a file |
| `move_file` | `filesystem.go` | Move/rename a file |
| `copy_file` | `filesystem.go` | Copy a file |
| `git_status` | `git.go` | Repository status (branch, staged, untracked) |
| `git_diff` | `git.go` | View diffs (working tree, staged, between refs) |
| `git_log` | `git.go` | Commit history with filtering |
| `git_branch` | `git.go` | List/create/delete/rename branches |
| `git_checkout` | `git.go` | Switch or create branches |
| `git_commit` | `git.go` | Stage files and commit |
| `git_push` | `git.go` | Push to remote (uses --force-with-lease) |
| `git_pull` | `git.go` | Pull from remote |
| `git_merge` | `git.go` | Merge a branch |

## Key Conventions

- All handlers receive `(w http.ResponseWriter, r *http.Request)`
- Database connections are global vars exported from `internal/db`
- All `/api` routes are token-protected; only public endpoints (e.g. `/health`) go on the root router
- User filtering is done via `?user_id=` query parameter
- Cross-package struct literals use keyed fields (go vet compliance)
- Sample data is inserted on first run if tables are empty
- `DATA_PATH` env var overrides the default `.data/` location; `SERVER_TOKEN_PATH` overrides the `.bs-token` location
