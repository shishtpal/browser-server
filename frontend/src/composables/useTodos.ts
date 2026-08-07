import type { Todo, TodoStatus, TodoFilter } from '../types'
import { ref, computed, watch, type Ref } from 'vue'
import { getTodos, createTodo, updateTodo, deleteTodo } from '../lib/api'
import { useTodoPriority } from './useTodoPriority'
import { useTodoDueDate } from './useTodoDueDate'
import { useTodoTags } from './useTodoTags'
import { useTodoSort } from './useTodoSort'

import { isOverdue, isDueToday, isDueThisWeek } from './useTodoDueDate'
import { useLocalStorage, useSessionStorage } from '@vueuse/core'

export function useTodos(selectedUserId: Ref<number | null>, domainFilter?: Ref<string | null>) {
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  
  const todos = useSessionStorage<Todo[]>(`bs.todos.todos`, [])

  const activeFilter = useLocalStorage<TodoFilter>(`bs.todos.activeFilter`, 'active')
  
  const searchQuery = ref('')
  
  const filters = [
    { label: 'All', value: 'all' as const },
    { label: 'Active', value: 'active' as const },
    { label: 'In Progress', value: 'in_progress' as const },
    { label: 'Completed', value: 'completed' as const },
    { label: 'Archived', value: 'archived' as const },
  ]

  const counts = computed(() => {
    const result = { total: 0, active: 0, inProgress: 0, completed: 0, archived: 0, overdue: 0 }
    for (const todo of todos.value) {
      if (todo.status === 'archived') result.archived++
      else result.total++
      if (todo.status === 'pending') result.active++
      else if (todo.status === 'in_progress') result.inProgress++
      else if (todo.status === 'completed') result.completed++
      if ((todo.status === 'pending' || todo.status === 'in_progress') && todo.start_date && isOverdue(todo)) result.overdue++
    }
    return result
  })
  const totalCount = computed(() => counts.value.total)
  const activeCount = computed(() => counts.value.active)
  const inProgressCount = computed(() => counts.value.inProgress)
  const completedCount = computed(() => counts.value.completed)
  const archivedCount = computed(() => counts.value.archived)
  const overdueCount = computed(() => counts.value.overdue)

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
    if (activeFilter.value === 'active') list = list.filter(t => t.status === 'pending' || t.status === 'in_progress')
    if (activeFilter.value === 'in_progress') list = list.filter(t => t.status === 'in_progress')
    if (activeFilter.value === 'completed') list = list.filter(t => t.status === 'completed')
    return list
  })

  const sort = useTodoSort(baseFiltered)
  const displayedTodos = sort.sorted

  // Set of TODOs IDs user has clicked to expand for sub-tasks
  const expandedTodoIds = useLocalStorage<Set<number>>(`bs.todos.expandedTodoIds`, new Set(), {
    serializer: {
      read: (v) => v ? new Set(JSON.parse(v)) : new Set(),
      write: (v) => JSON.stringify([...v]),
    },
  })

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

  function replaceTodo(updated: Todo) {
    const index = todos.value.findIndex(todo => todo.id === updated.id)
    if (index === -1) return
    const current = todos.value[index]
    todos.value[index] = { ...current, ...updated, subtasks: current.subtasks || [] }
  }

  async function updateTodoItem(id: number, data: Partial<Todo>) {
    const updated = await updateTodo(id, data)
    replaceTodo(updated)
    return updated
  }

  const addTodo = async (data: { title: string; description?: string; priority?: string; start_date?: string | null; end_date?: string | null; domain?: string; color?: string; rrule?: string | null; tags?: string[] }) => {
    if (!selectedUserId.value) return
    const title = data.title.trim()
    if (!title) return
    try {
      const created = await createTodo({
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
      todos.value.push({ ...created, subtasks: created.subtasks || [] })
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add todo'
    }
  }

  const toggleTodo = async (todo: Todo) => {
    try {
      const cycle: Record<string, TodoStatus> = {
        pending: 'in_progress',
        in_progress: 'completed',
        completed: 'pending',
      }
      const newStatus: TodoStatus = cycle[todo.status] || 'pending'
      await updateTodoItem(todo.id, { status: newStatus })
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update todo'
    }
  }

  const togglePinned = async (todo: Todo) => {
    try {
      await updateTodoItem(todo.id, { pinned: !todo.pinned })
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update pin'
    }
  }

  const archiveTodo = async (todo: Todo) => {
    try {
      await updateTodoItem(todo.id, { status: 'archived' })
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to archive todo'
    }
  }

  const restoreTodo = async (todo: Todo) => {
    try {
      await updateTodoItem(todo.id, { status: 'pending' })
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to restore todo'
    }
  }

  const removeTodo = async (id: number) => {
    try {
      await deleteTodo(id)
      todos.value = todos.value.filter(todo => todo.id !== id)
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
    inProgressCount,
    completedCount,
    archivedCount,
    overdueCount,
    displayedTodos,
    loadTodos,
    addTodo,
    updateTodoItem,
    toggleTodo,
    togglePinned,
    archiveTodo,
    restoreTodo,
    removeTodo,
    priority,
    dueDate,
    tags,
    sort,
    expandedTodoIds,
  }
}
