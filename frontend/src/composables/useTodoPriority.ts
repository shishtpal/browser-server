import type { Todo, TodoPriority } from '../types'
import { computed, ref, type Ref } from 'vue'

export const PRIORITIES: { value: TodoPriority; label: string; color: string; accent: string }[] = [
  { value: 'low', label: 'Low', color: 'bg-emerald-50 text-emerald-700', accent: 'bg-emerald-500' },
  { value: 'medium', label: 'Medium', color: 'bg-amber-50 text-amber-700', accent: 'bg-amber-500' },
  { value: 'high', label: 'High', color: 'bg-orange-50 text-orange-700', accent: 'bg-orange-500' },
  { value: 'urgent', label: 'Urgent', color: 'bg-red-50 text-red-700', accent: 'bg-red-500' },
]

export function prioritySortWeight(p: TodoPriority): number {
  switch (p) {
    case 'urgent': return 0
    case 'high': return 1
    case 'medium': return 2
    case 'low': return 3
  }
  return 2
}

export function useTodoPriority() {
  const selectedPriority: Ref<TodoPriority | null> = ref(null)

  const priorityClass = (p: TodoPriority | string) => {
    const found = PRIORITIES.find(pr => pr.value === p)
    return found ? found.color : 'bg-gray-50 text-gray-600'
  }

  const accentClass = (p: TodoPriority | string) => {
    const found = PRIORITIES.find(pr => pr.value === p)
    return found ? found.accent : 'bg-gray-400'
  }

  const filteredByPriority = (todos: Ref<Todo[]>) =>
    computed(() => {
      if (!selectedPriority.value) return todos.value
      return todos.value.filter(t => t.priority === selectedPriority.value)
    })

  function clearPriority() {
    selectedPriority.value = null
  }

  return {
    selectedPriority,
    filteredByPriority,
    priorityClass,
    accentClass,
    clearPriority,
  }
}
