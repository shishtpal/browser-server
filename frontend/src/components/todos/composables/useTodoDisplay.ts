import type { Todo } from '../../../types';
import { computed, ref, type ComputedRef } from 'vue';
import { getScreenshotUrl } from '../../../lib/api';
import { statusAriaLabel, statusLabel } from '../todoFormat';

/**
 * Shared display logic for the todo card variants (TodoCard, TodoGridCard,
 * TodoKanbanCard). Keeps subtask counts, screenshot URL, and status text in a
 * single place so the cards don't drift apart.
 *
 * Deletion goes straight up through component events — the owning page runs
 * the shared confirm dialog before calling the API.
 */
export function useTodoDisplay(getTodo: () => Todo) {
  const todo: ComputedRef<Todo> = computed(getTodo);

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

  const label = computed(() => statusLabel(todo.value.status));
  const ariaLabel = computed(() => statusAriaLabel(todo.value.status));

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
    statusLabel: label,
    statusAriaLabel: ariaLabel,
    endDateLabel,
  };
}
