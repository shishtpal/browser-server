# AI Chat

The AI chat module lives in `internal/ai/` and is self-contained — it manages its own config, database, providers, and tools. It is initialized in `cmd/server/main.go` via `aiapi.Init()` and registers its routes on the authenticated `/api` subrouter.

## Configuration

Place two sibling files next to the server binary: `bs-ai-config.json` for behavior toggles and `bs-ai-models.json` for the provider/model catalog. The module reads them at startup; if the main file is missing or has `"enabled": false`, the feature is reported as disabled. If `bs-ai-models.json` is missing while the main config exists, AI is also disabled.

An optional `bs-ai-mcp.json` sibling configures stdio or Streamable HTTP MCP servers. `internal/ai/mcp` owns protocol transports, discovery, public-name routing, result normalization, and session lifecycle. MCP tools are appended to the validated runtime allowlist and adapted into `internal/ai/tools`; they must continue to use the normal active-tool, skill, approval, output-limit, persistence, and cancellation path. The browser remains MCP-protocol agnostic.

## Key sections in `bs-ai-config.json`

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
  "cors_enabled": false,
  "openrouter": {
    "site_url": "https://github.com/shishtpal/browser-server",
    "app_name": "Browser Server"
  }
}
```

- `openrouter` — user-editable OpenRouter attribution headers. When a provider's `base_url` points to OpenRouter, the agent chat (streaming and non-streaming), `ocr_image`, `recall_memory` (synthesize), image generation, video generation, and TTS attach `HTTP-Referer`/`Referer` (from `site_url`) and `X-Title` (from `app_name`) to their provider requests. Other OpenAI-compatible providers receive no attribution headers. Omitting the section uses the defaults shown above.

## Configured PATHs

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

## Provider/model catalog

Provider/model catalog in `bs-ai-models.json`:

```jsonc
{
  "providers": {
    "<name>": {
      "type": "openai_compatible",   // or "gemini_interactions"
      "base_url": "https://...",
      "api_key": "env:ENV_VAR_NAME",   // resolved from environment at runtime
      "request_timeout_seconds": 120,
      "retry_attempts": 10,
      "retry_delay_seconds": 5,
      "google_search": true,   // gemini_interactions only: adds the model's
                               // native google_search tool alongside server tools
      "models": [
        { "id": "openai/gpt-4o-mini", "label": "GPT-4o Mini", "supports_tools": true, "default": true }
      ]
    }
  }
}
```

Two provider types are supported. `openai_compatible` (OpenRouter, OpenAI, etc.)
is the default. `gemini_interactions` targets Google's Gemini Interactions API
(`POST {base}/interactions`, typically `https://generativelanguage.googleapis.com/v1beta`).
It is stateless — the full history is re-sent as an input step array every turn —
so branching/regeneration/editing keep working without interaction chaining.
Model IDs may carry the `models/` prefix (stripped before the request). The
client lives in `internal/ai/provider/gemini_interactions.go` and is selected by
`provider.New` (see `internal/ai/provider/provider.go`); unknown types fall back
to the OpenAI-compatible client, and config validation rejects them first.
`google_search: true` on a `gemini_interactions` provider appends the model's
native `google_search` tool to every request in addition to server-side tools.

API keys that start with `env:` are resolved from the corresponding environment variable.

Provider requests retry transient failures (network errors, timeouts, HTTP `429`/`5xx`, and malformed provider responses). `retry_attempts` is the number of retries after the initial request (`0` disables retries; valid range `0`-`20`), and `retry_delay_seconds` is the fixed delay between attempts (valid range `1`-`300`). Both regular and streaming completions use this policy, but a stream is never retried after it has emitted an event because doing so could duplicate output. Retry waits honor context cancellation.

## Architecture

| Package | Responsibility |
|---------|---------------|
| `ai/config` | Parses `bs-ai-config.json` and `bs-ai-models.json`, resolves env-based keys, exposes typed config |
| `ai/mcp` | Parses optional `bs-ai-mcp.json`, connects MCP servers, discovers/routes tools, and owns sessions |
| `ai/provider` | LLM abstraction; supports OpenAI-compatible (OpenRouter, OpenAI, etc.) and Gemini Interactions clients |
| `ai/store` | SQLite persistence for conversations + messages |
| `ai/tools` | Registry of server-side tools the model can invoke (e.g. `get_current_time`, `search_bookmarks`) |
| `ai/chat` | Orchestration: builds prompts, streams completions, handles multi-turn tool-call loops |
| `ai/bootstrap` | Provider-agnostic wiring shared by the HTTP server and `bs-ai-chat`: config → profiles → skills → store → MCP → chat service |
| `ai/api` | HTTP handlers for all `/api/ai/*` routes + the `Init()` / `Register()` / `Close()` lifecycle (server-only concerns layered on `ai/bootstrap`) |

For adding a new AI tool (tool function, registry, schemas, allowlist), see [`AGENTS.ai-tools.md`](AGENTS.ai-tools.md).

## bs-ai-chat CLI

`cmd/bs-ai-chat` builds a second `bs-*` binary that drives the same
`chat.Service.SubmitStream` pipeline as the HTTP SSE handler from a terminal.
Runs are persisted as normal conversations in `.data/bs-ai.db` and reuse
`bs-ai-config.json` / `bs-ai-models.json` / `bs-ai-mcp.json` unchanged, so a
machine already configured for the server needs zero new setup.

Key flags (see `--help` for the full list):

- `--provider` / `--model` — override the selection; default to
  `default_provider` and the provider's default model.
- `--prompt`, positional args, or piped stdin — the prompt. `--file` inlines
  file contents ahead of the prompt; `--image` attaches a validated image
  (requires a `supports_vision` model).
- `--yolo` — auto-approve tool calls. **Tools require `--yolo`**: interactive
  approval is not yet supported, so without it the CLI fails fast with
  `Error: tools require --yolo ...`. Use `--no-tools` to disable tools.
- `--tools <a,b>` — tool allowlist; `--skills <a,b>` — skills to activate;
  `--profile <name>` — system-prompt profile.
- `--conversation <id>` — continue an existing conversation.
- `--json` — one structured JSON object on stdout (includes `tool_calls` and
  `reasoning` only under `--verbose`).
- `--list-models` / `--list-tools` — discovery commands that make no model call.
- `--tool-output raw|auto|json` — per-request tool result format, mapped onto
  `SubmitRequest.RawToolOutput`: `raw` forces raw output, `json` (default)
  forces JSON, `auto` defers to the `tools.raw_output` config allowlist.
- `--working-dir <path>` — chdir into `path` after bootstrap (config and data
  files stay anchored to the binary/`--config`) but before reading `--file` /
  `--image` inputs and before any tool runs, so relative paths and
  cwd-sensitive tools (`execute_command`, file tools) operate there.
- `--verbose` — streams reasoning + tool-call trace to stderr. The answer
  always goes to stdout, so `bs-ai-chat "..." > answer.txt` captures exactly
  the answer while the trace stays on the terminal.

Config path resolution (first match wins):

1. `--config` flag
2. `BS_AI_CONFIG_PATH` environment variable
3. `bs-ai-config.json` in the current working directory

`bs-ai-models.json` and `bs-ai-mcp.json` resolve as siblings automatically.

Always run `bs-ai-chat` from a directory with the config files, or pass
`--config`. The binary opens the same SQLite database as the server with
pending-message reconciliation disabled (`store.OpenWithOptions` with
`ReconcilePending: false`) so it never cancels a generation the server has
in flight.
