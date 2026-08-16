# Browser Configuration

The browser automation AI tools (`browser_*`) are gated per tool by a sibling file next to the server binary: `bs-browser-config.json`. When the file is missing or `"enabled": false` every browser tool is unavailable. Each key under `tools` maps 1:1 to a tool name in `bs-ai-config.json` → `tools.allowed[]`; omitting a key keeps that tool enabled, and unknown keys are rejected so a typo surfaces immediately. See `internal/ai/browser/config/config.go` (the tool-name catalog) and `internal/ai/browser` for the tools.

Config path resolution (first match wins):

1. `BS_BROWSER_CONFIG_PATH` environment variable
2. `<executable dir>/bs-browser-config.json` (same `ExecutableDir()` anchor used by the AI config)

Key sections:

```jsonc
{
  "enabled": true,
  "tools": {
    "browser_list_instances": true,
    "browser_list_tabs": true,
    "browser_stats": true,
    "browser_navigate": true,
    "browser_click": true,
    "browser_type": true,
    "browser_press": true,
    "browser_scroll": true,
    "browser_wait": true,
    "browser_scrape": true,
    "browser_eval": true,
    "browser_screenshot": true,
    "browser_new_tab": true,
    "browser_close_tab": true,
    "browser_focus": true,
    "browser_select": true,
    "browser_check": true,
    "browser_hover": true,
    "browser_drag": true,
    "browser_get_cookies": true,
    "browser_set_cookie": true,
    "browser_pdf": true,
    "browser_bring_to_front": true,
    "browser_storage": true,
    "browser_execute": true
  },
  "eval": {
    "default_mode": "inject",
    "domains": {}
  },
  "timeouts": {
    "default_command_timeout_ms": 60000,
    "max_command_timeout_ms": 600000,
    "selector_timeout_ms": 10000
  }
}
```

How it applies: each browser tool carries an `Available` closure bound to its tool name. The tool registry evaluates that closure when building the model's toolset (so a disabled tool never reaches the model or the Chat tools panel) and again at execution time. Because the closure re-reads the process-global config, the admin Project Settings editor hot-reloads changes: disabling a tool hides it on the next provider step with no restart. A disabled tool reports `"tool is unavailable"` if the model still attempts it mid-conversation.

## `eval` — per-domain eval execution modes

The `eval` section steers how JavaScript runs in the page. Two modes exist:
`inject` (default; main-world `<script>` injection from the content script, no
debugger permission or infobar) and `cdp` (CDP `Runtime.evaluate` via the
debugger API, which shows the debugging infobar but bypasses page CSP that
blocks injected scripts). A call's explicit `mode` param always wins; the
config only fills in the default.

- `default_mode` — used when a call omits `mode` and no domain rule matches.
  `""`/missing resolves to `"inject"`.
- `domains` — a map (`{"youtube.com": "cdp", "*.twitch.tv": "inject"}`) or a
  list of hosts (each mapped to `"cdp"`) that forces a mode for specific
  sites, so CSP-strict domains work without the model requesting it. Matching
  is host-level and case-insensitive; a bare host also covers its subdomains
  (`youtube.com` matches `www.youtube.com`), so a leading `*. ` is optional.
  Keys may carry a scheme/port (stripped), and invalid modes or empty /
  wildcard-only patterns are rejected at save time.

The browser bus consults this for `browser_eval` and `browser_execute`
(orchestrate) commands whenever the resolved tab's params omit `mode`; a
`new`-role tab with no URL yet falls back to `default_mode`. Wired via
`bus.SetEvalModeFunc` from `internal/ai/browser/config`, so the admin editor
hot-reloads it. The extension still auto-falls back from `inject` to CDP when
a page CSP blocks the injected script (`result.via: "cdp-fallback"`).

**Do not add a JSON Schema `default` to the `mode` parameter.** Providers
that honor schema defaults fill omitted args with `"inject"`, which hard-codes
inject into every eval call and defeats both `default_mode` and the domain
rules (the bus only injects when `mode` is absent). The `mode` schemas in
`internal/ai/browser/core.go` / `workflow.go` intentionally declare no default;
regression tests in `schemas_test.go` enforce this.

## `timeouts` — per-command bounds

- `default_command_timeout_ms` — used when a command omits `timeout_ms`
  (`0` → `60000`). Minimum `100`.
- `max_command_timeout_ms` — hard ceiling on `timeout_ms` across every entry
  point (AI tools, REST `/api/browser/cmd`, CLI). `0` → `600000`; must be `>=`
  the default.
- `selector_timeout_ms` — default selector-polling budget for `browser_wait`
  when the model omits it. `0` → `10000`.

The browser bus applies these via `bus.SetCommandLimitsFunc`, so they bound
the AI executor, the REST relay, and the CLI alike and hot-reload with the
admin editor. The tool schemas surface the active values at build time.
