# Browser Server Extension

This extension is built with Vite, TypeScript, Vue 3, and Tailwind CSS v4. It records browsing activity to Browser Server and exposes server-side bookmark/history search through Chrome's omnibox.

## Commands

```bash
pnpm install
pnpm dev       # vite build --watch
pnpm build     # production build to extension/dist
pnpm typecheck # vue-tsc --noEmit
```

## Load In Browser

After building, load the unpacked extension from `extension/`.
The root `manifest.json` points Chrome/Edge to the built files under `dist/`.

## Omnibox Search

Type `bs` in Chrome's address bar, press Space or Tab, then enter a query. Suggestions come from `GET /api/search/omnibox`, not Chrome's local browser history, so imported/synced Browser Server history can still appear after clearing browser history.

Results are labeled `[History]` or `[Bookmark]`. History suggestions include the URL visit count recorded in `history.db`.

## Layout

- `src/background.ts` — MV3 service worker, including history sync, usage flushing, and omnibox listeners
- `src/popup/` — Vue 3 popup UI (`PopupApp.vue`, `HistoryPanel.vue`, `TodosPanel.vue`)
- `src/options/` — Vue 3 settings UI (`OptionsApp.vue`)
- `src/composables/` — shared state (settings, history, todos, API client)
- `src/lib/` — chrome helpers, formatting, settings persistence
- `src/styles/tailwind.css` — Tailwind v4 entry, imported by both UIs
- `shared/browser-client/` — reusable typed API client and domain types

## Scope

- Vite multi-entry build for popup, options, and background
- Shared API client under `shared/browser-client`
- Tailwind v4-based popup and options UI
- MV3 service worker bundled from TypeScript
- Chrome omnibox keyword `bs` for server-side bookmark/history suggestions
- Vue 3 SFCs with shared composables instead of imperative DOM code
