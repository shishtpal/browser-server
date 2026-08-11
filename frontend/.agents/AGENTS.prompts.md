# AGENTS.prompts.md — Prompt Library Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers the prompt library in `components/prompts/`, shared by the **Chat** and **Image** pages (both mount `PromptManager.vue`; the Chat top bar exposes the toggle).

State is split into two app-wide composables in `src/composables/`:

- [`usePrompts.ts`](../src/composables/usePrompts.ts) — per-user prompt CRUD + search + tag/untagged filters (takes the `currentUserId` ref; immediate load on user change).
- [`usePromptManager.ts`](../src/composables/usePromptManager.ts) — UI orchestration: grid/editor view switching, search/sort/layout state, draft editing with dirty tracking and tags, and persistence actions. All data access is **injected** through `PromptManagerDeps` so the composable stays free of API concerns.

```
../components/prompts/
├── format.ts                   # Prompt formatters/helpers
├── PromptManager.vue           # Root shell (grid/editor) — used by Chat + Image pages
├── PromptManagerShell.vue, PromptManagerHeader.vue
├── PromptToolbar.vue           # Search + sort + layout toggle
├── PromptGrid.vue, PromptCard.vue
├── PromptEditor.vue            # Draft editing with dirty tracking, tags
├── PromptTagSidebar.vue        # Tag filter with counts (+ Untagged)
└── PromptSearchDropdown.vue    # Quick search dropdown (Chat input)
```

Data comes from `lib/api/prompts.ts` — raw `fetch` to `/api/prompts` (GET/POST/PUT/DELETE) with `authHeaders()` — since prompts are not part of the shared client.
