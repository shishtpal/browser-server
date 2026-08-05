# Browser Server

Browser Server is a self-hosted personal-data service with a web app and browser extensions. It stores todos, bookmarks, browsing history, password-wallet entries, screenshots, per-domain usage analytics, and AI chat conversations in local SQLite databases.

The project includes:

- A Go REST API protected by a single operator API token
- An Astro + Vue web interface
- An AI chat interface with multi-provider LLM support and server-side tool calling
- Chromium and Firefox extensions for history capture, usage tracking, popup access, and omnibox search
- Shared TypeScript packages for API types, client code, utilities, and extension UI/runtime code

> [!WARNING]
> This project is intended for personal, trusted environments. Wallet passwords are stored in SQLite without encryption. Do not expose the server directly to the public internet or use the wallet for sensitive production credentials.

## Features

- CRUD APIs and web views for todos, bookmarks, history, wallet entries, prompts, and users
- Bookmark and browser-history imports
- Todo screenshot capture and storage
- Domain usage analytics
- AI chat with streaming responses, configurable providers (OpenRouter, OpenAI, etc.), server-side tool calling, and optional web/file/memory/skill integrations
- Prompt management with folder-aware storage and search
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
# Put the token inside of `.bs-token` file, along with go binary

# Start the server
./bin/server.exe
```

Open [http://localhost:9191](http://localhost:9191), then enter the token printed by `token generate` in the web app's API token settings.

The build output is arranged as follows because the server resolves its static assets relative to the executable:

```text
bin/
├── server.exe
├── .bs-token
├── .data/
└── frontend/dist/
```

The token and data directories are created when their corresponding commands run; they are not build artifacts.

## Configuration

| Setting | Default | Description |
| --- | --- | --- |
| `--port PORT` | `9191` | Server port; takes precedence over `PORT` |
| `PORT` | `9191` | Server port when `--port` is not supplied |
| `DATA_PATH` | `.data/` beside the executable | SQLite databases and screenshot files |
| `SERVER_TOKEN_PATH` | `.bs-token` beside the executable | Operator token file |
| `bs-ai-config.json` | beside the executable | AI chat behavior config (tools, chat, memory, web/file/skills settings) |
| `bs-ai-models.json` | beside the executable | AI provider/model catalog |
| `bs-ai-mcp.json` | beside `bs-ai-config.json` | Optional local or remote MCP tool servers |
| `bs-ai-voice.json` | beside `bs-ai-config.json` | Optional voice typing providers, models, languages, and recording limits |
| `BS_AI_CONFIG_PATH` | — | Override path to `bs-ai-config.json` |
| `BS_AI_MODELS_PATH` | — | Override path to `bs-ai-models.json` |
| `BS_AI_MCP_PATH` | — | Override path to `bs-ai-mcp.json` |
| `BS_AI_VOICE_PATH` | — | Override path to `bs-ai-voice.json` |

Examples:

```powershell
./bin/server.exe --port 9090

$env:DATA_PATH = "D:\BrowserServerData"
$env:SERVER_TOKEN_PATH = "D:\BrowserServerData\.bs-token"
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
curl http://localhost:9191/health

curl -X POST http://localhost:9191/api/routes \
  -H "Authorization: Bearer YOUR_TOKEN"

curl "http://localhost:9191/api/todos?user_id=1" \
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

- API base URL (normally `http://localhost:9191`)
- The token generated by the server
- The data user ID
- Automatic capture preferences

In Chromium, type `bs` in the address bar and press Space or Tab to search the server's bookmarks and grouped history.

To capture the current page, right-click the page or selected text and open the **Browser Server** menu. You can save a bookmark, create a todo, or create a todo with a screenshot. Selected text and the source URL are included automatically. The default shortcuts are `Alt+Shift+B` for a bookmark and `Alt+Shift+T` for a todo; these can be changed from the browser's extension-shortcut settings. Captures made while the server is unavailable are stored locally and retried automatically.

## AI Chat

When `logging.enabled` is true, every interactive and durable-task provider call is audited in `request_logs`; correlated tool decisions and execution outcomes are stored in `tool_calls`.
Payload capture still requires `logging.log_full_payload` and applies secret/image redaction and `max_payload_bytes` limits.
Audit write failures are logged but do not fail otherwise successful AI work.
Authenticated operators can inspect bounded records with `GET /api/ai/logs` (filters: `source`, `status`, `conversation_id`, `task_id`, `limit`, `offset`) and aggregates with `GET /api/ai/monitoring?window_hours=24` (maximum 90 days).
See [AI Agent Logging and Monitoring Usage Guide](AI-LOGGING-MONITORING-GUIDE.md) for configuration, API examples, SQL analysis, and troubleshooting.

The server includes an optional AI chat feature that connects to OpenAI-compatible LLM providers (OpenRouter, OpenAI, etc.) and supports streaming responses with server-side tool calling.

### Setup

Create two sibling files next to the server binary: `bs-ai-config.json` for behavior toggles and `bs-ai-models.json` for the provider/model catalog. Provider API keys are read from environment variables.

`bs-ai-config.json`:

```json
{
  "cors_enabled": false,
  "default_provider": "openrouter",
  "tools": {
    "enabled": true,
    "allowed": ["search_tool", "get_current_time", "search_todos", "read_file", "write_file"],
    "max_iterations": 100
  },
  "web_search": {
    "enabled": true,
    "default_provider": "auto",
    "timeout_seconds": 30,
    "max_results": 10,
    "fallback": true
  },
  "memory": {
    "directory": ".memory"
  },
  "skills": {
    "enabled": true,
    "directory": ".skills"
  },
  "chat": { "system_prompt": "You are a helpful assistant.", "stream": true, "temperature": 0.7 }
}
```

`bs-ai-models.json`:

```json
{
  "providers": {
    "openrouter": {
      "type": "openai_compatible",
      "base_url": "https://openrouter.ai/api/v1",
      "api_key": "env:OPENROUTER_API_KEY",
      "request_timeout_seconds": 120,
      "retry_attempts": 10,
      "retry_delay_seconds": 5,
      "models": [
        { "id": "openai/gpt-4o-mini", "label": "GPT-4o Mini", "supports_tools": true, "default": true, "max_output_tokens": 4096 },
        { "id": "anthropic/claude-sonnet-4", "label": "Claude Sonnet 4", "supports_tools": true, "max_output_tokens": 8192 }
      ]
    }
  }
}
```

Set the API key environment variable (e.g. `$env:OPENROUTER_API_KEY = "sk-..."`) and restart the server. You can also set `BS_AI_CONFIG_PATH` or `BS_AI_MODELS_PATH` to point at a different location.

The web app will show the AI Chat page once both files are detected. If either file is missing, the chat page displays a "disabled" state with instructions.

Cross-origin API access is disabled by default.
Set `"cors_enabled": true` and restart the server to enable it, so the frontend development server can call the API from another port.

### MCP tool servers

AI Chat can discover tools from optional Model Context Protocol servers. Copy `bs-ai-mcp.json.example` to `bs-ai-mcp.json` beside `bs-ai-config.json`, then configure local stdio commands or remote Streamable HTTP endpoints:

```json
{
  "mcpServers": {
    "local-tools": {
      "command": "example-mcp-server",
      "args": ["--workspace", "."],
      "cwd": ".",
      "env": {
        "EXAMPLE_API_KEY": "env:EXAMPLE_API_KEY"
      },
      "allowed_tools": []
    },
    "remote-tools": {
      "url": "https://mcp.example.com/mcp",
      "headers": {
        "Authorization": "env:EXAMPLE_MCP_AUTHORIZATION"
      },
      "allowed_tools": ["search"]
    }
  }
}
```

An empty `allowed_tools` list permits every tool discovered from that server. Browser Server exposes each usable tool under a collision-safe name such as `mcp_local-tools_search` and groups it under `MCP: local-tools` in the Chat tools panel. MCP tools use the same per-tool toggles, `search_tool` discovery, approval/YOLO policy, skill restrictions, output limits, persistence, and cancellation path as built-in tools.

Values beginning with `env:` are resolved from the Browser Server process environment at startup. For an `Authorization` header, the referenced environment value must contain the complete header value, such as `Bearer ...`. Never put credentials directly in the JSON file. The authenticated `/api/ai/config` response reports only sanitized server status and public tool names.

Local stdio entries execute the configured program with Browser Server's operating-system permissions, so treat `bs-ai-mcp.json` as privileged operator configuration. Remote endpoints must use HTTPS except for loopback development servers. One unreachable server is reported as unavailable without disabling built-in tools or other connected MCP servers. Configuration and tool catalogs are loaded at startup; restart Browser Server after changing the file. Use `BS_AI_MCP_PATH` to load it from another location.

### CLI: bs-ai-chat

`bs-ai-chat` is a second `bs-*` binary that asks the configured models a
question from a terminal. It reuses the same `bs-ai-config.json` /
`bs-ai-models.json` / `bs-ai-mcp.json` files as the server, runs the full
agentic tool loop, and persists every run as a normal conversation in
`.data/bs-ai.db` (visible in the web UI).

```powershell
# Build alongside the server (or run ./scripts/build.ps1 -Target AIChat)
go build -o bin/bs-ai-chat.exe ./cmd/bs-ai-chat

# Discovery — no model call
bs-ai-chat.exe --list-models
bs-ai-chat.exe --list-tools

# Simplest path
bs-ai-chat.exe --no-tools "say hello in one word"

# Provider/model override
bs-ai-chat.exe --provider opencode.ai --model big-pickle --no-tools "hi"

# Tool loop (auto-approved) + reasoning/trace
bs-ai-chat.exe --yolo --verbose "what time is it in New Delhi?"

# Inline a file into the prompt
bs-ai-chat.exe --no-tools --file go.mod "which Go version?"

# Attach an image (requires a supports_vision model)
bs-ai-chat.exe --no-tools --provider openrouter.ai --model "qwen/qwen3-vl-30b-a3b-instruct" --image photo.png "describe this"

# Pipe a prompt from stdin
Get-Content notes.md | ./bin/bs-ai-chat.exe --no-tools

# Machine-readable output for scripting
bs-ai-chat.exe --json --yolo --verbose "what time is it?" | ConvertFrom-Json

# Continue an existing conversation
bs-ai-chat.exe --conversation conv_abc123 --no-tools "and in New York?"
```

Tools require `--yolo` (auto-approve); interactive approval is not yet
supported in the CLI — use `--no-tools` to disable them. The answer goes to
stdout and all trace output goes to stderr, so
`./bin/bs-ai-chat.exe --yolo --verbose "..." > answer.txt` captures exactly the
answer. See `bs-ai-chat --help` for the full flag list and config path
resolution chain.

### Voice typing

AI Chat voice typing is configured independently in `bs-ai-voice.json`. The supplied file defines Sarvam's streaming speech-to-text service with Auto, Hindi, and English language choices and is disabled by default. Set `"enabled": true` and provide the server-side key before startup:

```powershell
$env:SARVAM_API_KEY = "your-key"
```

The browser sends mono PCM audio to the token-protected `/api/ai/voice/transcribe` WebSocket. Browser Server validates the configured provider/model/language and adds the Sarvam API-key header upstream, so the secret is never exposed to the web app. Set `"enabled": false` in `bs-ai-voice.json` to disable voice typing, or remove the optional file. Use `BS_AI_VOICE_PATH` to load it from another location.

### Provider retries

Retry behavior is configured independently for each provider in `bs-ai-models.json`:

| Setting | Default | Valid range | Description |
| --- | --- | --- | --- |
| `request_timeout_seconds` | `120` | `1`-`600` | Timeout for each individual provider request |
| `retry_attempts` | `10` | `0`-`20` | Retries after the initial request; use `0` to disable retries |
| `retry_delay_seconds` | `5` | `1`-`300` | Fixed delay between retries |

The server retries transient failures such as network errors, timeouts, rate limits (`429`), provider errors (`5xx`), and malformed provider responses. Other `4xx` responses are returned immediately because retrying an invalid request or API key will not resolve it. Streaming requests are retried only before any output is emitted, preventing duplicate partial responses. Stopping a generation also cancels any pending retry delay.

Tool-call continuation has an additional recovery path. If the provider fails after one or more tools have run, every failure is recoverable regardless of HTTP status (including `400`). In manual mode the chat shows a **tool-call recovery** prompt with Resume and Stop actions; feedback can also be supplied before resuming. In YOLO mode the server resumes automatically every five seconds until the request succeeds or generation is stopped. Recovery removes the most recent assistant tool-call turn and its paired tool results from the provider payload before retrying, which avoids resending the tool call that caused the failed continuation.

### Key features

- Multiple provider and model selection from the web UI
- Streaming responses via SSE (Server-Sent Events)
- Configurable retries for transient provider failures
- Server-side tools the model can call (with user approval or auto-approve "YOLO mode")
- Calendar event creation and search via AI tool calls
- Prompt search and management via AI tools
- Optional web search, file operations, memory, and skill activation for richer workflows
- Conversation history persisted in SQLite
- Regenerate previous responses, stop in-progress generation

API key values starting with `env:` are resolved from environment variables at runtime so secrets stay out of config files.

## Development

Common commands from the repository root:

```powershell
# Backend checks/build
go test ./...
go vet ./...
go build -o bin/server.exe ./cmd/server

# Shared Libraries
pnpm --filter @browser-server/shared-modal type-check

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

The frontend development server defaults to `http://localhost:4321` and talks to the API at `http://localhost:9191`.

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

Keep pull requests focused and explain what changed, why it changed, and which checks passed. Never commit generated output or local secrets, including `bin/`, `dist/`, `node_modules/`, `.data/`, `.bs-token`, `.env` files, or AI config files that contain real API keys.

## Repository layout

```text
cmd/server/                    Go entry point and router
internal/                      Auth, database, handlers, middleware, models, and AI module
frontend/                      Astro + Vue web app
extension/                     Chromium extension wrapper
extension-firefox/             Firefox extension wrapper
shared/browser-client/         Canonical typed API client
shared/browser-types/          Shared domain and API types
shared/browser-utils/          Framework-free utilities
shared/browser-extension-core/ Shared Vue extension UI and runtime logic
scripts/build.ps1              Web app + Go release build
bs-ai-config.json              AI chat behavior config (tools, chat, memory, etc.)
bs-ai-models.json              AI provider/model catalog
bs-ai-mcp.json.example         Optional MCP tool server configuration example
PRD.md                         Detailed product and API documentation
ROADMAP.md                     Completed and planned work
```

## Data and backups

By default, data lives in `.data/` beside the running executable. The directory contains separate databases such as `todos.db`, `bookmarks.db`, `history.db`, `wallet.db`, `usage.db`, and `bs-ai.db` (AI conversations), plus uploaded screenshots.

Stop the server before copying the data directory for a simple consistent backup. Back up the token file separately if clients should continue using the same credential after a restore.

## License

MIT
