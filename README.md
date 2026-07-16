# Browser Server

Browser Server is a self-hosted personal-data service with a web app and browser extensions. It stores todos, bookmarks, browsing history, password-wallet entries, screenshots, and per-domain usage analytics in local SQLite databases.

The project includes:

- A Go REST API protected by a single operator API token
- An Astro + Vue web interface
- Chromium and Firefox extensions for history capture, usage tracking, popup access, and omnibox search
- Shared TypeScript packages for API types, client code, utilities, and extension UI/runtime code

> [!WARNING]
> This project is intended for personal, trusted environments. Wallet passwords are stored in SQLite without encryption. Do not expose the server directly to the public internet or use the wallet for sensitive production credentials.

## Features

- CRUD APIs and web views for todos, bookmarks, history, wallet entries, and users
- Bookmark and browser-history imports
- Todo screenshot capture and storage
- Domain usage analytics
- Combined bookmark/history search through the extension omnibox keyword `bs`
- One-click bookmark and todo capture from the page context menu or keyboard shortcuts
- Bearer-token authentication for every `/api/*` endpoint
- Configurable data directory and server port
- Separate local SQLite databases for each domain

## Requirements

- [Git](https://git-scm.com/downloads) for cloning and contributing
- [Go 1.25+](https://go.dev/dl/)
- A C compiler supported by Go, because `go-sqlite3` requires CGO
- [Node.js](https://nodejs.org/) and [pnpm 11](https://pnpm.io/installation)
- PowerShell for the provided full-build script

On Windows, a MinGW-w64 toolchain is one option for supplying the required C compiler.

## Get the repository

Clone the canonical repository and enter the project directory:

```powershell
git clone https://github.com/shishtpal/browser-server.git
Set-Location browser-server
```

To update an existing checkout without overwriting local work:

```powershell
git status
git pull --ff-only origin main
pnpm install
```

Run `git status` first and commit or stash local changes before pulling. `--ff-only` prevents Git from creating an unintended merge commit.

## Quick start

Run these commands from the repository root in PowerShell:

```powershell
# Install workspace dependencies
corepack enable
pnpm install

# SQLite requires CGO
$env:CGO_ENABLED = "1"

# Build the web app and server into bin/
./scripts/build.ps1

# Create the operator token (first run only)
./bin/server.exe token generate
# Put the token inside of `.server-token` file, along with go binary

# Start the server
./bin/server.exe
```

Open [http://localhost:8080](http://localhost:8080), then enter the token printed by `token generate` in the web app's API token settings.

The build output is arranged as follows because the server resolves its static assets relative to the executable:

```text
bin/
├── server.exe
├── .server-token
├── .data/
└── frontend/dist/
```

The token and data directories are created when their corresponding commands run; they are not build artifacts.

## Configuration

| Setting | Default | Description |
| --- | --- | --- |
| `--port PORT` | `8080` | Server port; takes precedence over `PORT` |
| `PORT` | `8080` | Server port when `--port` is not supplied |
| `DATA_PATH` | `.data/` beside the executable | SQLite databases and screenshot files |
| `SERVER_TOKEN_PATH` | `.server-token` beside the executable | Operator token file |

Examples:

```powershell
./bin/server.exe --port 9090

$env:DATA_PATH = "D:\BrowserServerData"
$env:SERVER_TOKEN_PATH = "D:\BrowserServerData\.server-token"
./bin/server.exe
```

Rotate the operator token with:

```powershell
./bin/server.exe token refresh
```

After rotation, update the token stored by the web app and each browser extension.

## API authentication

`GET /health` is public. Every route under `/api/` requires the operator token:

```bash
curl http://localhost:8080/health

curl -X POST http://localhost:8080/api/routes \
  -H "Authorization: Bearer YOUR_TOKEN"

curl "http://localhost:8080/api/todos?user_id=1" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

The server returns `401` for a missing or invalid token. If no token file was available at startup, protected routes return `503`; generate a token and restart the server.

See [PRD.md](PRD.md) for the detailed API reference. The authenticated `POST /api/routes` endpoint also returns the server's route catalog.

## Browser extensions

Install dependencies once at the workspace root, then build the extension for your browser.

### Chromium (Chrome, Edge, and compatible browsers)

```powershell
pnpm --dir extension build
pnpm --dir extension type-check
```

Open the browser's extensions page, enable developer mode, choose **Load unpacked**, and select the repository's `extension/` directory. Its root manifest points to the generated files in `extension/dist/`.

### Firefox

```powershell
pnpm --dir extension-firefox build
pnpm --dir extension-firefox type-check
```

For temporary local installation, open `about:debugging`, choose **This Firefox** → **Load Temporary Add-on**, and select `extension-firefox/manifest.json`.

In either extension's options page, configure:

- API base URL (normally `http://localhost:8080`)
- The token generated by the server
- The data user ID
- Automatic capture preferences

In Chromium, type `bs` in the address bar and press Space or Tab to search the server's bookmarks and grouped history.

To capture the current page, right-click the page or selected text and open the **Browser Server** menu. You can save a bookmark, create a todo, or create a todo with a screenshot. Selected text and the source URL are included automatically. The default shortcuts are `Alt+Shift+B` for a bookmark and `Alt+Shift+T` for a todo; these can be changed from the browser's extension-shortcut settings. Captures made while the server is unavailable are stored locally and retried automatically.

## Development

Common commands from the repository root:

```powershell
# Backend checks/build
go test ./...
go vet ./...
go build -o bin/server.exe ./cmd/server

# Web app
pnpm --dir frontend dev
pnpm --dir frontend build

# Chromium extension
pnpm --dir extension dev
pnpm --dir extension type-check

# Firefox extension
pnpm --dir extension-firefox build
pnpm --dir extension-firefox type-check
```

The frontend development server defaults to `http://localhost:4321` and talks to the API at `http://localhost:8080`.

## Contributing with Git

For changes you intend to contribute, use a fork and a short-lived branch instead of committing directly to `main`.

1. Fork [shishtpal/browser-server](https://github.com/shishtpal/browser-server) on GitHub.
2. Clone your fork and register this repository as `upstream`:

   ```powershell
   git clone https://github.com/YOUR_USERNAME/browser-server.git
   Set-Location browser-server
   git remote add upstream https://github.com/shishtpal/browser-server.git
   git fetch upstream
   ```

3. Create a focused branch from the latest upstream `main`:

   ```powershell
   git switch main
   git pull --ff-only upstream main
   git switch -c feat/short-description
   ```

4. Make and verify the change. Run the checks relevant to the packages you touched, using the commands in [Development](#development).
5. Review and commit only the intended files:

   ```powershell
   git status
   git diff
   git add path/to/changed-file
   git diff --cached
   git commit -m "feat(scope): describe the change"
   ```

   The repository follows Conventional Commit-style subjects such as `feat(extension): ...`, `fix(server): ...`, `docs(readme): ...`, and `chore(scripts): ...`.

6. Push the branch and open a pull request against `shishtpal/browser-server:main`:

   ```powershell
   git push -u origin feat/short-description
   ```

Keep pull requests focused and explain what changed, why it changed, and which checks passed. Never commit generated output or local secrets, including `bin/`, `dist/`, `node_modules/`, `.data/`, `.server-token`, or `.env` files.

## Repository layout

```text
cmd/server/                    Go entry point and router
internal/                      Auth, database, handlers, middleware, and models
frontend/                      Astro + Vue web app
extension/                     Chromium extension wrapper
extension-firefox/             Firefox extension wrapper
shared/browser-client/         Canonical typed API client
shared/browser-types/          Shared domain and API types
shared/browser-utils/          Framework-free utilities
shared/browser-extension-core/ Shared Vue extension UI and runtime logic
scripts/build.ps1              Web app + Go release build
PRD.md                         Detailed product and API documentation
ROADMAP.md                     Completed and planned work
```

## Data and backups

By default, data lives in `.data/` beside the running executable. The directory contains separate databases such as `todos.db`, `bookmarks.db`, `history.db`, `wallet.db`, and `usage.db`, plus uploaded screenshots.

Stop the server before copying the data directory for a simple consistent backup. Back up the token file separately if clients should continue using the same credential after a restore.

## License

MIT
