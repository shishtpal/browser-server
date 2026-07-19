# AGENTS.md — Browser Extension

Vite + Vue 3 + TailwindCSS Manifest V3 browser extension for Browser Server. This file covers `extension/`; the [root `AGENTS.md`](../AGENTS.md) covers the Go backend and cross-cutting concerns.

## What it does

Captures browsing history and per-domain time usage in real time and syncs them to the local server (`localhost:9191`). Provides a popup UI (todos, history, bookmarks, wallet, analytics views), an options page for settings, and Chrome omnibox suggestions from server-side bookmarks/history.

## Tech Stack

- **Vite 8** — multi-entry build (popup, options, background service worker, content script)
- **Vue 3** (`<script setup lang="ts">`) for popup + options UIs
- **TailwindCSS 4** via `@tailwindcss/vite`; styles in shared core
- **Manifest V3** — service-worker background script, `chrome.*` APIs
- **Shared workspace packages** — `@browser-server/shared-client`, `@browser-server/shared-types`, `@browser-server/shared-utils`, `@browser-server/extension-core`
- **`@types/chrome`** for the extension APIs

## Commands

Run from `extension/` (pnpm workspace):

```bash
pnpm dev          # vite build --watch
pnpm build        # vite build → dist/
pnpm type-check   # vue-tsc --noEmit
```

Load the unpacked extension by pointing Chrome at the `extension/` directory (the manifest references `dist/*` outputs). Run `pnpm type-check` before considering a change done — there is no test suite.

## Build outputs (`vite.config.ts`)

Main build has 5 entry points, output to `dist/`:
- `popup` → `dist/popup.html`
- `options` → `dist/options.html`
- `history` → `dist/history.html` (full-page history browser)
- `bookmarksGraph` → `dist/bookmarks-graph.html` (bookmark graph visualization)
- `background` → `dist/background.js` (service worker, fixed name for manifest)

A separate `content` mode build produces `dist/contentScript.js` (IIFE, no module).

`manifest.json` references these outputs. If you add an entry point, update both `vite.config.ts` `rollupOptions.input` and `manifest.json`.

## Structure

This extension is a **thin wrapper** around the shared `@browser-server/extension-core` package (`shared/browser-extension-core/`). The wrapper provides a Chrome-specific `BrowserApi` adapter; all business logic, UI components, composables, and background orchestration live in the shared core (used by both the Chromium and Firefox extensions).

```
extension/
├── manifest.json         # MV3 manifest (permissions, omnibox keyword, background, popup, options)
├── popup.html, options.html, history.html, bookmarks-graph.html
├── vite.config.ts
└── src/
    ├── adapter.ts         # ChromeAdapter — implements BrowserApi interface for chrome.* APIs
    ├── background.ts      # Wires ChromeAdapter → initBackground() from extension-core
    └── contentScript.ts   # Wires content script from extension-core
```

### Shared extension core (`shared/browser-extension-core/`)

All real logic lives here and is shared between Chromium and Firefox wrappers:

```
shared/browser-extension-core/src/
├── background.ts          # initBackground(): tab/idle/alarm/omnibox listeners, history + usage sync
├── browserApi.ts          # BrowserApi interface — abstraction over chrome/browser APIs
├── contentScript.ts       # Content script entry
├── composables/           # use<Domain>View() — popup view-model logic
│   ├── useApiClient.ts, useExtensionSettings.ts
│   ├── useTodosView.ts, useBookmarksView.ts, useHistoryView.ts, useWalletView.ts, useAnalyticsView.ts
│   └── useHistoryBrowser.ts
├── popup/                 # PopupApp.vue + domain panels (TodosPanel, BookmarksPanel, etc.)
├── options/               # OptionsApp.vue + main.ts (settings page)
├── history/               # Full-page history browser UI
├── bookmarks-graph/       # Full-page bookmark graph visualization
├── lib/
│   ├── settings.ts        # ExtensionSettings type + storage persistence
│   ├── browser.ts         # BrowserApi wrappers (active tab, screenshot capture, etc.)
│   ├── timeTracker.ts     # Per-domain time accumulation + flush to /api/analytics/usage
│   └── oneClickCapture.ts # Context-menu and keyboard-shortcut capture logic
└── styles/tailwind.css
```

## Conventions

### Architecture: thin wrapper + shared core

The Chromium extension (`extension/`) and Firefox extension (`extension-firefox/`) are both thin wrappers. Each provides:
1. A browser-specific `BrowserApi` adapter (e.g. `ChromeAdapter` using `chrome.*` APIs)
2. An entry point that wires the adapter and calls `initBackground()` from the shared core
3. A manifest and HTML entry points

All business logic, Vue components, composables, and library code live in `shared/browser-extension-core/` and must remain browser-agnostic (access browser APIs only through the `BrowserApi` interface).

### Settings drive everything

User config lives in `ExtensionSettings` (`apiBase`, `apiToken`, `userId`, `autoCapture`), defined in the shared core's [`lib/settings.ts`](../shared/browser-extension-core/src/lib/settings.ts) and persisted to browser storage under the `tracker_settings` key. Always go through `getSettings()` / `saveSettings()`; never read storage ad hoc. When adding a field, update `ExtensionSettings`, `DEFAULT_SETTINGS`, and the options form in `OptionsApp.vue`.

### API access via the shared client

Create a client with the shared core's [`composables/useApiClient.ts`](../shared/browser-extension-core/src/composables/useApiClient.ts):

```ts
createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })
```

The same pattern is used in `background.ts` and `lib/timeTracker.ts`. **Always pass `getToken`** so the `Authorization: Bearer` header is sent — without it, every request gets `401`. New endpoints go in the shared client (`shared/browser-client`), not duplicated here.

### Authentication / token

- The token is entered on the options page (`OptionsApp.vue`, an `apiToken` password field) and stored in settings.
- It is generated server-side via `server token generate` (see root AGENTS.md).
- Screenshot image URLs use the shared client's `getScreenshotUrl`, which appends `?token=` for `<img>` loads.

### Background service worker

The extension's [`background.ts`](src/background.ts) simply wires the `ChromeAdapter` and calls `initBackground()` from the shared core. The actual logic lives in [`shared/browser-extension-core/src/background.ts`](../shared/browser-extension-core/src/background.ts) and is event-driven (no persistent state): it listens to tab, window, idle, and alarm events via the `BrowserApi` interface. History is posted on tab navigation; time usage is buffered by `TimeTracker` and flushed on an alarm. Network failures are swallowed with `console.debug` (server may be offline) — don't throw out of listeners.

### Omnibox search

The manifest defines the Chrome omnibox keyword `bs`. After the user types `bs` and presses Space/Tab, `background.ts` calls `client.searchOmnibox({ user_id, q, limit })`, which hits `GET /api/search/omnibox` with the stored API token. Suggestions must stay clearly labeled by source:
- History suggestions use `[History]` and include the server-side `visit_count`.
- Bookmark suggestions use `[Bookmark]` and can include folder path, tags, or description.

The server balances results across bookmark and history matches when both are present, so the extension should render the returned order directly instead of filtering or re-ranking one source away.

Use Chrome's omnibox XML markup carefully: escape user/server text before inserting it into `<match>` or `<dim>` descriptions. If settings are incomplete or the server is offline, return no suggestions and log with `console.debug`; never throw out of an omnibox listener.

### Popup view-models

Each popup panel has a `use<Domain>View()` composable in the shared core that takes the client + userId refs and exposes view state + actions, mapping domain models to view shapes (e.g. `TodoView`). Keep browser API calls in `lib/browser.ts` (through the `BrowserApi` interface), not in components.

### Styling

TailwindCSS 4 utilities via the shared core's `styles/tailwind.css`; the popup/options use a dark slate theme. Reuse existing panel/option markup patterns rather than introducing new component primitives.

## Modifying the extension

Most changes belong in `shared/browser-extension-core/`, not in this wrapper:

| Change type | Where to edit |
|---|---|
| Add a popup panel / option | `shared/browser-extension-core/src/popup/` or `options/` |
| Add a composable | `shared/browser-extension-core/src/composables/` |
| Change background logic | `shared/browser-extension-core/src/background.ts` |
| Add a new browser API call | `shared/browser-extension-core/src/browserApi.ts` (interface) + `extension/src/adapter.ts` (Chrome impl) + `extension-firefox/src/adapter.ts` (Firefox impl) |
| Add a Vite entry point | `vite.config.ts` `rollupOptions.input` + `manifest.json` |
| Add a new permission | `manifest.json` |

After changes, run `pnpm build` and `pnpm type-check` in this directory to verify.
