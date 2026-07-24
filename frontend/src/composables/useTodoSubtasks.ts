import type { Todo } from '../types'
import { computed, ref, type ComputedRef, type Ref } from 'vue'
import { getSubtasks, createSubtask, updateTodo, deleteTodo } from '../lib/api'

export function useTodoSubtasks(parentId: Ref<number | null>, userId: Ref<number | null>) {
  const subtasks: Ref<Todo[]> = ref([])
  const isLoading: Ref<boolean> = ref(false)
  const error: Ref<string | null> = ref(null)

  const progress: ComputedRef<{ done: number; total: number }> = computed(() => {
    const total = subtasks.value.length
    const done = subtasks.value.filter(t => t.completed).length
    return { done, total }
  })

  async function loadSubtasks() {
    if (!parentId.value) {
      subtasks.value = []
      return
    }
    isLoading.value = true
    error.value = null
    try {
      subtasks.value = await getSubtasks(parentId.value)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load subtasks'
    } finally {
      isLoading.value = false
    }
  }

  async function addSubtask(title: string) {
    if (!parentId.value || !title.trim() || !userId.value) return
    const todo = await createSubtask(parentId.value, { user_id: userId.value, title: title.trim() })
    subtasks.value.push(todo)
  }

  async function toggleSubtask(subtask: Todo) {
    try {
      const newCompleted = !subtask.completed
      await updateTodo(subtask.id, { completed: newCompleted })
      subtasks.value = subtasks.value.map(t =>
        t.id === subtask.id ? { ...t, completed: newCompleted, updated_at: new Date().toISOString() } : t
      )
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to toggle subtask'
    }
  }

  async function removeSubtask(id: number) {
    try {
      await deleteTodo(id)
      subtasks.value = subtasks.value.filter(t => t.id !== id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete subtask'
    }
  }

  return {
    subtasks,
    isLoading,
    error,
    progress,
    loadSubtasks,
    addSubtask,
    toggleSubtask,
    removeSubtask,
  }
}
