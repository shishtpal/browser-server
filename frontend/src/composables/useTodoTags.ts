import type { Todo } from '../types'
import { computed, ref, type ComputedRef, type Ref } from 'vue'

export function useTodoTags(todos: Ref<Todo[]>) {
  const selectedTag: Ref<string | null> = ref(null)

  const allTags: ComputedRef<string[]> = computed(() => {
    const tagSet = new Set<string>()
    todos.value.forEach(t => t.tags?.forEach(tag => tagSet.add(tag)))
    return Array.from(tagSet).sort()
  })

  const filteredByTag: ComputedRef<Todo[]> = computed(() => {
    if (!selectedTag.value) return todos.value
    return todos.value.filter(t => (t.tags || []).includes(selectedTag.value))
  })

  const hasTagFilter = computed(() => selectedTag.value !== null)

  function selectTag(tag: string | null) {
    selectedTag.value = tag
  }

  function clearTagFilter() {
    selectedTag.value = null
  }

  return {
    allTags,
    selectedTag,
    filteredByTag,
    hasTagFilter,
    selectTag,
    clearTagFilter,
  }
}
