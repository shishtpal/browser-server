import { ref, computed, watch, type Ref } from 'vue'
import { getTodos, createTodo, updateTodo, deleteTodo } from '../lib/api'
import type { Todo } from '../types'

export function useTodos(selectedUserId: Ref<number | null>, domainFilter?: Ref<string | null>) {
  const todos = ref<Todo[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const newTitle = ref('')
  const newDescription = ref('')
  const activeFilter = ref<'all' | 'active' | 'completed'>('all')
  const filters = [
    { label: 'All', value: 'all' as const },
    { label: 'Active', value: 'active' as const },
    { label: 'Completed', value: 'completed' as const },
  ]

  const editingId = ref<number | null>(null)
  const editTitle = ref('')
  const editDescription = ref('')

  const activeCount = computed(() => todos.value.filter(t => !t.completed).length)
  const completedCount = computed(() => todos.value.filter(t => t.completed).length)
  const filteredTodos = computed(() => {
    if (activeFilter.value === 'active') return todos.value.filter(t => !t.completed)
    if (activeFilter.value === 'completed') return todos.value.filter(t => t.completed)
    return todos.value
  })

  const loadTodos = async () => {
    if (!selectedUserId.value) return
    isLoading.value = true
    error.value = null
    try {
      todos.value = await getTodos(selectedUserId.value, domainFilter?.value ?? undefined)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load todos'
    } finally {
      isLoading.value = false
    }
  }

  const addTodo = async () => {
    if (!selectedUserId.value || !newTitle.value.trim()) return
    try {
      await createTodo({ user_id: selectedUserId.value, title: newTitle.value.trim(), description: newDescription.value.trim() || undefined })
      newTitle.value = ''
      newDescription.value = ''
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add todo'
    }
  }

  const toggleTodo = async (todo: Todo) => {
    try {
      await updateTodo(todo.id, { ...todo, completed: !todo.completed })
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update todo'
    }
  }

  const startEdit = (todo: Todo) => {
    editingId.value = todo.id
    editTitle.value = todo.title
    editDescription.value = todo.description
  }

  const cancelEdit = () => {
    editingId.value = null
  }

  const saveEdit = async (todo: Todo, title: string, description: string) => {
    try {
      await updateTodo(todo.id, { ...todo, title, description })
      editingId.value = null
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update todo'
    }
  }

  const removeTodo = async (id: number) => {
    try {
      await deleteTodo(id)
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete todo'
    }
  }

  watch(activeFilter, () => {
    if (selectedUserId.value) loadTodos()
  })

  if (domainFilter) {
    watch(domainFilter, () => {
      if (selectedUserId.value) loadTodos()
    })
  }

  return {
    todos,
    isLoading,
    error,
    newTitle,
    newDescription,
    activeFilter,
    filters,
    editingId,
    editTitle,
    editDescription,
    activeCount,
    completedCount,
    filteredTodos,
    loadTodos,
    addTodo,
    toggleTodo,
    startEdit,
    cancelEdit,
    saveEdit,
    removeTodo,
  }
}
