import { ref, computed, watch, type Ref } from 'vue'
import { getTodos, createTodo, updateTodo, deleteTodo } from '../lib/api'
import { useTodoPriority } from './useTodoPriority'
import { useTodoDueDate } from './useTodoDueDate'
import { useTodoTags } from './useTodoTags'
import { useTodoSort } from './useTodoSort'
import { useTodoSubtasks } from './useTodoSubtasks'
import { useTodoReorder } from './useTodoReorder'
import { isOverdue, isDueToday, isDueThisWeek } from './useTodoDueDate'
import type { Todo } from '../types'

export function useTodos(selectedUserId: Ref<number | null>, domainFilter?: Ref<string | null>) {
  const todos = ref<Todo[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const activeFilter = ref<'all' | 'active' | 'completed' | 'archived'>('all')
  const searchQuery = ref('')
  const filters = [
    { label: 'All', value: 'all' as const },
    { label: 'Active', value: 'active' as const },
    { label: 'Completed', value: 'completed' as const },
    { label: 'Archived', value: 'archived' as const },
  ]

  const totalCount = computed(() => todos.value.filter(t => t.status !== 'archived').length)
  const activeCount = computed(() => todos.value.filter(t => t.status === 'pending').length)
  const completedCount = computed(() => todos.value.filter(t => t.status === 'completed').length)
  const archivedCount = computed(() => todos.value.filter(t => t.status === 'archived').length)
  const overdueCount = computed(() => todos.value.filter(t => t.status === 'pending' && t.start_date && isOverdue(t)).length)

  const priority = useTodoPriority()
  const dueDate = useTodoDueDate()
  const tags = useTodoTags(todos)

  const baseFiltered = computed(() => {
    let list = todos.value
    if (activeFilter.value === 'archived') {
      list = list.filter(t => t.status === 'archived')
    } else {
      list = list.filter(t => t.status !== 'archived')
    }
    const query = searchQuery.value.trim().toLowerCase()
    if (query) {
      list = list.filter(t =>
        t.title.toLowerCase().includes(query)
        || t.description?.toLowerCase().includes(query)
        || (t.tags || []).some(tag => tag.toLowerCase().includes(query)),
      )
    }
    if (priority.selectedPriority.value) {
      list = list.filter(t => t.priority === priority.selectedPriority.value)
    }
    if (dueDate.dueDateFilter.value) {
      list = list.filter(t => {
        switch (dueDate.dueDateFilter.value) {
          case 'overdue': return isOverdue(t)
          case 'today': return isDueToday(t)
          case 'this_week': return isDueThisWeek(t)
        }
        return true
      })
    }
    if (tags.selectedTag.value) {
      list = list.filter(t => (t.tags || []).includes(tags.selectedTag.value!))
    }
    if (activeFilter.value === 'active') list = list.filter(t => t.status === 'pending')
    if (activeFilter.value === 'completed') list = list.filter(t => t.status === 'completed')
    return list
  })

  const sort = useTodoSort(baseFiltered)
  const displayedTodos = sort.sorted

  const expandedTodoId = ref<number | null>(null)
  const subtasks = useTodoSubtasks(expandedTodoId, selectedUserId)

  const loadTodos = async () => {
    if (!selectedUserId.value) return
    isLoading.value = true
    error.value = null
    try {
      const domain = domainFilter?.value ?? undefined
      todos.value = await getTodos(selectedUserId.value, domain, { archived: true })
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load todos'
    } finally {
      isLoading.value = false
    }
  }

  const reorder = useTodoReorder(todos, loadTodos)

  const addTodo = async (data: { title: string; description?: string; priority?: string; start_date?: string | null; end_date?: string | null; domain?: string; color?: string; rrule?: string | null; tags?: string[] }) => {
    if (!selectedUserId.value) return
    const title = data.title.trim()
    if (!title) return
    try {
      await createTodo({
        user_id: selectedUserId.value,
        title,
        description: data.description || undefined,
        priority: (data.priority || 'medium') as Todo['priority'],
        start_date: data.start_date ?? null,
        end_date: data.end_date ?? null,
        domain: data.domain || undefined,
        color: data.color || undefined,
        rrule: data.rrule || undefined,
        tags: data.tags || [],
      })
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add todo'
    }
  }

  const toggleTodo = async (todo: Todo) => {
    try {
      const newStatus = todo.status === 'completed' ? 'pending' : 'completed'
      await updateTodo(todo.id, { status: newStatus })
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update todo'
    }
  }

  const togglePinned = async (todo: Todo) => {
    try {
      await updateTodo(todo.id, { pinned: !todo.pinned })
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update pin'
    }
  }

  const archiveTodo = async (todo: Todo) => {
    try {
      await updateTodo(todo.id, { status: 'archived' })
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to archive todo'
    }
  }

  const restoreTodo = async (todo: Todo) => {
    try {
      await updateTodo(todo.id, { status: 'pending' })
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to restore todo'
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

  if (domainFilter) {
    watch(domainFilter, () => { if (selectedUserId.value) loadTodos() })
  }

  return {
    todos,
    isLoading,
    error,
    activeFilter,
    searchQuery,
    filters,
    totalCount,
    activeCount,
    completedCount,
    archivedCount,
    overdueCount,
    displayedTodos,
    loadTodos,
    addTodo,
    toggleTodo,
    togglePinned,
    archiveTodo,
    restoreTodo,
    removeTodo,
    priority,
    dueDate,
    tags,
    sort,
    subtasks,
    reorder,
    expandedTodoId,
  }
}
