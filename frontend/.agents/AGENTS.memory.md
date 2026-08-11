# AGENTS.memory.md — Memory Graph Admin Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers the memory admin page in `components/MemoryPage.vue`.

`MemoryPage.vue` is self-contained — it manages the whole memory-graph admin UI inline (no separate page composable): an interactive SVG graph canvas (pan/zoom, draggable nodes, kind-colored nodes, edge labels by relationship), a fragment list/search with kind filter, stats (`fragments`, `edges`), and a create/edit form for fragments (title, kind, status, parent, tags, summary, markdown body) including link management (add/remove non-`child_of` links) and the **Run maintenance** action.

```
../components/MemoryPage.vue     # Graph canvas + fragment editor (self-contained)
```

Data comes from `lib/api/memory.ts`, backed by the shared client's `/api/ai/memory` endpoints: `getAIMemoryStats`, `getAIMemoryGraph`, `getAIMemoryFragment`, `writeAIMemory` (batched ops), and `maintainAIMemory`. This UI mirrors the agent-facing memory system the Go backend exposes.
