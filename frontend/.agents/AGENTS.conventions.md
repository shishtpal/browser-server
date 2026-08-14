# AGENTS.conventions.md — Frontend Conventions (Web App)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers conventions, API access, authentication/token handling, styling, and the "Adding a new page" checklist for the Astro + Vue frontend.

## Astro pages mount Vue islands

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

Static pages (`about.astro`, `contact.astro`, `404.astro`, `faqs.md`) may use plain Astro markup; `ContactUs.astro` is an Astro component, everything else is a Vue island.

## Components use `<script setup lang="ts">`

All Vue components use the composition API with `<script setup>`. Keep page-level state and data loading in a composable and import it into the `*Page.vue` component.

## Composables own data + state

Two locations, same pattern:

- **Domain-scoped** composables live next to their module: `components/<domain>/composables/` (e.g. [`components/todos/composables/useTodos.ts`](../src/components/todos/composables/useTodos.ts)). Each page has a `use<Domain>Page.ts` that orchestrates the page (modals, confirmations, navigation) over one or more data composables.
- **App-wide** composables (user session, prompt library) live in [`src/composables/`](../src/composables/) (`useUser`, `useUsers`, `usePrompts`, `usePromptManager`, `useResizableSidebar`).

The standard pattern:

- `items`, `isLoading`, `error` refs
- a `load*()` that sets `isLoading`, calls the API, and traps errors into `error`
- mutating actions (`add*`, `update*`, `remove*`) that call the API then re-`load`
- `watch` user/filter refs to reload

## API access

- Prefer functions exported from [`lib/api`](../src/lib/api/index.ts) — the barrel re-exports every domain module (`todos.ts`, `bookmarks.ts`, …).
- [`lib/api/client.ts`](../src/lib/api/client.ts) owns the shared client: `API_BASE` (derived from `window.location` origin, fallback `http://localhost:9191`), the `createBrowserServerClient(API_BASE, { getToken })` instance, and `authHeaders`.
- New endpoints belong in the **shared client** (`shared/browser-client`) first, then a thin re-export in `lib/api/<domain>.ts` plus an entry in `lib/api/index.ts`.
- Any raw `fetch` in `lib/api/*` MUST include the correct auth header. Ordinary modules use `authHeaders()`; `admin.ts` is the deliberate exception and uses `adminHeaders()` so the disjoint Project Settings credential never flows to ordinary routes.

## Authentication / token

- The API token is stored in `localStorage` via [`lib/auth.ts`](../src/lib/auth.ts) (`getToken`/`setToken`/`clearToken`/`hasToken`/`authHeaders`); any change dispatches an `api-token-changed` event.
- [`components/ApiTokenSettings.vue`](../src/components/ApiTokenSettings.vue) is the header widget for entering/clearing the token; pages and widgets that cache API data (e.g. `useMonitoringPage`) listen for `api-token-changed` to reload.
- Screenshot `<img>` URLs carry the token as a `?token=` query param (the shared client's `getScreenshotUrl` / `getGeneratedImageUrl` handle this) since image requests can't send headers.

## Styling

- TailwindCSS 4 utility classes; support light/dark via `dark:` variants (theme toggled by `ThemeToggle.vue`, persisted in `localStorage` under `theme`; the `@custom-variant dark` is defined in `global.css`).
- Icons come from `@lucide/vue` — never hand-write inline SVGs or use emoji glyphs.
- Reuse `components/ui/*` primitives (`Button`, `Modal`, `InputField`, `SelectField`, `TextAreaField`, `PageHeader`, `StatCard`, `MultiSelectDropdown`, `UserSelector`, `SearchableSelect`, …) instead of re-styling buttons/inputs/modals.
- Delete confirmations go through `@browser-server/shared-modal`'s `useModal().confirmDelete` (never bare `confirm()`); see [`AGENTS.shared-modal.md`](./AGENTS.shared-modal.md).

## Adding a new page

1. Create `components/<Domain>Page.vue` (+ any `components/<domain>/` sub-components).
2. Add `components/<domain>/composables/use<Domain>Page.ts` (and data composables) for state/data; keep the page component thin wiring.
3. Add API methods to the shared client, then re-export via `lib/api/<domain>.ts` and add it to `lib/api/index.ts`.
4. Create `pages/<domain>.astro` mounting the page with `client:only="vue"`.
5. Add a nav entry to the `primaryNav` (or `secondaryNav`) arrays in `layouts/Layout.astro`; SVG icons for primary items are inlined per-item in the template.
