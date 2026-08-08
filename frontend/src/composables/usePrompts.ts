import { ref, computed, watch, type Ref } from 'vue'
import { getPrompts, createPrompt, updatePrompt, deletePrompt, searchPrompts } from '../lib/api'
import type { PromptResponse, CreatePromptInput, UpdatePromptInput } from '../types'

export const UNTAGGED_FILTER = Symbol('untagged-filter')
export type TagFilter = string | typeof UNTAGGED_FILTER | null

export interface TagItem {
  name: string
  count: number
}

export function usePrompts(userId: Ref<number | null>) {
  const prompts = ref<PromptResponse[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // Search state
  const searchQuery = ref('')
  const searchResults = ref<PromptResponse[]>([])
  const isSearching = ref(false)

  // UI state
  const activeTag = ref<TagFilter>(null)
  const showPromptManager = ref(false)

  const currentUserId = computed(() => userId.value)

  const allTags = computed<TagItem[]>(() => {
    const counts = new Map<string, number>()
    for (const p of prompts.value) {
      for (const tag of p.tags || []) {
        counts.set(tag, (counts.get(tag) ?? 0) + 1)
      }
    }
    const items: TagItem[] = []
    for (const [name, count] of counts.entries()) {
      items.push({ name, count })
    }
    return items.sort((a, b) => b.count - a.count || a.name.localeCompare(b.name))
  })

  const untaggedCount = computed(() => prompts.value.filter((p) => !p.tags?.length).length)

  const loadPrompts = async (query?: string | null) => {
    if (!currentUserId.value) return
    isLoading.value = true
    error.value = null
    try {
      prompts.value = await getPrompts(currentUserId.value, query ?? undefined)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load prompts'
    } finally {
      isLoading.value = false
    }
  }

  const doSearch = async (query: string) => {
    if (!currentUserId.value) return
    isSearching.value = true
    try {
      searchResults.value = await searchPrompts(currentUserId.value, query, 20)
    } catch {
      searchResults.value = []
    } finally {
      isSearching.value = false
    }
  }

  const filterByTag = (tag: TagFilter) => {
    activeTag.value = tag
  }

  const addPrompt = async (data: CreatePromptInput) => {
    if (!currentUserId.value) return
    try {
      const resp = await createPrompt({ ...data, user_id: currentUserId.value })
      prompts.value.unshift(resp)
      return resp
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create prompt'
    }
  }

  const editPrompt = async (id: number, data: UpdatePromptInput) => {
    try {
      const resp = await updatePrompt(id, data)
      const idx = prompts.value.findIndex((p) => p.id === id)
      if (idx >= 0) prompts.value[idx] = resp
      return resp
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update prompt'
    }
  }

  const removePrompt = async (id: number) => {
    try {
      await deletePrompt(id)
      prompts.value = prompts.value.filter((p) => p.id !== id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete prompt'
    }
  }

  watch(
    () => currentUserId.value,
    (val) => {
      if (val && val > 0) {
        activeTag.value = null
        loadPrompts()
      }
    },
  )

  return {
    prompts,
    isLoading,
    error,
    searchQuery,
    searchResults,
    isSearching,
    activeTag,
    showPromptManager,
    allTags,
    untaggedCount,
    loadPrompts,
    doSearch,
    filterByTag,
    addPrompt,
    editPrompt,
    removePrompt,
  }
}
