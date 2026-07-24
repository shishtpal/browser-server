import type { Todo } from '../types'
import { computed, ref, type ComputedRef, type Ref } from 'vue'
import { reorderTodos } from '../lib/api'

export function useTodoReorder(todos: Ref<Todo[]>, loadTodos: () => Promise<void>) {
  const isDragging: Ref<boolean> = ref(false)
  const isReordering: Ref<boolean> = ref(false)

  const droppableList = computed(() => todos.value.map(t => ({ id: t.id })))

  async function onDragEnd(event: { oldIndex: number; newIndex: number }) {
    if (event.oldIndex === event.newIndex) return

    const updated = [...todos.value]
    const [moved] = updated.splice(event.oldIndex, 1)
    updated.splice(event.newIndex, 0, moved)

    const items = updated.map((t, idx) => ({ id: t.id, position: idx }))
    isReordering.value = true
    try {
      await reorderTodos(items)
      await loadTodos()
    } catch (e) {
      await loadTodos()
    } finally {
      isReordering.value = false
      isDragging.value = false
    }
  }

  async function persistOrder(orderedIds: number[]) {
    isReordering.value = true
    try {
      const items = orderedIds.map((id, idx) => ({ id, position: idx }))
      await reorderTodos(items)
      await loadTodos()
    } catch (e) {
      await loadTodos()
    } finally {
      isReordering.value = false
    }
  }

  return {
    isDragging,
    isReordering,
    droppableList,
    onDragEnd,
    persistOrder,
  }
}
