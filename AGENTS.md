# AGENTS.md

## Project Overview

Browser Server is a Go-based REST API server with an Astro + Vue frontend and Chromium and Firefox browser extensions. It manages personal data: todos, bookmarks, browsing history, a password wallet, screenshots, domain usage analytics, prompt templates, and AI-powered chat conversations. Data is stored in SQLite databases under `.data/`.

It is a **pnpm workspace monorepo**: the Go backend lives at the root, while `frontend/`, `extension/`, `extension-firefox/`, and `shared/*` are TypeScript workspace packages.

## Current repo notes

- AI configuration is split into two sibling files: `bs-ai-config.json` for behavior toggles (tools, chat, memory, web/file/skills settings, `default_provider`, `cors_enabled`) and `bs-ai-models.json` for the provider/model catalog. Keep config examples and documentation aligned with the schema in `internal/ai/config/types.go`.
- Prompt management and prompt folders are part of the shared domain model under `internal/prompt/`; they should remain the single source of truth for prompt validation and storage.
- When changing shared domain code, keep HTTP concerns in handlers and tool-argument validation in AI tools rather than duplicating logic in both layers.

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

### Configuration

Place two sibling files next to the server binary: `bs-ai-config.json` for behavior toggles and `bs-ai-models.json` for the provider/model catalog. The module reads them at startup; if the main file is missing or has `"enabled": false`, the feature is reported as disabled. If `bs-ai-models.json` is missing while the main config exists, AI is also disabled.

Key sections in `bs-ai-config.json`:

```jsonc
{
  "default_provider": "openrouter",
  "tools": { "enabled": true, "allowed": ["get_current_time", "search_bookmarks"], "max_iterations": 5 },
  "chat": { "system_prompt": "...", "max_history_messages": 30, "stream": true, "temperature": 0.7 },
  "logging": { "enabled": true, "db_path": ".data/bs-ai.db", "retention_days": 60 },
  "web_search": { ... },
  "file_tools": { ... },
  "memory": { ... },
  "skills": { ... },
  "paths": { "additional_dirs": [], "binaries": {} },
  "cors_enabled": false
}
```

### Configured PATHs

The `paths` section in `bs-ai-config.json` lets the operator specify additional directories and explicit binary overrides so AI tools (`execute_command`, `execute_python`, git tools, `get_diagnostics`) can find executables that are not on the server's inherited PATH:

```jsonc
"paths": {
  "additional_dirs": ["/opt/homebrew/bin", "/usr/local/go/bin"],
  "binaries": { "git": "/usr/local/git/bin/git" }
}
```

Windows example — make `go` available to `execute_command`:

```jsonc
"paths": {
  "additional_dirs": [],
  "binaries": { "go": "D:/Tools/lang.go/bin/go.exe" }
}
```

- `additional_dirs` are prepended to the `PATH` of child processes and searched (in order) before the system PATH.
- `binaries` maps a binary name to an explicit full path. Tools that invoke a known binary directly (`get_diagnostics`, git tools, `execute_python`) use the exact mapped path, bypassing PATH entirely.
- For commands run by `execute_command`, the parent directories of mapped binaries are prepended to the child `PATH` (before `additional_dirs`), so a command such as `go version` resolves to the configured executable. The mapping key must match the executable's basename without a platform extension (`.exe`, `.cmd`, `.bat`, `.com`) for shell name resolution; arbitrary aliases (e.g. `"go": "custom-tool.exe"`) are honored by direct tools but not by `execute_command`.
- Both are optional; empty/missing `paths` = inherit parent process PATH.
- Relative paths are resolved against the config file's directory.
- Limits: 20 additional dirs, 30 binaries.

Provider/model catalog in `bs-ai-models.json`:

```jsonc
{
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
  }
}
```

API keys that start with `env:` are resolved from the corresponding environment variable.

Provider requests retry transient failures (network errors, timeouts, HTTP `429`/`5xx`, and malformed provider responses). `retry_attempts` is the number of retries after the initial request (`0` disables retries; valid range `0`-`20`), and `retry_delay_seconds` is the fixed delay between attempts (valid range `1`-`300`). Both regular and streaming completions use this policy, but a stream is never retried after it has emitted an event because doing so could duplicate output. Retry waits honor context cancellation.

### How to add an AI tool

1. Implement the tool in `internal/ai/tools/` (register it in `registry.go`).
2. Add the JSON schema beside the code (e.g. `schemas/<tool>.json`).
3. If the tool should be selectable/allowed by operators, list it in the `tools.allowed` array of `bs-ai-config.json`.
4. Update the `tools` description in `internal/ai/tools/` and any relevant `AGENTS.md`/`README.md` sections.

### Architecture

| Package | Responsibility |
|---------|---------------|
| `ai/config` | Parses `bs-ai-config.json` and `bs-ai-models.json`, resolves env-based keys, exposes typed config |
| `ai/provider` | LLM abstraction; currently supports OpenAI-compatible (OpenRouter, OpenAI, etc.) |
| `ai/store` | SQLite persistence for conversations + messages |
| `ai/tools` | Registry of server-side tools the model can invoke (e.g. `get_current_time`, `search_bookmarks`) |
| `ai/chat` | Orchestration: builds prompts, streams completions, handles multi-turn tool-call loops |
| `ai/api` | HTTP handlers for all `/api/ai/*` routes + the `Init()` / `Register()` / `Close()` lifecycle |

## Search / Omnibox

The extension's Chrome omnibox integration uses the keyword `bs` and calls `GET /api/search/omnibox` through the shared client. The endpoint combines:
- URL-grouped records from `history.db`, with `visit_count` showing how many times each URL was opened.
- Records from `bookmarks.db`, including bookmark metadata such as tags and folder path.

Results use a normalized `OmniboxSearchResult` shape in `internal/models` and `shared/browser-types`, with `source: "history"` or `source: "bookmark"` so clients can label suggestions clearly. The endpoint is token-protected like the rest of `/api`; the extension reads `apiBase`, `apiToken`, and `userId` from settings and passes the token via `createBrowserServerClient(..., { getToken })`.

When both sources have matches, the omnibox endpoint should preserve a balanced mix so bookmark suggestions are not crowded out by high-volume history matches. If one source has no matches, the other source can use the full result limit.

## Domain Packages (shared business logic)

Domains that are reachable from **both** the REST API and the AI tools keep their
logic in a dedicated package under `internal/<domain>/` rather than duplicating it
in `internal/handlers/` and `internal/ai/tools/`:

| Package | Used by |
|---------|---------|
| `internal/todo` | `handlers/todos.go`, `todo_subtasks.go`, `todo_reorder.go`, and the `add_todo_record`, `update_todo_record`, `manage_calendar`, `search_todos`, `search_calendar` tools |
| `internal/prompt` | `handlers/prompts.go` and the `manage_prompt`, `search_prompts` tools |
| `internal/bookmark` | `handlers/bookmarks.go`, `bookmark_import.go`, and the `search_bookmarks` tool |
| `internal/history` | `handlers/history.go`, `history_import.go`, `search.go`, and the `search_history` tool |

Each package is layered the same way:

- **`<domain>.go`** — pure validation and constants (field limits, valid enum
  values, date parsing). No database access, so it is trivial to unit test.
- **`store.go`** — the single source of truth for the SQL: the `Columns`
  constant, the row `Scan`, `Create`, an `UpdateBuilder` for partial updates,
  and ownership checks. Every query that scans a row **must** select `Columns`.
- **`view.go`** — the two renderings of a record: `Response(...)` for the REST
  API (typed `models.*Response`) and `Map(...)` for AI tools (a flat
  `map[string]any` where blank optional strings become `null`).

**Rules when working in these domains:**

- Never re-declare a column list, row scanner, or validation table in a handler
  or a tool — import the domain package instead. Adding a column then means
  editing one `Columns` constant.
- Keep HTTP concerns (status codes, `helpers.WriteError`) in handlers and
  tool-argument concerns (`strict(...)`) in tools. Domain packages return plain
  values and sentinel errors (`ErrNotFound`, `ErrFolderNotOwned`, …) that the
  caller maps to its own response format.
- The REST API and the AI tools intentionally accept **different status sets**
  for todos: use `IsValidCoreStatus` for the API and `IsValidStatus` for tools,
  which additionally tolerates the legacy `done`/`cancelled` aliases.
- Prompt rows store `created_at` with sub-second precision from Go rather than
  SQL's whole-second `CURRENT_TIMESTAMP`, because prompt listings order by
  `created_at` and would otherwise be non-deterministic within the same second.

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
| `get_current_time` | `get_current_time.go` | Get server time in a timezone |
| `search_todos` | `search_todos.go` | Search todo database (filter by status, priority, text) |
| `search_calendar` | `search_calendar.go` | Search calendar events (todos with scheduled dates, date range filtering) |
| `manage_calendar` | `manage_calendar.go` | Manage calendar events: add, edit, remove, get (todos with start_date/end_date/rrule) |
| `search_bookmarks` | `search_bookmarks.go` | Search bookmark database |
| `search_history` | `search_history.go` | Search browsing history |
| `execute_command` | `execute_command.go` | Run a shell command (30s timeout) |
| `web_search` | `web.go` | Search the web (requires web_search config) |
| `web_fetch` | `web.go` | Fetch content from a URL |
| `read_file` | `read_file.go` | Read a UTF-8 file (32 KiB max) |
| `write_file` | `write_file.go` | Create/overwrite a file |
| `edit_file` | `edit_file.go` | Find-and-replace edit in a file |
| `multi_edit` | `multi_edit.go` | Atomic multi-file find-and-replace edits |
| `list_directory` | `list_directory.go` | List directory contents |
| `delete_file` | `delete_file.go` | Delete a file |
| `move_file` | `move_file.go` | Move/rename a file |
| `copy_file` | `copy_file.go` | Copy a file |
| `directory_tree` | `directory_tree.go` | Recursive directory tree listing |
| `search_code` | `search_code.go` | Regex search across files |
| `analyze_code` | `analyze_code.go` | AST-based code analysis |
| `get_diagnostics` | `get_diagnostics.go` | Get compile/lint diagnostics for a file |
| `git_status` | `git_status.go` | Repository status (branch, staged, untracked) |
| `git_diff` | `git_diff.go` | View diffs (working tree, staged, between refs) |
| `git_log` | `git_log.go` | Commit history with filtering |
| `git_branch` | `git_branch.go` | List/create/delete/rename branches |
| `git_checkout` | `git_checkout.go` | Switch or create branches |
| `git_commit` | `git_commit.go` | Stage files and commit |
| `git_push` | `git_push.go` | Push to remote (uses --force-with-lease) |
| `git_pull` | `git_pull.go` | Pull from remote |
| `git_merge` | `git_merge.go` | Merge a branch |
| `ai_remember` | `memory.go` | Store a memory |
| `ai_recall` | `memory.go` | Recall a specific memory |
| `ai_search_memory` | `memory.go` | Search stored memories |
| `ai_list_memories` | `memory.go` | List all memories |
| `ai_forget` | `memory.go` | Delete a memory |
| `ai_update_memory` | `memory.go` | Update an existing memory |
| `ai_resolve_references` | `memory.go` | Resolve memory cross-references |
| `ai_lazy_memory` | `memory.go` | Lazy-load memory content |
| `ai_manage_cache` | `memory.go` | Manage memory cache |
| `list_skills` | `skills.go` | List available AI skills |
| `activate_skill` | `skills.go` | Activate an AI skill |
| `deactivate_skill` | `skills.go` | Deactivate an AI skill |
| `get_active_skills` | `skills.go` | Get currently active skills |

## Key Conventions

- All handlers receive `(w http.ResponseWriter, r *http.Request)`
- Database connections are global vars exported from `internal/db`
- All `/api` routes are token-protected; only public endpoints (e.g. `/health`) go on the root router
- User filtering is done via `?user_id=` query parameter
- Cross-package struct literals use keyed fields (go vet compliance)
- Sample data is inserted on first run if tables are empty
- `DATA_PATH` env var overrides the default `.data/` location; `SERVER_TOKEN_PATH` overrides the `.bs-token` location
