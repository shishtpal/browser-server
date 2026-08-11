# AGENTS.bookmarks.md — Bookmarks Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers `components/bookmarks/`.

Both pages are thin wiring over their page composables. Shared helpers live next to the domain; the near-identical Chrome import cards both use `ui/ImportCard.vue`. Delete confirmations go through `@browser-server/shared-modal`'s `useModal().confirmDelete` (never bare `confirm()`).

```
../components/bookmarks/
├── bookmarkFormat.ts          # Search columns + parse/format helpers (host, initials, tag matching)
├── composables/
│   ├── useBookmarkPage.ts     # Page wiring: edit modal + delete confirm + tree view state
│   ├── useBookmarks.ts        # List + tag filter + search + CRUD (immediate load on user change)
│   └── useBookmarkTree.ts     # folder_path → tree edges + expansion
├── BookmarkForm.vue           # Quick-add form (local state, emits payload)
├── BookmarkSearchBar.vue      # Column select + query + flat/tree toggle + counts
├── BookmarkTagFilter.vue      # Scrollable tag pills + active-filter banner
├── BookmarkEditModal.vue, BookmarkTag.vue, BookmarkImport.vue
├── BookmarkTreeView.vue / BookmarkTreeNode.vue
└── views/                     # BookmarkFlatView = BookmarkTableRow (desktop) + BookmarkCard (mobile)
```
