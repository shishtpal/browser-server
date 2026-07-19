# AGENTS.md — Frontend (web app)

Astro + Vue + TailwindCSS web app for Browser Server. This file covers `frontend/`; the [root `AGENTS.md`](../AGENTS.md) covers the Go backend and cross-cutting concerns.

## Tech Stack

- **Astro 6** — routing via file-based pages in `src/pages/`, ships zero JS by default
- **Vue 3** (`<script setup lang="ts">`) — all interactivity lives in Vue islands
- **TailwindCSS 4** — via `@tailwindcss/vite`; global styles in `src/styles/global.css`
- **Shared workspace packages** — `@browser-server/shared-client`, `@browser-server/shared-types`, `@browser-server/shared-utils` (linked from `../shared/*`)

> Note: the package is named `docs-spi` in `package.json` for historical reasons; it is the web frontend.

## Commands

Run from `frontend/` (pnpm workspace):

```bash
pnpm dev       # astro dev (local dev server, default :4321)
pnpm build     # astro build → dist/
pnpm preview   # preview the production build
```

The full release build is driven by `../scripts/build.ps1`, which runs the frontend build and copies `dist/` next to the Go binary for static serving.

## Structure

```
frontend/src/
├── pages/            # Astro routes (.astro) + content (faqs.md). One per nav item.
│   ├── index.astro   # Todos (home)
│   ├── bookmarks.astro, history.astro, wallet.astro, analytics.astro, users.astro
│   ├── about.astro, contact.astro, 404.astro
├── layouts/Layout.astro   # Shared shell: nav, theme, header widgets
├── components/       # Vue components
│   ├── <Domain>Page.vue   # Top-level page component per domain (TodoPage, WalletPage, …)
│   ├── todos/, bookmarks/, history/, wallet/   # Per-domain sub-components
│   ├── ui/           # Reusable presentational components (Button, Modal, InputField, …)
│   ├── ServerStatus.vue, ThemeToggle.vue, ApiTokenSettings.vue   # Header widgets
├── composables/      # use<Domain>() — state + data-loading logic (Vue composition API)
├── lib/
│   ├── api.ts        # App-facing API wrapper (delegates to shared client + raw fetch)
│   ├── auth.ts       # API token storage (localStorage) + authHeaders()
│   └── utils.ts      # App-specific helpers
└── types.ts          # Re-exports @browser-server/shared-types
```

## Conventions

### Astro pages mount Vue islands

Pages are thin: import `Layout` and the domain's `*Page.vue`, mount it with `client:only="vue"`. Don't put logic in `.astro` files.

```astro
---
import TodoPage from '../components/TodoPage.vue'
import Layout from '../layouts/Layout.astro'
---
<Layout title="Todos">
  <main><TodoPage client:only="vue" /></main>
</Layout>
```

### Components use `<script setup lang="ts">`

All Vue components use the composition API with `<script setup>`. Keep page-level state and data loading in a `composables/use<Domain>.ts` and import it into the `*Page.vue` component.

### Composables own data + state

A composable (e.g. [`composables/useTodos.ts`](src/composables/useTodos.ts)) returns `ref`s plus async actions. The standard pattern:
- `items`, `isLoading`, `error` refs
- a `load*()` that sets `isLoading`, calls the API, and traps errors into `error`
- mutating actions (`add*`, `update*`, `remove*`) that call the API then re-`load`
- `watch` user/filter refs to reload

### API access

- Prefer functions exported from [`lib/api.ts`](src/lib/api.ts) — they wrap the shared client (`createBrowserServerClient(API_BASE, { getToken })`).
- New endpoints belong in the **shared client** (`shared/browser-client`) first, then a thin re-export here.
- Any raw `fetch` in `lib/api.ts` MUST include the auth header: `headers: { ...authHeaders() }` (JSON) or `headers: authHeaders()` (GET/DELETE/FormData). Otherwise requests get `401`.
- `API_BASE` is `http://localhost:9191`.

### Authentication / token

- The API token is stored in `localStorage` via [`lib/auth.ts`](src/lib/auth.ts) (`getToken`/`setToken`/`clearToken`/`authHeaders`).
- [`components/ApiTokenSettings.vue`](src/components/ApiTokenSettings.vue) is the header widget for entering/clearing the token; it dispatches an `api-token-changed` event on change.
- Screenshot `<img>` URLs carry the token as a `?token=` query param (the shared client's `getScreenshotUrl` handles this) since image requests can't send headers.

### Styling

- TailwindCSS 4 utility classes; support light/dark via `dark:` variants (theme toggled by `ThemeToggle.vue`, persisted in `localStorage` under `theme`).
- Reuse `components/ui/*` primitives instead of re-styling buttons/inputs/modals.

## Adding a new page

1. Create `components/<Domain>Page.vue` (+ any `components/<domain>/` sub-components).
2. Add `composables/use<Domain>.ts` for state/data.
3. Add API methods to the shared client, then re-export via `lib/api.ts`.
4. Create `pages/<domain>.astro` mounting the page with `client:only="vue"`.
5. Add a nav entry to the `navItems` array in `layouts/Layout.astro`.
