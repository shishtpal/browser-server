import { computed, ref } from 'vue';
import type { Todo } from '../types';
import { getScreenshotUrl } from '../lib/api';

export type TodoDeleteHandler = (id: number) => void;

/**
 * Shared display logic for the todo card components (TodoCard, TodoGridCard,
 * TodoKanbanCard). Keeps status, subtask, screenshot, and delete behavior in a
 * single place so the card variants don't drift apart.
 *
 * @param getTodo getter returning the current todo prop (reactive-safe when the
 *   parent replaces the todo object, e.g. on refetch)
 * @param onDelete optional callback invoked after the user confirms deletion
 */
export function useTodoDisplay(getTodo: () => Todo, onDelete?: TodoDeleteHandler) {
  const todo = computed(getTodo);

  const screenshotUrl = computed(() =>
    todo.value.screenshot_path ? getScreenshotUrl(todo.value.id) : '',
  );

  const subtaskCount = computed(() => (todo.value.subtasks || []).length);
  const subtaskDoneCount = computed(
    () => (todo.value.subtasks || []).filter((s) => s.status === 'completed').length,
  );

  const showSubtasks = ref(false);

  function toggleSubtaskVisibility() {
    showSubtasks.value = !showSubtasks.value;
  }

  function confirmDelete() {
    if (window.confirm(`Delete "${todo.value.title}"?`)) {
      onDelete?.(todo.value.id);
    }
  }

  const statusLabel = computed(() => {
    switch (todo.value.status) {
      case 'pending':
        return 'Pending';
      case 'in_progress':
        return 'In Progress';
      case 'completed':
        return 'Completed';
      case 'archived':
        return 'Archived';
      default:
        return todo.value.status;
    }
  });

  const statusAriaLabel = computed(() => {
    if (todo.value.status === 'archived') return 'Archived todo';
    if (todo.value.status === 'completed') return 'Mark as active';
    if (todo.value.status === 'in_progress') return 'Mark as completed';
    return 'Mark as in progress';
  });

  const statusToggleClass = computed(() => {
    if (todo.value.status === 'completed') return 'border-emerald-500 bg-emerald-500 text-white';
    if (todo.value.status === 'in_progress') return 'border-blue-500 bg-blue-500 text-white';
    return 'border-gray-300 text-transparent hover:border-indigo-400 dark:border-slate-600 dark:hover:border-indigo-400';
  });

  const endDateLabel = computed(() => {
    if (!todo.value.end_date) return '';
    return new Date(todo.value.end_date).toLocaleDateString();
  });

  return {
    screenshotUrl,
    subtaskCount,
    subtaskDoneCount,
    showSubtasks,
    toggleSubtaskVisibility,
    confirmDelete,
    statusLabel,
    statusAriaLabel,
    statusToggleClass,
    endDateLabel,
  };
}
