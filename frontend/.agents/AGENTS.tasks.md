# AGENTS.tasks.md — Background AI Tasks Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers the background agent tasks page in `components/tasks/`.

`TasksPage.vue` is thin wiring over `tasks/composables/useTasksPage.ts`, which submits/cancels/deletes tasks with shared-modal confirmations. The data layer is `useAITasks.ts`: it loads the task list and status (`enabled`, `workers`, per-status `counts`) and polls every 3 seconds **only while** any task is queued or running — the server has no WS stream, so progress is visible only by re-reading the list. The filter pills (All / Queued / Running / Completed / Failed) drive a `hasActive` live badge. When `tasks.enabled` is false in `bs-ai-config.json`, the page explains the fix instead of showing an empty queue.

```
../components/tasks/
├── taskFormat.ts               # Status meta, labels, colors
├── composables/
│   ├── useTasksPage.ts         # Page orchestration: submit/cancel/delete confirm, initial load
│   └── useAITasks.ts           # List + status polling (3s while active) + filters + counts
├── TaskSubmitForm.vue          # New-task form (worker select, prompt)
├── TaskFilterBar.vue           # Status filter pills with counts + live badge
└── TaskCard.vue                # Task row: status, progress, cancel/delete actions
```

Data comes from `lib/api/ai.ts` — `listAITasks`, `getAITaskStatus`, `createAITask`, `cancelAITask`, `deleteAITask` — backed by the shared client's `/api/ai/tasks` endpoints.
