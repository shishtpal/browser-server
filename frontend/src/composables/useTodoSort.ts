import type { Todo, TodoSortField } from '../types'
import { computed, ref, type ComputedRef, type Ref } from 'vue'

export const SORT_OPTIONS: { value: TodoSortField; label: string }[] = [
  { value: 'position', label: 'Position' },
  { value: 'priority', label: 'Priority' },
  { value: 'start_date', label: 'Date' },
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
        case 'start_date': {
          const ad = a.start_date ? new Date(a.start_date).getTime() : Infinity
          const bd = b.start_date ? new Date(b.start_date).getTime() : Infinity
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
