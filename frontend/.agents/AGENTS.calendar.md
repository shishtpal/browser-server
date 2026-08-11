# AGENTS.calendar.md — Calendar Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers `components/calendar/`.

`CalendarPage.vue` is thin wiring over `composables/useCalendarPage.ts` (editor + detail modals, drag-move rescheduling, view drill-down) composing `useCalendar.ts` (view/date navigation) and `useCalendarTodos.ts` (day buckets + header stats). Day-cell drop handlers are shared via `useCalendarDragDrop.ts → useCalendarDayDrop()`; month cells reuse `CalendarDayCell.vue` (priority dots on mobile, chips on desktop). Priority/status styling is sourced from `todos/todoFormat.ts`.

```
../components/calendar/
├── types.ts                   # CalendarView, CalendarDay, DateRange, CalendarStats
├── composables/
│   ├── useCalendarPage.ts     # Page orchestration for CalendarPage.vue
│   ├── useCalendar.ts         # currentDate/view/dateRange + navigation
│   ├── useCalendarTodos.ts    # Todos → visible range, per-day buckets, stats (immediate load)
│   └── useCalendarDragDrop.ts # DnD MIME helpers + useCalendarDayDrop() cell handlers
├── CalendarHeader.vue         # Prev/Today/Next + Day/Week/Month/Year switcher
├── CalendarMonthView.vue      # Grid; delegates cells to CalendarDayCell.vue
├── CalendarDayCell.vue        # One month cell (dots on mobile, chips + "+N more" on desktop)
├── CalendarWeekView.vue       # 7 columns, horizontally scrollable on mobile
├── CalendarDayView.vue        # All-day list for the selected date
├── CalendarYearView.vue       # Click-to-edit year header + 12 mini calendars
├── YearMonthCard.vue          # One mini month w/ heatmap days + count badges
├── CalendarTodoChip.vue       # Draggable todo chip
└── CalendarTodoDetail.vue     # Read-only detail modal (edit entry point)
```
