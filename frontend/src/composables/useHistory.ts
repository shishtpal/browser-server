import { ref, computed, type Ref } from 'vue'
import { getHistory, createHistory, deleteHistory } from '../lib/api'
import { formatDuration } from '../lib/utils'
import type { History } from '../types'

export function useHistory(selectedUserId: Ref<number | null>) {
  const historyEntries = ref<History[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const urlFilter = ref('')

  const newUrl = ref('')
  const newTitle = ref('')
  const newDuration = ref('')

  const totalDuration = computed(() =>
    formatDuration(historyEntries.value.reduce((sum, h) => sum + h.duration, 0))
  )

  const filteredHistory = computed(() => {
    if (!urlFilter.value.trim()) return historyEntries.value
    const q = urlFilter.value.toLowerCase()
    return historyEntries.value.filter(
      h => h.url.toLowerCase().includes(q) || h.title.toLowerCase().includes(q)
    )
  })

  const loadHistory = async () => {
    if (!selectedUserId.value) return
    isLoading.value = true
    error.value = null
    try {
      historyEntries.value = await getHistory(selectedUserId.value)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load history'
    } finally {
      isLoading.value = false
    }
  }

  const addEntry = async () => {
    if (!selectedUserId.value || !newUrl.value.trim() || !newTitle.value.trim()) return
    try {
      await createHistory({
        user_id: selectedUserId.value,
        url: newUrl.value.trim(),
        title: newTitle.value.trim(),
        duration: newDuration.value ? Number(newDuration.value) : 0,
      })
      newUrl.value = ''
      newTitle.value = ''
      newDuration.value = ''
      await loadHistory()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add history entry'
    }
  }

  const removeEntry = async (id: number) => {
    if (!confirm('Delete this history entry?')) return
    try {
      await deleteHistory(id)
      historyEntries.value = historyEntries.value.filter(h => h.id !== id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete entry'
    }
  }

  return {
    historyEntries,
    isLoading,
    error,
    urlFilter,
    newUrl,
    newTitle,
    newDuration,
    totalDuration,
    filteredHistory,
    loadHistory,
    addEntry,
    removeEntry,
  }
}
