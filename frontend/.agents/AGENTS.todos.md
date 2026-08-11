# AGENTS.todos.md — Todos Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers `components/todos/`.

`TodoPage.vue` is thin wiring; all coordination lives in `components/todos/composables/useTodoPage.ts` (view mode, editor modal, screenshot lightbox, delete confirm, kanban/list reorder) on top of `useTodos.ts` (list, filters, sort, CRUD). Presentation constants (priority/status meta, due-date predicates, sort + recurrence options) are centralized in `todoFormat.ts` — never re-declare them per component. The todo editor modal (`todos/editor/`) is owned by the todos domain and reused by the Calendar page.

```
../components/todos/
├── todoFormat.ts              # PRIORITY_META, STATUS_META, due-date predicates, RRULE/SORT options
├── composables/
│   ├── useTodoPage.ts         # Page orchestration for TodoPage.vue
│   ├── useTodos.ts            # List state + filters + CRUD (immediate load on user change)
│   ├── useTodoPriority.ts, useTodoDueDate.ts, useTodoTags.ts  # Filter state only
│   ├── useTodoSort.ts, useTodoSubtasks.ts, useTodoDisplay.ts  # Card-shared display logic
│   └── useTodoListDrag.ts     # vuedraggable (mobile) + native HTML5 rows (desktop)
├── TodoActionsBar.vue         # View toggle, status tabs, filter selects, active chips
├── TodoViewToggle.vue, TodoSortBar.vue, TodoSearchBar.vue, TodoFilterSelect.vue
├── TodoStatusToggle.vue       # Round status checkbox (pending → in_progress → completed)
├── TodoPriorityBadge.vue, TodoDueDateBadge.vue, TodoTagBadges.vue, TodoMetaChips.vue
├── TodoCardActions.vue        # Pin / archive-restore / edit / delete icon actions
├── TodoSubtaskProgress.vue
├── views/                     # TodoListView (table + mobile cards), TodoTableRow, TodoCard,
│                              # TodoKanbanBoard, TodoKanbanCard, TodoGridView, TodoGridCard,
│                              # TodoSubtaskList (inline edit + reorder)
└── editor/                    # TodoEditorModal + TodoEditorForm (also used by CalendarPage)
```
