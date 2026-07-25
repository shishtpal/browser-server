import { ref, computed, watch, type Ref, type ComputedRef } from 'vue'
import { format, startOfDay } from 'date-fns'
import { getTodos, createTodo, updateTodo, deleteTodo } from '../lib/api'
import type { Todo, CreateTodoInput } from '../types'
import type { CalendarDay, CalendarStats, DateRange } from '../components/calendar/types'

export function useCalendarTodos(
  selectedUserId: Ref<number | null>,
  dateRange: ComputedRef<DateRange>,
) {
  const todos = ref<Todo[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const rangeStart = computed(() => startOfDay(dateRange.value.start))
  const rangeEnd = computed(() => startOfDay(dateRange.value.end))

  const visibleTodos = computed(() => {
    return todos.value.filter((t) => {
      if (!t.start_date) return false
      const start = startOfDay(new Date(t.start_date + 'T00:00:00'))
      return start >= rangeStart.value && start <= rangeEnd.value
    })
  })

  const days = computed<CalendarDay[]>(() => {
    const result: CalendarDay[] = []
    const current = new Date(rangeStart.value)
    const today = new Date()
    // For month view, use the month from the middle of the range to determine "current month"
    const midDate = new Date(rangeStart.value)
    midDate.setDate(midDate.getDate() + Math.floor((rangeEnd.value.getTime() - rangeStart.value.getTime()) / (1000 * 60 * 60 * 24) / 2))
    const viewMonth = midDate.getMonth()
    const viewYear = midDate.getFullYear()

    while (current <= rangeEnd.value) {
      const dateStr = format(current, 'yyyy-MM-dd')
      result.push({
        date: dateStr,
        isToday: isSameDay(current, today),
        isCurrentMonth: current.getMonth() === viewMonth && current.getFullYear() === viewYear,
        isWeekend: current.getDay() === 0 || current.getDay() === 6,
        todos: visibleTodos.value.filter((t) => t.start_date === dateStr),
      })
      current.setDate(current.getDate() + 1)
    }
    return result
  })

  const stats = computed<CalendarStats>(() => {
    const todayStr = format(new Date(), 'yyyy-MM-dd')
    const todayCount = todos.value.filter((t) => t.start_date === todayStr && t.status === 'pending').length
    const overdueCount = todos.value.filter((t) => {
      if (!t.start_date || t.status !== 'pending') return false
      return startOfDay(new Date(t.start_date + 'T00:00:00')) < startOfDay(new Date())
    }).length
    const completedCount = todos.value.filter((t) => t.status === 'completed').length
    return { todayCount, overdueCount, completedCount }
  })

  const loadTodos = async () => {
    if (!selectedUserId.value) return
    isLoading.value = true
    error.value = null
    try {
      const data = await getTodos(selectedUserId.value)
      todos.value = data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load todos'
    } finally {
      isLoading.value = false
    }
  }

  const addTodo = async (data: CreateTodoInput) => {
    try {
      await createTodo(data)
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create todo'
      throw e
    }
  }

  const updateTodoItem = async (id: number, data: Partial<Todo>) => {
    try {
      await updateTodo(id, data)
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update todo'
      throw e
    }
  }

  const removeTodo = async (id: number) => {
    try {
      await deleteTodo(id)
      await loadTodos()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete todo'
      throw e
    }
  }

  return {
    todos,
    isLoading,
    error,
    visibleTodos,
    days,
    stats,
    loadTodos,
    addTodo,
    updateTodoItem,
    removeTodo,
  }
}

function isSameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
}
