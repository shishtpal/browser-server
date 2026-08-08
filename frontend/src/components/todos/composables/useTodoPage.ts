import type { CreateTodoInput, ReorderItem, Todo, TodoPriority, TodoView } from '../../../types';
import { ref, type Ref } from 'vue';
import { useLocalStorage } from '@vueuse/core';
import { useModal } from '@browser-server/shared-modal';
import { getScreenshotUrl, reorderTodos } from '../../../lib/api';
import { useTodos } from './useTodos';
import { useTodoListDrag } from './useTodoListDrag';

/**
 * Orchestrates the whole Todos page: view mode, the editor modal, the
 * screenshot lightbox, delete confirmation, kanban/list reordering and
 * subtask patching — so TodoPage.vue stays pure wiring.
 * Domain state lives in useTodos.
 */
export function useTodoPage(userId: Ref<number | null>) {
  const todosApi = useTodos(userId);

  const view = useLocalStorage<TodoView>(`bs.todos.view`, 'list');

  const { selectedPriority, clearPriority } = todosApi.priority;
  const { dueDateFilter, clearDueDateFilter } = todosApi.dueDate;
  const { allTags, selectedTag, clearTagFilter } = todosApi.tags;
  const { sortField, sortDir, setSort, toggleDir: toggleSortDir } = todosApi.sort;

  function clearAllFilters() {
    clearPriority();
    clearDueDateFilter();
    clearTagFilter();
  }

  const listDrag = useTodoListDrag(todosApi.displayedTodos, { reload: todosApi.loadTodos });

  /* ------------------------------ kanban moved ---------------------------- */

  async function onKanbanReorder(items: ReorderItem[]) {
    await reorderTodos(items);
    await todosApi.loadTodos();
  }

  async function onKanbanPriorityChange(payload: {
    todo: Todo;
    newPriority: string;
    items: ReorderItem[];
  }) {
    await reorderTodos(payload.items);
    await todosApi.updateTodoItem(payload.todo.id, {
      priority: payload.newPriority as TodoPriority,
    });
  }

  /* ------------------------- subtask toggle patch ------------------------- */

  /** Reflect a toggled subtask onto its parent without a refetch. */
  function onSubtaskToggled(updated: Todo) {
    const parentIndex = todosApi.todos.value.findIndex((t) => t.id === updated.parent_id);
    if (parentIndex === -1) return;
    const parent = todosApi.todos.value[parentIndex];
    todosApi.todos.value[parentIndex] = {
      ...parent,
      subtasks: (parent.subtasks || []).map((s) => (s.id === updated.id ? updated : s)),
    };
  }

  /* ------------------------------ editor modal ---------------------------- */

  const editorOpen = ref(false);
  const editingTodo = ref<Todo | null>(null);
  const editorDueDate = ref('');

  function openCreateModal() {
    if (!userId.value) return;
    editingTodo.value = null;
    editorDueDate.value = '';
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

  /* -------------------------- delete confirmation ------------------------- */

  const { confirmDelete: confirmDeleteModal } = useModal();

  async function confirmDelete(id: number) {
    const confirmed = await confirmDeleteModal(
      'Delete this todo?',
      'This action cannot be undone. The todo and its data will be permanently removed.',
    );
    if (confirmed) await todosApi.removeTodo(id);
  }

  /** Delete pressed from inside the editor: close it, then confirm. */
  async function handleEditorDelete() {
    if (!editingTodo.value) return;
    const id = editingTodo.value.id;
    closeEditor();
    await confirmDelete(id);
  }

  /* --------------------------- screenshot viewer -------------------------- */

  const screenshotModal = ref<{ open: boolean; url: string; title: string }>({
    open: false,
    url: '',
    title: '',
  });

  function openScreenshot(todo: Todo) {
    screenshotModal.value = {
      open: true,
      url: getScreenshotUrl(todo.id),
      title: todo.title,
    };
  }

  function closeScreenshot() {
    screenshotModal.value.open = false;
  }

  return {
    todosApi,
    view,
    // filters (flattened for the template)
    selectedPriority,
    dueDateFilter,
    allTags,
    selectedTag,
    sortField,
    sortDir,
    setSort,
    toggleSortDir,
    clearAllFilters,
    // list drag (mobile vuedraggable + desktop native rows)
    ...listDrag,
    // kanban / subtask events
    onKanbanReorder,
    onKanbanPriorityChange,
    onSubtaskToggled,
    // editor modal
    editorOpen,
    editingTodo,
    editorDueDate,
    openCreateModal,
    openEditModal,
    closeEditor,
    handleCreate,
    handleUpdate,
    handleEditorDelete,
    // delete + screenshot
    confirmDelete,
    screenshotModal,
    openScreenshot,
    closeScreenshot,
  };
}
