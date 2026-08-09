# @browser-server/shared-modal

Imperative modal/confirm service for Browser Server frontends.

## Usage

Mount the host once at the app root (the frontend does this via `AppModalHost.vue` in the layout):

```vue
<script setup lang="ts">
import { ModalHost } from '@browser-server/shared-modal';
</script>

<template>
  <ModalHost />
</template>
```

Then request dialogs from any component or composable:

```ts
import { useModal } from '@browser-server/shared-modal';

const { confirm, confirmDelete, alert } = useModal();

// Resolves true/false; Escape/backdrop resolve false (unless `persistent`).
if (await confirm('Sync now?', 'Local changes will be pushed.')) { … }

// Destructive (red) variant.
if (await confirmDelete('Delete this todo?', 'This cannot be undone.')) { … }

// Single OK button; resolves when dismissed.
await alert('Import complete', '42 bookmarks were imported.');
```

### Options (`ModalOptions`)

| Option        | Default                           | Purpose                                   |
| ------------- | --------------------------------- | ----------------------------------------- |
| `cancelText`  | `"Cancel"`                        | Cancel button label (confirm/danger only) |
| `confirmText` | `"Confirm"` / `"Delete"` / `"OK"` | Primary button label (per dialog kind)    |
| `persistent`  | `false`                           | Disable Escape/backdrop dismissal         |
| `panelClass`  | —                                 | Extra class on the dialog panel           |

## Structure

```
src/
├── index.ts          # public exports
├── types.ts          # ModalKind / ModalOptions / ModalRequest / ModalApi
├── store.ts          # reactive request queue (module-level singleton)
├── useModal.ts       # confirm / confirmDelete / alert
├── ModalHost.vue     # teleports the active dialog, focus trap, scroll lock, transitions
├── ConfirmDialog.vue # the dialog UI (icon per kind, actions)
└── modal.css         # self-contained styles (also importable for pre-bundling)
```

Styles are scoped under `bsm-*` and theme-aware via the app's `.dark` class. The host self-imports `modal.css`, so apps don't strictly need the explicit CSS import (kept for compatibility).

## Notes

- Dialogs queue: only the first unresolved request is visible at a time; subsequent calls stack.
- `useModal()` is safe to call from composables (no inject/provide — a module-level store is shared).
- `pendingCount()` returns how many dialogs are awaiting answers (debugging/tests).
