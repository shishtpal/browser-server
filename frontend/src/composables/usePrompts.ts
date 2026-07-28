import { ref, computed, watch, type Ref } from 'vue'
import { getPromptFolders, createPromptFolder, updatePromptFolder, deletePromptFolder, getPrompts, createPrompt, updatePrompt, deletePrompt, searchPrompts } from '../lib/api'
import type { PromptFolder, PromptResponse, CreatePromptFolderInput, CreatePromptInput, UpdatePromptFolderInput, UpdatePromptInput } from '../types'

export function usePrompts(userId: Ref<number | null>) {
  const folders = ref<PromptFolder[]>([])
  const prompts = ref<PromptResponse[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // Search state
  const searchQuery = ref('')
  const searchResults = ref<PromptResponse[]>([])
  const isSearching = ref(false)

  // UI state
  const selectedFolderId = ref<number | null>(null)
  const showPromptManager = ref(false)

  const currentUserId = computed(() => userId.value)

  const ungroupedPrompts = computed(() => prompts.value.filter(p => p.folder_id == null))
  const folderMap = computed(() => {
    const map = new Map<number, PromptFolder>()
    for (const f of folders.value) map.set(f.id, f)
    return map
  })

  const loadFolders = async () => {
    if (!currentUserId.value) return
    isLoading.value = true
    error.value = null
    try {
      folders.value = await getPromptFolders(currentUserId.value)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load folders'
    } finally {
      isLoading.value = false
    }
  }

  const loadPrompts = async (folderId?: number | null, query?: string) => {
    if (!currentUserId.value) return
    isLoading.value = true
    error.value = null
    try {
      prompts.value = await getPrompts(currentUserId.value, folderId, query)
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

  const addFolder = async (data: CreatePromptFolderInput) => {
    if (!currentUserId.value) return
    try {
      const folder = await createPromptFolder({ ...data, user_id: currentUserId.value })
      folders.value.push(folder)
      return folder
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create folder'
    }
  }

  const renameFolder = async (id: number, data: UpdatePromptFolderInput) => {
    try {
      const folder = await updatePromptFolder(id, data)
      const idx = folders.value.findIndex(f => f.id === id)
      if (idx >= 0) folders.value[idx] = folder
      return folder
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to rename folder'
    }
  }

  const removeFolder = async (id: number) => {
    try {
      await deletePromptFolder(id)
      folders.value = folders.value.filter(f => f.id !== id)
      prompts.value = prompts.value.map(p => p.folder_id === id ? { ...p, folder_id: null, folder_name: null } : p)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete folder'
    }
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
      const idx = prompts.value.findIndex(p => p.id === id)
      if (idx >= 0) prompts.value[idx] = resp
      return resp
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update prompt'
    }
  }

  const removePrompt = async (id: number) => {
    try {
      await deletePrompt(id)
      prompts.value = prompts.value.filter(p => p.id !== id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete prompt'
    }
  }

  watch(() => currentUserId.value, (val) => {
    if (val && val > 0) {
      loadFolders()
      loadPrompts(null)
    }
  })

  return {
    folders,
    prompts,
    isLoading,
    error,
    searchQuery,
    searchResults,
    isSearching,
    selectedFolderId,
    showPromptManager,
    ungroupedPrompts,
    folderMap,
    loadFolders,
    loadPrompts,
    doSearch,
    addFolder,
    renameFolder,
    removeFolder,
    addPrompt,
    editPrompt,
    removePrompt,
  }
}
