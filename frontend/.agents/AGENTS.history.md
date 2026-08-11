# AGENTS.history.md — History Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers `components/history/`.

Both pages are thin wiring over their page composables. Shared helpers live next to the domain; the near-identical Chrome import cards both use `ui/ImportCard.vue`. Delete confirmations go through `@browser-server/shared-modal`'s `useModal().confirmDelete` (never bare `confirm()`).

```
../components/history/
├── composables/
│   ├── useHistoryPage.ts      # Infinite scroll (vueuse IntersectionObserver) + delete confirm
│   └── useHistory.ts          # Paged list + filter + add/delete (immediate load on user change)
├── HistoryAddForm.vue, HistorySearchBar.vue
├── HistoryTableRow.vue (desktop) / HistoryCard.vue (mobile timeline)
└── HistoryImport.vue
```
