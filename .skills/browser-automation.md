---
name: browser-automation
label: Browser Automation
description: Efficiently control a browser for web automation, scraping, testing, and multi-step flows.
category: Development
tags: [browser, automation, chrome, firefox]
tools:
  - search_tool
  - recall_memory
  - write_memory
  - execute_command
  - execute_python
  - read_file
  - write_file
  - multi_edit
  - list_directory
---

All 25 tools share a common `target` object (browser profile + tab selector). Reuse `session_id` across a multi-step task so the tab stays locked to it.

## Tool Inventory (25)

| # | Tool | Category | One-liner |
|---|------|----------|-----------|
| 1 | `browser_list_instances` | Discovery | List online browser profiles (instance_id, label, browser, version) |
| 2 | `browser_list_tabs` | Discovery | List tabs: tab_uuid, url, title, active |
| 3 | `browser_stats` | Discovery | Automation health: instances, tab count, command counters (*read-only, no `target` needed*) |
| 4 | `browser_navigate` | Navigation | Navigate to a URL and wait for load |
| 5 | `browser_new_tab` | Navigation | Open a new tab (optionally at a URL), returns new tab_uuid |
| 6 | `browser_close_tab` | Navigation | Close the target tab |
| 7 | `browser_bring_to_front` | Navigation | Foreground the target tab's window |
| 8 | `browser_click` | Interaction | Click via CSS selector, XPath, or visible text |
| 9 | `browser_type` | Interaction | Type text into an input (clear, set value, fire events) |
| 10 | `browser_focus` | Interaction | Focus an element so subsequent type/press target it |
| 11 | `browser_press` | Interaction | Send a key or shortcut (Enter, Tab, Escape, Ctrl+S) |
| 12 | `browser_select` | Interaction | Set a `<select>` by option value or visible label |
| 13 | `browser_check` | Interaction | Check/uncheck a checkbox or radio button |
| 14 | `browser_hover` | Interaction | Fire mouseover on an element (menus, tooltips) |
| 15 | `browser_drag` | Interaction | Drag source element to target selector or coordinates |
| 16 | `browser_scroll` | Interaction | Scroll page or element (direction, amount, or into view) |
| 17 | `browser_scrape` | Extraction | Extract structured data: text, links, attributes, forms, tables |
| 18 | `browser_screenshot` | Extraction | Capture a PNG screenshot (URL + local path) |
| 19 | `browser_eval` | Extraction | Run JavaScript and return JSON-serialized value (gated) |
| 20 | `browser_pdf` | Extraction | Export page to PDF (CDP; URL + local path) |
| 21 | `browser_get_cookies` | State | Read cookies for the current origin or specific domain |
| 22 | `browser_set_cookie` | State | Set a single cookie on the current origin |
| 23 | `browser_storage` | State | Read/write `localStorage` or `sessionStorage` (get/set/remove/list) |
| 24 | `browser_execute` | Multi-step | Run a sequence of actions atomically in one tab (canonical for multi-step flows) |
| 25 | `browser_wait` | Utility | Wait for a duration and/or a selector to appear |

*26 total — `browser_execute` and `browser_stats` are orchestration/meta tools that sit outside the category table.*

## Core Patterns

### 1. Always discover first
```
browser_list_instances  → pick browser (first_online=true or instance_id)
browser_list_tabs       → find tab (uuid, url glob, title glob, or role)
```

### 2. Target object (every command tool needs it)
```json
{
  "target": {
    "browser": { "first_online": true },
    "session_id": "my-task-token",
    "tab": { "role": "active" }
  }
}
```
- `browser`: select profile via `instance_id`, `label`, or `first_online: true`
- `tab`: select via `uuid`, `url` glob, `title` glob, or `role` ("active" | "new"). `"new"` only works on `browser_new_tab` and `browser_execute`; `browser_navigate` targets **existing** tabs only and fails with `tab_not_found` otherwise.
- `session_id`: reuse across a multi-step task to keep the tab locked
- `browser_stats` is read-only and takes no `target`. `browser_list_instances`/`browser_list_tabs` accept target but resolve server-side without dispatching a command.

### 3. Multi-step flows → use `browser_execute`
This is the **canonical** way to chain actions. Individual tools are for probes only.

```json
{
  "target": { "browser": { "first_online": true }, "session_id": "s1", "tab": { "role": "new" } },
  "steps": [
    { "action": "navigate", "params": { "url": "https://example.com" } },
    { "action": "click", "selector": { "css": "button#login" } },
    { "action": "type", "selector": { "css": "input[name='email']" }, "params": { "text": "user@test.com", "clear": true } },
    { "action": "type", "selector": { "css": "input[name='password']" }, "params": { "text": "secret" } },
    { "action": "press", "params": { "keys": "Enter" } },
    { "action": "scrape", "params": { "extract": ["text", "links"] } }
  ]
}
```

Supported step actions: `navigate`, `click`, `type`, `press`, `select`, `check`, `hover`, `drag`, `scroll`, `focus`, `scrape`, `wait`, `eval`. Steps run in one content-script frame — background-API actions (`screenshot`, `new_tab`, `close_tab`, `get_cookies`, `set_cookie`, `pdf`, `bring_to_front`) are **not** available as steps; issue them as separate top-level tools.

Eval steps accept `params: { expression, mode?, timeout_ms? }`. `mode` defaults to `"inject"` — the extension injects a `<script>` tag that runs the expression in the page's **main world** (full access to page globals) with no debugger permission or infobar. If an inject-mode eval step is blocked by the page CSP (e.g. YouTube, `eval_inject_csp_blocked`) or produces nothing, it **automatically retries via CDP** and reports `via: "cdp-fallback"` on that step — the same flow, but via the debugger API (bypasses page CSP; needs the debugger permission and shows the debugging infobar). A step's own `params.mode` overrides the flow mode for that step.

### 4. Selector formats
All interaction tools accept three selector styles:
```json
{ "selector": { "css": "button[data-testid='submit']" } }
{ "selector": { "xpath": "//button[@data-testid='submit']" } }
{ "selector": { "text": "Submit" } }
```

### 5. Data extraction
- **`browser_scrape`** — structured data from the page
  - `extract`: `["text", "links", "attributes", "forms", "table"]`
  - `scope`: `"page"`, `"element"` (with selector), `"table"` (first `<table>`)
  - `rows_to_csv`: render table rows to CSV
  - `include_hidden`: include hidden elements
  - `max_links`: cap on link count (default 100)
- **`browser_screenshot`** — PNG of visible area; `result.data` is the server URL, `result.path` the local file path (feed `result.path` to `ocr_image`/`read_file` for further analysis)
- **`browser_eval`** — run JS in the page's main world and get a JSON string under `result.data` (`undefined` → `null`). `mode` defaults to `"inject"`: the extension injects a `<script>` tag that runs the expression in the page main world (no debugger permission or infobar). If the page CSP blocks inline scripts (e.g. YouTube — `eval_inject_csp_blocked`), the extension **automatically retries via CDP** and the result carries `via: "cdp-fallback"` (debugging infobar appears). Use `mode: "cdp"` directly to skip the inject attempt — CDP `Runtime.evaluate` via the debugger API bypasses page CSP but needs the debugger permission. Non-serializable results (circular values like `window`, DOM nodes → `{}`) — keep expressions returning plain data: extract primitives (`el.textContent`, `el.href`, `Boolean(el)`) instead of returning nodes. Set `timeout_ms` (default 60s) for long-running expressions. Gated; use only when scrape/click/type are insufficient.
- **`browser_pdf`** — export the page to PDF via CDP `Page.printToPDF` (works on Chromium browsers: Chrome, Edge, Brave). `result.data` is the server URL, `result.path` the local file path (feed `result.path` to `read_file` for further analysis). Firefox profiles lack `Page.printToPDF` and report `pdf_unsupported`.
- **`browser_storage`** — read/write the page's `localStorage` (`type: "local"`) or `sessionStorage` (`type: "session"`) from the page's main world. `action` ∈ `get` (one key → `{key, found, value}`), `set` (store a raw string value; `JSON.stringify` first for structured payloads), `remove` (→ `{ok, key, found}`), `list` (→ `{count, entries, truncated}`). By default oversized values are capped (get: 8 KiB, list entries: 512 chars) — pass `raw: true` for full untruncated values. Values are origin-scoped, so the target tab must be on the site whose storage you need. Works as a `browser_execute` step (e.g. to seed auth state before a flow).

### 6. Framework widgets (Vuetify comboboxes, custom dropdowns)
Raw `browser_type` / native value-setter does **not** bind to the widget's model — the value silently resets to `""` on re-render. Detect and drive such fields with this flow, and verify with a read-back eval before submitting:

```
1. browser_click   selector { css: "input[name='subject']" }     # focus/open
2. browser_eval    mode "cdp":                                    # force menu open
     const el = document.querySelector("input[name='subject']");
     el.closest('.v-field').setAttribute('aria-expanded','true');
     el.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true}));
3. browser_eval    mode "cdp":                                    # read options
     JSON.stringify([...document.querySelectorAll('[role=option]')].map(o=>o.textContent.trim()))
4. browser_click   selector { text: "<chosen option text>" }      # click option by text
5. browser_eval    → document.querySelector("input[name='subject']").value  # verify bound
```

The option list (`[role=option]`, `aria-owns` menu element) only renders **after** the widget is open — force `aria-expanded` + a keydown event first, then read it.

## Quick Reference by Task

| Task | Tools |
|------|-------|
| Login flow | `browser_execute` (navigate → click → type → press → scrape) |
| Form fill | `focus` → `type` → `select` → `check` → `click` |
| Form fill (framework widgets) | click to open → eval (force menu + `[role=option]`) → click option by text → eval verify |
| Scrape a page | `navigate` → `scrape` (text, links, table) |
| Screenshot evidence | `navigate` → `screenshot` |
| Export page to PDF | `pdf` (CDP; saved under `.data/browser-pdfs/`) |
| Cookie management | `get_cookies` / `set_cookie` |
| Web Storage management | `storage` (`type: local`/`session`, `action: get`/`set`/`remove`/`list`) |
| Complex drag-and-drop | `drag` (source → target) |
| Multi-page workflow | `browser_execute` with multiple steps |
| Debug automation | `stats` → `list_instances` → `list_tabs` → `screenshot` |

## Rules of Thumb

1. **`browser_execute` for multi-step; individual tools for probes.** Don't chain 10 separate calls when one `execute` does it atomically.
2. **Discover before acting.** Call `list_instances` and `list_tabs` before the first interaction to know what you're working with.
3. **Reuse `session_id`** across a multi-step task so the tab stays locked.
4. **`browser_eval` is gated.** Use `scrape`/`click`/`type` first; fall back to `eval` only when they can't do the job.
5. **`browser_pdf` needs CDP.** Works on Chromium browsers (Chrome, Edge, Brave); Firefox profiles lack `Page.printToPDF` and report `pdf_unsupported`.
6. **Timeouts default to 60 000 ms** (max 600 000). Set higher for slow pages, but chunk work instead of letting a single call time out.
7. **`browser_type` fires framework-compatible events.** It focuses, sets value, and fires input events — don't manually focus first unless the site requires it.
8. **`browser_check` dispatches proper change events.** Use it for checkboxes/radios instead of clicking.
9. **`browser_select` fires change/input events.** Prefer over clicking a `<select>` then an `<option>`.
10. **`rollback_url`** on `browser_execute` restores the page if any step fails (best effort).
11. **`browser_execute` cannot nest.** Steps are validated against the primitive specs server-side; an `execute` step in the list fails fast.
12. **After `navigate`, re-resolve the tab before interacting.** The bus syncs the tab registry after navigation, but the new tab's UUID may differ from the old one; call `list_tabs` if you need to chain manually instead of using `browser_execute`.
13. **`storage` steps work inside `browser_execute`.** Seed or clear site state (tokens, flags) in the same atomic flow that drives the page; steps take `{type, action, key, value, raw}` in `params`.
14. **Open new tabs with `browser_new_tab`, not `navigate`.** `browser_navigate` targets existing tabs only — `tab.role:"new"` fails with `tab_not_found`. It also rejects browser-internal pages (`edge://`, `chrome://`, `about:`) with `unsupported_page`; open a real URL in a new tab instead.
15. **Framework widgets ignore raw typing.** Vuetify comboboxes and custom dropdowns reset unbound values to `""` on re-render. Click to open, force the menu with a keydown event, click an option by its text, then verify via read-back eval.
16. **SPA CSP may block inject-mode eval.** If `browser_eval` (or an eval step) times out or returns nothing on a strict-CSP SPA, retry with `mode:"cdp"`.
