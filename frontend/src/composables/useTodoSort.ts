import type { Todo } from '../types'
import { computed, ref, type ComputedRef, type Ref } from 'vue'

export type TodoSortField = 'position' | 'priority' | 'due_date' | 'created_at' | 'title'

export const SORT_OPTIONS: { value: TodoSortField; label: string }[] = [
  { value: 'position', label: 'Position' },
  { value: 'priority', label: 'Priority' },
  { value: 'due_date', label: 'Due date' },
  { value: 'created_at', label: 'Created' },
  { value: 'title', label: 'Title' },
]

const PRIORITY_WEIGHT: Record<string, number> = { urgent: 0, high: 1, medium: 2, low: 3 }

export function useTodoSort(sourceTodos: Ref<Todo[]>) {
  const sortField: Ref<TodoSortField> = ref('position')
  const sortDir: Ref<'asc' | 'desc'> = ref('asc')

  const sorted: ComputedRef<Todo[]> = computed(() => {
    const list = [...sourceTodos.value]
    const field = sortField.value
    const dir = sortDir.value === 'asc' ? 1 : -1

    list.sort((a, b) => {
      if (a.pinned !== b.pinned) return a.pinned ? -1 : 1

      let cmp = 0
      switch (field) {
        case 'position':
          cmp = a.position - b.position
          break
        case 'priority':
          cmp = (PRIORITY_WEIGHT[a.priority] ?? 4) - (PRIORITY_WEIGHT[b.priority] ?? 4)
          break
        case 'due_date': {
          const ad = a.due_date ? new Date(a.due_date).getTime() : Infinity
          const bd = b.due_date ? new Date(b.due_date).getTime() : Infinity
          cmp = ad - bd
          break
        }
        case 'created_at': {
          const ac = new Date(a.created_at).getTime()
          const bc = new Date(b.created_at).getTime()
          cmp = ac - bc
          break
        }
        case 'title':
          cmp = a.title.localeCompare(b.title)
          break
      }
      return cmp * dir
    })
    return list
  })

  function toggleDir() {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  }

  function setSort(field: TodoSortField) {
    if (sortField.value === field) {
      toggleDir()
    } else {
      sortField.value = field
      sortDir.value = 'asc'
    }
  }

  return {
    sortField,
    sortDir,
    sorted,
    setSort,
    toggleDir,
  }
}
