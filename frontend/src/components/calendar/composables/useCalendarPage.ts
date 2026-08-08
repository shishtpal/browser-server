import type { CreateTodoInput, Todo } from '../../../types';
import { format } from 'date-fns';
import { computed, ref, type Ref } from 'vue';
import { useCalendar } from './useCalendar';
import { useCalendarTodos } from './useCalendarTodos';

/**
 * Orchestrates the Calendar page: view navigation + drill-down, the todo
 * editor modal, the read-only detail modal, and drag-move rescheduling —
 * so CalendarPage.vue stays pure wiring.
 * Domain state lives in useCalendar / useCalendarTodos.
 */
export function useCalendarPage(userId: Ref<number | null>) {
  const calendar = useCalendar();
  const todosApi = useCalendarTodos(userId, calendar.dateRange);

  /* ------------------------------ editor modal ----------------------------- */

  const editorOpen = ref(false);
  const editingTodo = ref<Todo | null>(null);
  const editorDueDate = ref('');

  function openCreateModal(date?: string) {
    editingTodo.value = null;
    editorDueDate.value = date || format(new Date(), 'yyyy-MM-dd');
    editorOpen.value = true;
  }

  function openEditModal(todo: Todo) {
    editingTodo.value = todo;
    editorDueDate.value = todo.start_date || '';
    editorOpen.value = true;
  }

  function closeEditor() {
    editorOpen.value = false;
    editingTodo.value = null;
  }

  async function handleCreate(data: CreateTodoInput) {
    await todosApi.addTodo(data);
  }

  async function handleUpdate(id: number, data: Partial<Todo>) {
    await todosApi.updateTodoItem(id, data);
  }

  async function handleDelete() {
    if (!editingTodo.value) return;
    await todosApi.removeTodo(editingTodo.value.id);
    closeEditor();
  }

  /* ------------------------------ detail modal ----------------------------- */

  /** Clicking a calendar chip opens the read-only detail first. */
  const detailTodo = ref<Todo | null>(null);

  function openDetail(todo: Todo) {
    detailTodo.value = todo;
  }

  function closeDetail() {
    detailTodo.value = null;
  }

  function editFromDetail(todo: Todo) {
    detailTodo.value = null;
    openEditModal(todo);
  }

  /* ------------------------------- drag move ------------------------------- */

  async function handleTodoMove(payload: { todo: Todo; date: string }) {
    await todosApi.updateTodoItem(payload.todo.id, { start_date: payload.date });
  }

  /* ------------------------------ view helpers ----------------------------- */

  /** Day view gets just the current day's bucket. */
  const currentDayData = computed(() => {
    if (calendar.view.value !== 'day') return undefined;
    const dateStr = format(calendar.currentDate.value, 'yyyy-MM-dd');
    return todosApi.days.value.find((d) => d.date === dateStr) ?? todosApi.days.value[0];
  });

  return {
    calendar,
    todosApi,
    // editor
    editorOpen,
    editingTodo,
    editorDueDate,
    openCreateModal,
    openEditModal,
    closeEditor,
    handleCreate,
    handleUpdate,
    handleDelete,
    // detail
    detailTodo,
    openDetail,
    closeDetail,
    editFromDetail,
    // move + view data
    handleTodoMove,
    currentDayData,
  };
}
