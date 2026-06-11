# ROADMAP.md

## ✅ Done

- **Go REST API** — Full CRUD for 5 domains: todos, bookmarks, browsing history, password wallet, users
- **Modular project structure** — `cmd/server/main.go` entry point, `internal/` packages for db, models, helpers, handlers
- **SQLite storage** — 5 separate `.data/*.db` files, auto-created with schema on first run
- **Tag-based bookmark filtering** — Tags stored as JSON strings in SQLite, filtered client-side, exposed as `[]string` in API
- **`/routes` endpoint** — Self-documenting API route listing via POST
- **Static frontend serving** — Astro + Vue + TailwindCSS frontend served from `frontend/dist/`
- **Full build script** — `scripts/build.ps1` builds frontend (bun or npm), Go binary, and copies dist for static serving
- **Monolith migration** — Refactored ~900-line single file into proper Go module structure
- **go vet compliance** — All cross-package struct literals use keyed fields

## 🔜 Next

- **Frontend UI pages** — Build pages for each domain (todos list, bookmarks grid, history timeline, wallet table, user management)
- **Authentication** — Login/logout, session management, protect routes
- **Input validation** — Validate request bodies (required fields, URL format, email format)
- **Error handling improvements** — Better error messages, structured error responses
- **Logging middleware** — Request logging with duration, status codes
- **CORS middleware** — Enable cross-origin requests for dev convenience
- **Tests** — Unit tests for handlers, integration tests for API endpoints
- **Health check endpoint** — `/health` for monitoring
- **Docker support** — Dockerfile for containerized deployment
- **CI/CD** — GitHub Actions or similar for automated build/test
