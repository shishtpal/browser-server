# AGENTS.shared-modal.md — Shared Modal Package (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers `@browser-server/shared-modal` (`shared/browser-modal/`).

`@browser-server/shared-modal` is a self-contained imperative dialog service:
`ModalHost.vue` (mounted once via `AppModalHost.vue` in the layout) renders
the module-level request queue from `store.ts`;
`useModal()` exposes `confirm` / `confirmDelete` / `alert`, each returning a promise.
Dialogs queue, trap focus, lock body scroll, honor `persistent`, and theme via the
app's `.dark` class and `bsm-*` CSS in `src/modal.css` (imported once by the
frontend in `src/styles/global.css` via `@import '@browser-server/shared-modal/modal.css'`).
