import type {
  DueDateFilter,
  Todo,
  TodoFilter,
  TodoPriority,
  TodoSortField,
  TodoStatus,
} from '../../types';

/* ------------------------------------------------------------------ */
/* Priority — single source of truth (previously duplicated in         */
/* useTodoPriority.ts, TodoPriorityBadge.vue, the calendar chip and    */
/* priority-dot maps, which had already drifted apart).                 */
/* ------------------------------------------------------------------ */

export const PRIORITY_ORDER: TodoPriority[] = ['low', 'medium', 'high', 'urgent'];

export interface PriorityMeta {
  label: string;
  /** Badge background/text classes */
  badgeClass: string;
  /** Solid dot / accent classes */
  dotClass: string;
  /** Sort weight (urgent first) */
  weight: number;
  /** Calendar chip background classes. */
  chipClass: string;
}

export const PRIORITY_META: Record<TodoPriority, PriorityMeta> = {
  low: {
    label: 'Low',
    badgeClass: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400',
    dotClass: 'bg-emerald-500',
    weight: 3,
    chipClass:
      'border-slate-200/60 bg-slate-100/80 text-slate-700 hover:border-slate-300 dark:border-slate-700/60 dark:bg-slate-800/60 dark:text-slate-300 dark:hover:border-slate-600',
  },
  medium: {
    label: 'Medium',
    badgeClass: 'bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400',
    dotClass: 'bg-blue-500',
    weight: 2,
    chipClass:
      'border-blue-200/60 bg-blue-50/80 text-blue-900 hover:border-blue-300 dark:border-blue-900/30 dark:bg-blue-950/30 dark:text-blue-200 dark:hover:border-blue-800/60',
  },
  high: {
    label: 'High',
    badgeClass: 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400',
    dotClass: 'bg-amber-500',
    weight: 1,
    chipClass:
      'border-amber-200/60 bg-amber-50/80 text-amber-900 hover:border-amber-300 dark:border-amber-900/30 dark:bg-amber-950/30 dark:text-amber-200 dark:hover:border-amber-800/60',
  },
  urgent: {
    label: 'Urgent',
    badgeClass: 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400',
    dotClass: 'bg-red-500',
    weight: 0,
    chipClass:
      'border-red-200/60 bg-red-50/80 text-red-900 hover:border-red-300 dark:border-red-900/30 dark:bg-red-950/30 dark:text-red-200 dark:hover:border-red-800/60',
  },
};

export const priorityMeta = (p: TodoPriority | string): PriorityMeta =>
  PRIORITY_META[p as TodoPriority] ?? PRIORITY_META.medium;

export const priorityWeight = (p: TodoPriority | string): number => priorityMeta(p).weight;

/** Priority dot color; muted when the todo is completed or archived. */
export function todoDotClass(todo: Pick<Todo, 'priority' | 'status'>): string {
  if (todo.status === 'completed' || todo.status === 'archived')
    return 'bg-slate-300 dark:bg-slate-600';
  return priorityMeta(todo.priority).dotClass;
}

/** Calendar chip background; muted when completed. */
export function todoChipClass(todo: Pick<Todo, 'priority' | 'status'>): string {
  if (todo.status === 'completed') {
    return 'border-transparent bg-slate-100/60 text-slate-400 dark:bg-slate-800/40 dark:text-slate-500';
  }
  return priorityMeta(todo.priority).chipClass;
}

/* ------------------------------------------------------------------ */
/* Status                                                              */
/* ------------------------------------------------------------------ */

export const STATUS_META: Record<TodoStatus, { label: string; badgeClass: string }> = {
  pending: {
    label: 'Pending',
    badgeClass: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
  },
  in_progress: {
    label: 'In Progress',
    badgeClass: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  },
  completed: {
    label: 'Completed',
    badgeClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
  },
  archived: {
    label: 'Archived',
    badgeClass: 'bg-slate-100 text-slate-500 dark:bg-slate-700 dark:text-slate-400',
  },
};

/** Status cycle on toggle: pending → in_progress → completed → pending. */
export const NEXT_STATUS: Record<string, TodoStatus> = {
  pending: 'in_progress',
  in_progress: 'completed',
  completed: 'pending',
};

export const statusLabel = (status: TodoStatus | string): string =>
  STATUS_META[status as TodoStatus]?.label ?? status;

export const statusAriaLabel = (status: TodoStatus): string => {
  if (status === 'archived') return 'Archived todo';
  if (status === 'completed') return 'Mark as active';
  if (status === 'in_progress') return 'Mark as completed';
  return 'Mark as in progress';
};

export const STATUS_FILTERS: { label: string; value: TodoFilter }[] = [
  { label: 'All', value: 'all' },
  { label: 'Active', value: 'active' },
  { label: 'In Progress', value: 'in_progress' },
  { label: 'Completed', value: 'completed' },
  { label: 'Archived', value: 'archived' },
];

/* ------------------------------------------------------------------ */
/* Due date predicates + labels (from useTodoDueDate)                  */
/* ------------------------------------------------------------------ */

type DatedTodo = Pick<Todo, 'start_date' | 'status'>;

const isActive = (t: DatedTodo) => t.status !== 'completed' && t.status !== 'archived';

export function isOverdue(todo: DatedTodo): boolean {
  if (!todo.start_date || !isActive(todo)) return false;
  return new Date(todo.start_date) < new Date(new Date().toDateString());
}

export function isDueToday(todo: DatedTodo): boolean {
  if (!todo.start_date || !isActive(todo)) return false;
  return new Date(todo.start_date).toDateString() === new Date().toDateString();
}

export function isDueThisWeek(todo: DatedTodo): boolean {
  if (!todo.start_date || !isActive(todo)) return false;
  const due = new Date(todo.start_date);
  const now = new Date();
  const weekEnd = new Date(now);
  weekEnd.setDate(now.getDate() + ((7 - now.getDay()) % 7));
  return due >= now && due <= weekEnd;
}

export const DUE_DATE_FILTERS: { value: Exclude<DueDateFilter, null>; label: string }[] = [
  { value: 'overdue', label: 'Overdue' },
  { value: 'today', label: 'Due today' },
  { value: 'this_week', label: 'This week' },
];

export const DUE_DATE_FILTER_LABELS: Record<string, string> = {
  overdue: 'Overdue',
  today: 'Today',
  this_week: 'This week',
};

export function matchesDueDateFilter(todo: Todo, filter: DueDateFilter): boolean {
  switch (filter) {
    case 'overdue':
      return isOverdue(todo);
    case 'today':
      return isDueToday(todo);
    case 'this_week':
      return isDueThisWeek(todo);
    default:
      return true;
  }
}

export function dueDateBadgeClass(todo: DatedTodo): string {
  if (!isActive(todo)) return 'bg-gray-100 text-gray-500 dark:bg-slate-700 dark:text-slate-400';
  if (isOverdue(todo)) return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400';
  if (isDueToday(todo))
    return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400';
  if (isDueThisWeek(todo))
    return 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-400';
  return 'bg-gray-100 text-gray-600 dark:bg-slate-700 dark:text-slate-400';
}

/** "Overdue" | "Today" | "This week" | localized date | null. */
export function dueDateLabel(todo: DatedTodo): string | null {
  if (!todo.start_date) return null;
  if (isOverdue(todo)) return 'Overdue';
  if (isDueToday(todo)) return 'Today';
  if (isDueThisWeek(todo)) return 'This week';
  return new Date(todo.start_date).toLocaleDateString();
}

/* ------------------------------------------------------------------ */
/* Recurrence                                                          */
/* ------------------------------------------------------------------ */

export const RRULE_OPTIONS: { value: string; label: string }[] = [
  { value: '', label: 'None' },
  { value: 'FREQ=DAILY', label: 'Daily' },
  { value: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR', label: 'Every Weekday' },
  { value: 'FREQ=WEEKLY', label: 'Weekly' },
  { value: 'FREQ=WEEKLY;INTERVAL=2', label: 'Every 2 Weeks' },
  { value: 'FREQ=MONTHLY', label: 'Monthly' },
  { value: 'FREQ=YEARLY', label: 'Yearly' },
  { value: 'custom', label: 'Custom…' },
];

export function formatRrule(rrule: string): string {
  return RRULE_OPTIONS.find((o) => o.value === rrule)?.label ?? rrule;
}

/* ------------------------------------------------------------------ */
/* Sorting                                                             */
/* ------------------------------------------------------------------ */

export const SORT_OPTIONS: { value: TodoSortField; label: string }[] = [
  { value: 'position', label: 'Position' },
  { value: 'priority', label: 'Priority' },
  { value: 'start_date', label: 'Date' },
  { value: 'created_at', label: 'Created' },
  { value: 'title', label: 'Title' },
];
