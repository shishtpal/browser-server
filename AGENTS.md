# AGENTS.md

## Project Overview

Browser Server is a Go-based REST API server with an Astro + Vue frontend and Chromium and Firefox browser extensions. It manages personal data: todos, bookmarks, browsing history, a password wallet, screenshots, domain usage analytics, prompt templates, and AI-powered chat conversations. Data is stored in SQLite databases under `.data/`.

It is a **pnpm workspace monorepo**: the Go backend lives at the root, while `frontend/`, `extension/`, `extension-firefox/`, and `shared/*` are TypeScript workspace packages.

## Detailed guidance

This file is the entry point. Topic-specific guidance lives in `.agents/` and
takes precedence for its topic:

| File | Covers |
|------|--------|
| [`AGENTS.overview.md`](.agents/AGENTS.overview.md) | Repo notes, sub-projects, tech stack, build/run, auth, database design, search/omnibox, key conventions |
| [`AGENTS.git-workflow.md`](.agents/AGENTS.git-workflow.md) | Branching, commits, push/PR workflow |
| [`AGENTS.ai.md`](.agents/AGENTS.ai.md) | AI chat module: config files, providers, PATHs, architecture, bs-ai-chat CLI |
| [`AGENTS.quiz.md`](.agents/AGENTS.quiz.md) | Quiz / question bank configuration |
| [`AGENTS.browser.md`](.agents/AGENTS.browser.md) | Browser automation tools, eval modes, timeouts |
| [`AGENTS.domains.md`](.agents/AGENTS.domains.md) | Shared domain packages (todo, prompt, bookmark, history, quiz) |
| [`AGENTS.routes.md`](.agents/AGENTS.routes.md) | How to add a new API route |
| [`AGENTS.ai-tools.md`](.agents/AGENTS.ai-tools.md) | How to add an AI tool + catalog of existing tools |

Frontend projects carry their own `AGENTS.md` that takes precedence within their
directory:

- [`frontend/AGENTS.md`](frontend/AGENTS.md) — Astro + Vue web app
- [`extension/AGENTS.md`](extension/AGENTS.md) — Vite + Vue browser extension

## Quick start

- **Stack**: Go 1.25 backend + Astro/Vue frontend + Chromium/Firefox extensions; pnpm 11 monorepo; SQLite under `.data/`.
- **Build**: `./scripts/build.ps1` (Go-only: `go build -o bin/server.exe ./cmd/server`). Requires `CGO_ENABLED=1`.
- **Run**: `./bin/server.exe token generate` once, then `./bin/server.exe` (serves `:9191`; `/api/*` token-protected, `/health` public).
- **Checks**: backend `go test ./...` + `go vet ./...`; frontend package build; extension `type-check` + package build.
- **Never commit**: `bin/`, `dist/`, `node_modules/`, `.data/`, `.bs-token`, `.env`, local logs.
