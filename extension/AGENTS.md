# AGENTS.md — Browser Extension

Vite + Vue 3 + TailwindCSS Manifest V3 browser extension for Browser Server. This file covers `extension/`; the [root `AGENTS.md`](../AGENTS.md) covers the Go backend and cross-cutting concerns.

## What it does

Captures browsing history and per-domain time usage in real time and syncs them to the local server (`localhost:9191`). Provides a popup UI (todos, history, bookmarks, wallet, analytics views), an options page for settings, and Chrome omnibox suggestions from server-side bookmarks/history.

## Tech Stack

- **Vite 8** — multi-entry build (popup, options, background service worker)
- **Vue 3** (`<script setup lang="ts">`) for popup + options UIs
- **TailwindCSS 4** via `@tailwindcss/vite`; styles in `src/styles/tailwind.css`
- **Manifest V3** — service-worker background script, `chrome.*` APIs
- **Shared workspace packages** — `@browser-server/shared-client`, `@browser-server/shared-types`, `@browser-server/shared-utils`
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

Three entry points, output to `dist/`:
- `popup` → `popup.html` (from `popup.html`)
- `options` → `options.html` (from `options.html`)
- `background` → `dist/background.js` (fixed name, referenced by the manifest service worker)

`manifest.json` references `dist/background.js`, `dist/popup.html`, `dist/options.html`. If you add an entry point, update both `vite.config.ts` `rollupOptions.input` and `manifest.json`.

## Structure

```
extension/
├── manifest.json         # MV3 manifest (permissions, omnibox keyword, background, popup, options)
├── popup.html, options.html
├── vite.config.ts
└── src/
    ├── background.ts      # Service worker: tab/idle/alarm/omnibox listeners, history + usage sync
    ├── popup/            # Popup UI
    │   ├── main.ts, PopupApp.vue
    │   └── <Domain>Panel.vue   # TodosPanel, HistoryPanel, BookmarksPanel, WalletPanel, AnalyticsPanel
    ├── options/          # OptionsApp.vue + main.ts (settings page)
    ├── composables/      # use<Domain>View() — popup view-model logic
    │   ├── useApiClient.ts      # createApiClient(settings) → shared client w/ token
    │   └── useExtensionSettings.ts
    ├── lib/
    │   ├── settings.ts   # ExtensionSettings type + chrome.storage.local persistence
    │   ├── browser.ts    # chrome.* wrappers (active tab, screenshot capture, etc.)
    │   └── timeTracker.ts # Per-domain time accumulation + flush to /api/analytics/usage
    └── styles/tailwind.css
```

## Conventions

### Settings drive everything

User config lives in [`lib/settings.ts`](src/lib/settings.ts) as `ExtensionSettings` (`apiBase`, `apiToken`, `userId`, `autoCapture`), persisted to `chrome.storage.local` under the `tracker_settings` key. Always go through `getSettings()` / `saveSettings()`; never read `chrome.storage` ad hoc. When adding a field, update `ExtensionSettings`, `DEFAULT_SETTINGS`, and the options form in `OptionsApp.vue`.

### API access via the shared client

Create a client with [`composables/useApiClient.ts`](src/composables/useApiClient.ts):

```ts
createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })
```

The same pattern is used directly in `background.ts` and `lib/timeTracker.ts`. **Always pass `getToken`** so the `Authorization: Bearer` header is sent — without it, every request gets `401`. New endpoints go in the shared client (`shared/browser-client`), not duplicated here.

### Authentication / token

- The token is entered on the options page (`OptionsApp.vue`, an `apiToken` password field) and stored in settings.
- It is generated server-side via `server token generate` (see root AGENTS.md).
- Screenshot image URLs use the shared client's `getScreenshotUrl`, which appends `?token=` for `<img>` loads.

### Background service worker

[`background.ts`](src/background.ts) is event-driven (no persistent state): it listens to `chrome.tabs`, `chrome.windows`, `chrome.idle`, and `chrome.alarms`. History is posted on tab navigation; time usage is buffered by `TimeTracker` and flushed on an alarm. Network failures are swallowed with `console.debug` (server may be offline) — don't throw out of listeners.

### Omnibox search

The manifest defines the Chrome omnibox keyword `bs`. After the user types `bs` and presses Space/Tab, `background.ts` calls `client.searchOmnibox({ user_id, q, limit })`, which hits `GET /api/search/omnibox` with the stored API token. Suggestions must stay clearly labeled by source:
- History suggestions use `[History]` and include the server-side `visit_count`.
- Bookmark suggestions use `[Bookmark]` and can include folder path, tags, or description.

The server balances results across bookmark and history matches when both are present, so the extension should render the returned order directly instead of filtering or re-ranking one source away.

Use Chrome's omnibox XML markup carefully: escape user/server text before inserting it into `<match>` or `<dim>` descriptions. If settings are incomplete or the server is offline, return no suggestions and log with `console.debug`; never throw out of an omnibox listener.

### Popup view-models

Each popup panel has a `use<Domain>View()` composable that takes the client + userId refs and exposes view state + actions, mapping domain models to view shapes (e.g. `TodoView`). Keep `chrome.*` calls in `lib/browser.ts`, not in components.

### Styling

TailwindCSS 4 utilities; the popup/options use a dark slate theme. Reuse existing panel/option markup patterns rather than introducing new component primitives.
