import type { Todo } from '../../../types';
import { ref, watch, type Ref } from 'vue';
import { reorderTodos } from '../../../lib/api';

interface DragEndEvent {
  oldIndex: number;
  newIndex: number;
}

/**
 * List-view dragging for the todos page:
 *  - `listTodos` mirrors the (sorted+filtered) source so vuedraggable can mutate it on mobile
 *  - desktop table rows use native HTML5 drag gated by the `.drag-handle` grip
 * Both persist through the provided `persist` (reorder) + `reload` callbacks.
 */
export function useTodoListDrag(
  sourceTodos: Readonly<Ref<Todo[]>>,
  options: { reload: () => void | Promise<void> },
) {
  const listTodos: Ref<Todo[]> = ref([]);

  watch(
    sourceTodos,
    (val) => {
      listTodos.value = [...val];
    },
    { immediate: true },
  );

  const persistOrder = async () => {
    try {
      await reorderTodos(listTodos.value.map((t, idx) => ({ id: t.id, position: idx })));
    } finally {
      // Refetch regardless of outcome so the list can never drift from the server.
      await options.reload();
    }
  };

  /** vuedraggable `end` (mobile card list). */
  async function onDragEnd(event: DragEndEvent) {
    if (event.oldIndex === event.newIndex) return;
    await persistOrder();
  }

  /* ----------------- native HTML5 drag for desktop table rows ----------------- */

  const dragId = ref<number | null>(null);
  const dragAllowed = ref(false);

  function onRowMouseDown(event: MouseEvent) {
    // Allow drag only when initiated from the drag handle.
    dragAllowed.value = !!(event.target && (event.target as HTMLElement).closest('.drag-handle'));
  }

  function onRowDragStart(event: DragEvent, id: number) {
    if (!dragAllowed.value) {
      event.preventDefault();
      return;
    }
    dragId.value = id;
    if (event.dataTransfer) {
      event.dataTransfer.setData('text/plain', String(id));
      event.dataTransfer.effectAllowed = 'move';
    }
  }

  function onRowDragOver(event: DragEvent, id: number) {
    if (dragId.value === null || dragId.value === id) return;
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
  }

  function onRowDrop(_event: DragEvent, id: number) {
    if (dragId.value === null || dragId.value === id) {
      dragId.value = null;
      return;
    }
    const fromIdx = listTodos.value.findIndex((t) => t.id === dragId.value);
    const toIdx = listTodos.value.findIndex((t) => t.id === id);
    dragId.value = null;
    if (fromIdx === -1 || toIdx === -1) return;
    const moved = listTodos.value.splice(fromIdx, 1)[0];
    listTodos.value.splice(toIdx, 0, moved);
    persistOrder();
  }

  function onRowDragEnd() {
    dragId.value = null;
    dragAllowed.value = false;
  }

  return {
    listTodos,
    onDragEnd,
    onRowMouseDown,
    onRowDragStart,
    onRowDragOver,
    onRowDrop,
    onRowDragEnd,
  };
}
