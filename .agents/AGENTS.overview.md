# Overview

## Project Overview

Browser Server is a Go-based REST API server with an Astro + Vue frontend and Chromium and Firefox browser extensions. It manages personal data: todos, bookmarks, browsing history, a password wallet, screenshots, domain usage analytics, prompt templates, and AI-powered chat conversations. Data is stored in SQLite databases under `.data/`.

It is a **pnpm workspace monorepo**: the Go backend lives at the root, while `frontend/`, `extension/`, `extension-firefox/`, and `shared/*` are TypeScript workspace packages.

## Current repo notes

- AI configuration uses sibling files: `bs-ai-config.json` for behavior toggles, `bs-ai-models.json` for the provider/model catalog, optional `bs-ai-mcp.json` for external MCP tool servers, `bs-ai-image-models.json` for image generation, and `bs-ai-tts.json` for text-to-speech. Keep config examples and documentation aligned with `internal/ai/config`, `internal/ai/mcp`, `internal/ai/images`, and `internal/ai/tts`.
- Quiz / Question Bank configuration uses `bs-quiz-config.json` next to the binary. See [`AGENTS.quiz.md`](AGENTS.quiz.md) and `internal/quiz/config/config.go`.
- Browser automation AI tools use `bs-browser-config.json` next to the binary to gate each `browser_*` tool individually. See [`AGENTS.browser.md`](AGENTS.browser.md) and `internal/ai/browser/config/config.go`. The tool-name catalog there must stay in sync with the tools built by `internal/ai/browser`.
- Prompt management and prompt folders are part of the shared domain model under `internal/prompt/`; they should remain the single source of truth for prompt validation and storage.
- When changing shared domain code, keep HTTP concerns in handlers and tool-argument validation in AI tools rather than duplicating logic in both layers.

## Sub-project guidance

This root `AGENTS.md` covers the Go backend and cross-cutting concerns. Each frontend project has its own `AGENTS.md` with details that take precedence within its directory:

- [`frontend/AGENTS.md`](../frontend/AGENTS.md) — Astro + Vue web app
- [`extension/AGENTS.md`](../extension/AGENTS.md) — Vite + Vue browser extension

## Tech Stack

- **Backend**: Go 1.25, gorilla/mux, mattn/go-sqlite3 (CGO required)
- **Frontend (web)**: Astro 6, Vue 3, TailwindCSS 4
- **Extensions**: Vite 8, Vue 3, TailwindCSS 4, Manifest V3 (Chromium and Firefox wrappers)
- **Shared packages**: framework-free API types/client/utilities, the shared markdown renderer (`shared/browser-markdown`), plus shared Vue extension code in `shared/browser-extension-core`
- **Package manager**: pnpm 11 (workspace defined in `pnpm-workspace.yaml`)
- **Build**: PowerShell script (`scripts/build.ps1`), `CGO_ENABLED=1` required
- **Auth**: disjoint opaque Bearer tokens — operator (`server token generate`) and optional project administrator (`server token admin-generate`)

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

- `server token generate` / `refresh` — create or rotate `.bs-token` for ordinary API access.
- `server token admin-generate` / `admin-refresh` / `admin-delete` — manage the optional `.bs-token-admin` credential for Project Settings.

## Authentication

There is no user login/registration. Ordinary APIs use an operator token while `/api/admin/*` uses a separate, opt-in administrator token; neither credential is accepted by the other tier. See [`internal/auth`](../internal/auth) and [`internal/middleware/auth.go`](../internal/middleware/auth.go).

- The token is an opaque random hex string stored in `.bs-token` alongside the binary (path overridable via `SERVER_TOKEN_PATH`).
- `auth.Load()` reads it into memory at startup; if missing, the server still starts but every `/api` request returns `503` until a token is generated.
- `middleware.Auth` protects ordinary `/api` routes with `.bs-token`; `middleware.AdminAuth` protects the earlier-registered `/api/admin` prefix with `.bs-token-admin`. Both accept `Authorization: Bearer <token>` or `?token=` and compare in constant time.
- Responses: `401` for missing/invalid credentials, `503` when no operator token is configured, and `403 admin_disabled` when no admin token is configured. `/health` is intentionally public.
- The multiple `users` records are data, **not** auth principals; `?user_id=` filtering is unchanged.
- Clients send the token through the shared client: `createBrowserServerClient(baseUrl, { getToken })`. The web app stores it in `localStorage` ([`frontend/src/lib/auth.ts`](../frontend/src/lib/auth.ts)); the extension stores it in settings.

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
- `ai-images.db` + `ai-images/` — generated image gallery (`internal/ai/images`)
- `ai-voices.db` + `ai-voices/` — generated speech gallery (`internal/ai/tts`)

Bookmark tags are stored as JSON strings in SQLite and parsed/presented as `[]string` in API responses.

## Search / Omnibox

The extension's Chrome omnibox integration uses the keyword `bs` and calls `GET /api/search/omnibox` through the shared client. The endpoint combines:
- URL-grouped records from `history.db`, with `visit_count` showing how many times each URL was opened.
- Records from `bookmarks.db`, including bookmark metadata such as tags and folder path.

Results use a normalized `OmniboxSearchResult` shape in `internal/models` and `shared/browser-types`, with `source: "history"` or `source: "bookmark"` so clients can label suggestions clearly. The endpoint is token-protected like the rest of `/api`; the extension reads `apiBase`, `apiToken`, and `userId` from settings and passes the token via `createBrowserServerClient(..., { getToken })`.

When both sources have matches, the omnibox endpoint should preserve a balanced mix so bookmark suggestions are not crowded out by high-volume history matches. If one source has no matches, the other source can use the full result limit.

## Key Conventions

- All handlers receive `(w http.ResponseWriter, r *http.Request)`
- Database connections are global vars exported from `internal/db`
- All `/api` routes are token-protected; ordinary routes use the operator token and `/api/admin/*` uses only the admin token. Only public endpoints (e.g. `/health`) go on the root router
- User filtering is done via `?user_id=` query parameter
- Cross-package struct literals use keyed fields (go vet compliance)
- Sample data is inserted on first run if tables are empty
- `DATA_PATH` env var overrides the default `.data/` location; `SERVER_TOKEN_PATH` overrides the `.bs-token` location
