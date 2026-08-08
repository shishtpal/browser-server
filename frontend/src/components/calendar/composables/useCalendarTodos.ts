import { computed, ref, watch, type ComputedRef, type Ref } from 'vue';
import { format, startOfDay } from 'date-fns';
import { createTodo, deleteTodo, getTodos, updateTodo } from '../../../lib/api';
import type { CreateTodoInput, Todo } from '../../../types';
import type { CalendarDay, CalendarStats, DateRange } from '../types';

/** Parse a date string that may be ISO 8601 (with T) or plain yyyy-MM-dd. */
export function parseToDate(raw: string): Date {
  return new Date(raw.includes('T') ? raw : raw + 'T00:00:00');
}

/** Normalize a date string (ISO or plain) to yyyy-MM-dd for comparison. */
export function toDateStr(raw: string): string {
  return format(parseToDate(raw), 'yyyy-MM-dd');
}

/**
 * Todos for the calendar: loads the user's full list (filtering to the visible
 * range happens client-side), derives the CalendarDay buckets and header stats,
 * and owns CRUD actions (each mutation refetches).
 *
 * Loading starts automatically (immediate watcher) whenever the user changes.
 */
export function useCalendarTodos(
  selectedUserId: Ref<number | null>,
  dateRange: ComputedRef<DateRange>,
) {
  const todos = ref<Todo[]>([]);
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  const rangeStart = computed(() => startOfDay(dateRange.value.start));
  const rangeEnd = computed(() => startOfDay(dateRange.value.end));

  const visibleTodos = computed(() =>
    todos.value.filter((t) => {
      if (!t.start_date) return false;
      const start = startOfDay(parseToDate(t.start_date));
      return start >= rangeStart.value && start <= rangeEnd.value;
    }),
  );

  const days = computed<CalendarDay[]>(() => {
    const result: CalendarDay[] = [];
    const current = new Date(rangeStart.value);
    const today = new Date();
    // For month view, use the month from the middle of the range as "current month".
    const midDate = new Date(rangeStart.value);
    midDate.setDate(
      midDate.getDate() +
        Math.floor(
          (rangeEnd.value.getTime() - rangeStart.value.getTime()) / (1000 * 60 * 60 * 24) / 2,
        ),
    );
    const viewMonth = midDate.getMonth();
    const viewYear = midDate.getFullYear();

    while (current <= rangeEnd.value) {
      const dateStr = format(current, 'yyyy-MM-dd');
      result.push({
        date: dateStr,
        isToday: isSameDay(current, today),
        isCurrentMonth: current.getMonth() === viewMonth && current.getFullYear() === viewYear,
        isWeekend: current.getDay() === 0 || current.getDay() === 6,
        todos: visibleTodos.value.filter(
          (t) => t.start_date && toDateStr(t.start_date) === dateStr,
        ),
      });
      current.setDate(current.getDate() + 1);
    }
    return result;
  });

  const stats = computed<CalendarStats>(() => {
    const todayStr = format(new Date(), 'yyyy-MM-dd');
    const isPending = (t: Todo) => t.status === 'pending' && Boolean(t.start_date);
    return {
      todayCount: todos.value.filter((t) => isPending(t) && toDateStr(t.start_date!) === todayStr)
        .length,
      overdueCount: todos.value.filter(
        (t) => isPending(t) && startOfDay(parseToDate(t.start_date!)) < startOfDay(new Date()),
      ).length,
      completedCount: todos.value.filter((t) => t.status === 'completed').length,
    };
  });

  const loadTodos = async () => {
    if (!selectedUserId.value) return;
    isLoading.value = true;
    error.value = null;
    try {
      todos.value = await getTodos(selectedUserId.value);
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load todos';
    } finally {
      isLoading.value = false;
    }
  };

  const addTodo = async (data: CreateTodoInput) => {
    try {
      await createTodo(data);
      await loadTodos();
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create todo';
      throw e;
    }
  };

  const updateTodoItem = async (id: number, data: Partial<Todo>) => {
    try {
      await updateTodo(id, data);
      await loadTodos();
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update todo';
      throw e;
    }
  };

  const removeTodo = async (id: number) => {
    try {
      await deleteTodo(id);
      await loadTodos();
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete todo';
      throw e;
    }
  };

  watch(
    selectedUserId,
    (id) => {
      if (id && id > 0) loadTodos();
      else todos.value = [];
    },
    { immediate: true },
  );

  return {
    todos,
    isLoading,
    error,
    visibleTodos,
    days,
    stats,
    loadTodos,
    addTodo,
    updateTodoItem,
    removeTodo,
  };
}

function isSameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}
