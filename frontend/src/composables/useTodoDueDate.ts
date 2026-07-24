import type { Todo } from '../types'
import { computed, ref, type Ref } from 'vue'

export function isOverdue(todo: Todo): boolean {
  if (!todo.due_date || todo.completed) return false
  return new Date(todo.due_date) < new Date(new Date().toDateString())
}

export function isDueToday(todo: Todo): boolean {
  if (!todo.due_date || todo.completed) return false
  return new Date(todo.due_date).toDateString() === new Date().toDateString()
}

export function isDueThisWeek(todo: Todo): boolean {
  if (!todo.due_date || todo.completed) return false
  const due = new Date(todo.due_date)
  const now = new Date()
  const weekEnd = new Date(now)
  weekEnd.setDate(now.getDate() + (7 - now.getDay()) % 7)
  return due >= now && due <= weekEnd
}

export function useTodoDueDate() {
  const dueDateFilter: Ref<'overdue' | 'today' | 'this_week' | null> = ref(null)

  const filteredByDueDate = (todos: Ref<Todo[]>) =>
    computed(() => {
      if (!dueDateFilter.value) return todos.value
      return todos.value.filter(t => {
        switch (dueDateFilter.value) {
          case 'overdue': return isOverdue(t)
          case 'today': return isDueToday(t)
          case 'this_week': return isDueThisWeek(t)
        }
        return true
      })
    })

  const dueDateBadgeClass = (todo: Todo) => {
    if (todo.completed) return 'bg-gray-100 text-gray-500'
    if (isOverdue(todo)) return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400'
    if (isDueToday(todo)) return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
    if (isDueThisWeek(todo)) return 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-400'
    return 'bg-gray-100 text-gray-600'
  }

  const dueDateLabel = (todo: Todo) => {
    if (!todo.due_date) return null
    const d = new Date(todo.due_date)
    if (isOverdue(todo)) return 'Overdue'
    if (isDueToday(todo)) return 'Today'
    if (isDueThisWeek(todo)) return 'This week'
    return d.toLocaleDateString()
  }

  function clearDueDateFilter() {
    dueDateFilter.value = null
  }

  return {
    dueDateFilter,
    filteredByDueDate,
    dueDateBadgeClass,
    dueDateLabel,
    clearDueDateFilter,
  }
}
