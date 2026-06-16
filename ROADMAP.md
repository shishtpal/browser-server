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

- [x] **Health check endpoint**
  - Add `/health` route in `cmd/server/main.go`
  - Return simple JSON status with uptime/build-ready signal
  - Confirm endpoint works without auth and can be used by Docker/CI checks

- [x] **Logging middleware**
  - Add middleware for method, path, status code, and request duration
  - Apply middleware centrally in router setup
  - Keep logs readable for local development and future production debugging

- [x] **CORS middleware**
  - Allow frontend dev origin during local development
  - Support common methods and headers used by the API
  - Keep configuration simple and environment-aware so production stays locked down

- **Input validation**
  - Validate required fields for each create/update request
  - Validate URL fields for bookmarks/history and email format for users
  - Return consistent `400 Bad Request` responses for invalid payloads

- **Error handling improvements**
  - Standardize JSON error response shape across handlers
  - Replace generic `http.Error` usage where structured responses are better
  - Improve database and parsing error messages without leaking internals

- **Authentication**
  - Introduce token-based auth strategy
  - Protect write routes and any sensitive wallet/user operations
  - Add frontend auth state handling and route guarding

- [x] **Shared frontend/extension code**
  - Use `shared/` for framework-agnostic TypeScript packages only, starting with API client, request/response types, and small pure utilities
  - Keep Vue components, Astro pages, browser-extension runtime code, and app-specific composables inside `frontend/` or `extension/`
  - Expand `shared/browser-client` into the canonical API layer for both apps instead of maintaining duplicate clients
  - Move duplicated types from `frontend/src/types.ts` into shared package exports and have both apps import from the same source
  - Add auth-aware client methods and shared error/result types after auth and error response formats are defined
  - Prefer an incremental migration: move types first, then API functions, then any reusable pure helpers
  - Avoid sharing UI too early; only extract design tokens or headless utilities later if duplication becomes real
  - Proposed package layout:
    - `shared/browser-types` — Domain models, request/response DTOs, auth/session types, import result types, shared error shapes
    - `shared/browser-client` — `createBrowserServerClient(baseUrl, options)`, per-domain API methods, auth-aware fetch wrapper, query helpers
    - `shared/browser-utils` — Pure helpers such as date formatting, URL normalization, tag serialization/parsing, environment-agnostic mappers
  - Ownership boundaries:
    - `frontend/` keeps Astro routes, Vue components, page-level composables, and presentation-specific view models
    - `extension/` keeps Chrome/browser API wrappers, popup/options state, storage/settings integration, and background-script logic
    - `shared/` must not import Vue, Astro, or browser-extension APIs
  - Migration sequence:
    - Phase 1: extract shared domain and API types from `frontend/src/types.ts`
    - Phase 2: replace duplicated frontend API functions with imports from `shared/browser-client`
    - Phase 3: add auth/session handling and shared error parsing after backend auth is stable
    - Phase 4: extract only truly duplicated pure helpers from app code into `shared/browser-utils`
  - Success criteria:
    - Frontend and extension compile against the same exported API types
    - Only one maintained API client implementation exists
    - Shared packages stay framework-free and testable in isolation

- **Frontend UI pages**
  - Build todos list with create/update/delete actions
  - Build bookmarks grid with tag filtering and bookmark form
  - Build history timeline/table with sorting and filtering
  - Build wallet table with secure create/edit flows
  - Build user management page and shared layout/navigation

- **Tests**
  - Add unit tests for helper functions and handler-level validation logic
  - Add integration tests for key CRUD API flows
  - Add test coverage for auth, error responses, and bookmark tag behavior

- **Docker support**
  - Add Dockerfile for Go server and bundled frontend assets
  - Ensure SQLite data path is configurable via environment variables/volume mount
  - Verify container build and local run workflow

- **CI/CD**
  - Add GitHub Actions workflow for Go build and frontend build
  - Run tests and fail fast on lint/build errors
  - Optionally publish artifacts or binaries after successful main-branch builds
