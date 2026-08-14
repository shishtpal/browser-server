# AGENTS.md — Frontend (web app)

Astro + Vue + TailwindCSS web app for Browser Server. This file covers `frontend/`; the [root `AGENTS.md`](../AGENTS.md) covers the Go backend and cross-cutting concerns.

## Tech Stack

- **Astro 7** — routing via file-based pages in `src/pages/`, ships zero JS by default
- **Vue 3.5** (`<script setup lang="ts">`) — all interactivity lives in Vue islands (`client:only="vue"`)
- **TailwindCSS 4** — via `@tailwindcss/vite`; global styles in `src/styles/global.css` (imports the shared-modal `modal.css` and declares an `@source` for the shared markdown renderer's class strings)
- **Icons** — `@lucide/vue`; never hand-write inline SVGs or use emoji glyphs
- **Other deps** — `@vueuse/core`, `date-fns`, `vuedraggable`
- **Shared workspace packages** — `@browser-server/shared-client`, `@browser-server/shared-types`, `@browser-server/shared-utils`, `@browser-server/shared-markdown`, `@browser-server/shared-modal` (linked from `../shared/*`)

## Commands

Run from `frontend/` (pnpm workspace):

```bash
pnpm dev            # astro dev (local dev server, default :4321)
pnpm build          # astro build → dist/
pnpm preview        # preview the production build
pnpm type-check     # vue-tsc --noEmit --skipLibCheck
pnpm astro-check    # pnpm astro check
pnpm format         # oxfmt
pnpm format:check   # oxfmt --check
```

The full release build is driven by `../scripts/build.ps1`, which runs the frontend build and copies `dist/` next to the Go binary for static serving.

```ps1
.\scripts\build.ps1 -Target Frontend
```

## Structure

```
frontend/src/
├── pages/            # Astro routes (.astro) + content (faqs.md). One per nav item.
│   ├── index.astro   # Todos (home)
│   ├── calendar.astro, bookmarks.astro, history.astro, wallet.astro
│   ├── analytics.astro   # Usage
│   ├── chat.astro, image.astro, tasks.astro, ai-monitoring.astro, memory.astro
│   ├── quiz.astro, users.astro, about.astro, contact.astro, 404.astro
│   └── faqs.md
├── layouts/Layout.astro   # Shared shell: primaryNav/secondaryNav, theme, header widgets, AppModalHost
├── components/       # Vue components
│   ├── <Domain>Page.vue   # Top-level page component per domain (TodoPage, WalletPage, ChatPage, …)
│   ├── ai-monitoring/, analytics/, bookmarks/, calendar/, chat/, history/, image/,
│   │   prompts/, quiz/, tasks/, todos/, wallet/   # Per-domain modules (see sub-module guides)
│   ├── ui/           # Reusable primitives (Button, Modal, InputField, SelectField,
│   │                 # PageHeader, StatCard, MultiSelectDropdown, UserSelector, …)
│   ├── AboutUs.vue, ContactUs.vue     # Product/about and contribution/contact pages
│   ├── ServerStatus.vue, ThemeToggle.vue, ApiTokenSettings.vue   # Header widgets
│   └── AppModalHost.vue  # Mounts the shared-modal host
├── composables/      # App-wide composables (useUser, useUsers, usePrompts,
│                     # usePromptManager, useResizableSidebar)
├── lib/
│   ├── api/          # App-facing API layer
│   │   ├── client.ts     # API_BASE, shared client instance, authHeaders
│   │   ├── index.ts      # Barrel — re-exports every domain module
│   │   ├── ai.ts         # AI chat, image, monitoring, tasks endpoints
│   │   ├── health.ts, todos.ts, bookmarks.ts, history.ts, wallet.ts,
│   │   │   users.ts, analytics.ts, memory.ts, prompts.ts, quiz.ts
│   ├── auth.ts       # Disjoint operator/admin token storage + header helpers
│   ├── descriptionLinks.ts  # linkifyDescription / hasLink helpers
│   └── utils.ts      # Re-exports from @browser-server/shared-utils
├── utils/
│   └── copyToClipboard.ts   # Clipboard helper with execCommand fallback
└── types.ts          # Re-exports @browser-server/shared-types
```

## Sub-module guides

- [`AGENTS.conventions.md`](./.agents/AGENTS.conventions.md) — Conventions, API access, auth, styling, and adding a new page
- [`AGENTS.chat.md`](./.agents/AGENTS.chat.md) — AI chat module
- [`AGENTS.prompts.md`](./.agents/AGENTS.prompts.md) — Prompt library (shared by Chat + Image)
- [`AGENTS.quiz.md`](./.agents/AGENTS.quiz.md) — Quiz / exam-prep module
- [`AGENTS.todos.md`](./.agents/AGENTS.todos.md) — Todos module
- [`AGENTS.calendar.md`](./.agents/AGENTS.calendar.md) — Calendar module
- [`AGENTS.bookmarks.md`](./.agents/AGENTS.bookmarks.md) — Bookmarks module
- [`AGENTS.history.md`](./.agents/AGENTS.history.md) — History module
- [`AGENTS.wallet.md`](./.agents/AGENTS.wallet.md) — Wallet module
- [`AGENTS.analytics.md`](./.agents/AGENTS.analytics.md) — Usage / analytics module
- [`AGENTS.image.md`](./.agents/AGENTS.image.md) — Image generation module
- [`AGENTS.tasks.md`](./.agents/AGENTS.tasks.md) — Background AI tasks module
- [`AGENTS.ai-monitoring.md`](./.agents/AGENTS.ai-monitoring.md) — AI request monitoring module
- [`AGENTS.memory.md`](./.agents/AGENTS.memory.md) — Memory graph admin module
- [`AGENTS.shared-modal.md`](./.agents/AGENTS.shared-modal.md) — Shared modal package

## Conventions

See [`AGENTS.conventions.md`](./.agents/AGENTS.conventions.md) for the full conventions, API access rules, authentication/token guidance, styling notes, and the "Adding a new page" checklist.
