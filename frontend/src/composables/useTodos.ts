import { ref, computed, watch, type Ref } from 'vue'
import { getTodos, createTodo, updateTodo, deleteTodo, getSubtasks, createSubtask } from '../lib/api'
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

  const newTitle = ref('')
  const newDescription = ref('')
  const newPriority = ref<'low' | 'medium' | 'high' | 'urgent' | ''>('')
  const newDueDate = ref<string | null>(null)
  const newTags = ref<string[]>([])
  const newMoreOpen = ref(false)

  const activeFilter = ref<'all' | 'active' | 'completed'>('all')
  const searchQuery = ref('')
  const filters = [
    { label: 'All', value: 'all' as const },
    { label: 'Active', value: 'active' as const },
    { label: 'Completed', value: 'completed' as const },
  ]

  const editingId = ref<number | null>(null)
  const editTitle = ref('')
  const editDescription = ref('')
  const editPriority = ref<'low' | 'medium' | 'high' | 'urgent' | ''>('')
  const editDueDate = ref<string | null>(null)
  const editTags = ref<string[]>([])

  const activeCount = computed(() => todos.value.filter(t => !t.completed).length)
  const completedCount = computed(() => todos.value.filter(t => t.completed).length)
  const overdueCount = computed(() => todos.value.filter(t => !t.completed && t.due_date && isOverdue(t)).length)

  const priority = useTodoPriority()
  const dueDate = useTodoDueDate()
  const tags = useTodoTags(todos)

  const baseFiltered = computed(() => {
    let list = todos.value
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
    if (activeFilter.value === 'active') list = list.filter(t => !t.completed)
    if (activeFilter.value === 'completed') list = list.filter(t => t.completed)
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
      const results = await getTodos(selectedUserId.value, domainFilter?.value ?? undefined)
      todos.value = results
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load todos'
    } finally {
      isLoading.value = false
    }
  }

  const reorder = useTodoReorder(todos, loadTodos)

  const addTodo = async (data?: { title: string; description?: string; priority?: string; due_date?: string | null; tags?: string[] }) => {
    if (!selectedUserId.value) return
    const title = data?.title || newTitle.value.trim()
    if (!title) return
    try {
      await createTodo({
        user_id: selectedUserId.value,
        title,
        description: data?.description || newDescription.value.trim() || undefined,
        priority: (data?.priority || newPriority.value || 'medium') as Todo['priority'],
        due_date: data?.due_date ?? newDueDate.value ?? null,
        tags: data?.tags || newTags.value,
      })
      if (!data) {
        newTitle.value = ''
        newDescription.value = ''
        newPriority.value = ''
        newDueDate.value = null
        newTags.value = []
        newMoreOpen.value = false
      }
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add todo'
    }
  }

  const toggleTodo = async (todo: Todo) => {
    try {
      await updateTodo(todo.id, { completed: !todo.completed })
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update todo'
    }
  }

  const startEdit = (todo: Todo) => {
    editingId.value = todo.id
    editTitle.value = todo.title
    editDescription.value = todo.description
    editPriority.value = (todo.priority as any) || 'medium'
    editDueDate.value = todo.due_date ?? null
    editTags.value = [...(todo.tags || [])]
  }

  const cancelEdit = () => {
    editingId.value = null
  }

  const saveEdit = async (todo: Todo, title: string, description: string, priority: string, dueDate: string | null, tags: string[]) => {
    try {
      await updateTodo(todo.id, {
        title,
        description,
        priority: (priority || 'medium') as Todo['priority'],
        due_date: dueDate ?? null,
        tags,
      })
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

  watch(activeFilter, () => { if (selectedUserId.value) loadTodos() })
  if (domainFilter) {
    watch(domainFilter, () => { if (selectedUserId.value) loadTodos() })
  }

  return {
    todos,
    isLoading,
    error,
    newTitle,
    newDescription,
    newPriority,
    newDueDate,
    newTags,
    newMoreOpen,
    activeFilter,
    searchQuery,
    filters,
    editingId,
    editTitle,
    editDescription,
    editPriority,
    editDueDate,
    editTags,
    activeCount,
    completedCount,
    overdueCount,
    displayedTodos,
    loadTodos,
    addTodo,
    toggleTodo,
    startEdit,
    cancelEdit,
    saveEdit,
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
