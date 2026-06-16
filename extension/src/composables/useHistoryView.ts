import type { History, BrowserServerClient } from '@browser-server/shared-client'
import { computed, ref, watch, type Ref } from 'vue'
import { summarizeHistory } from './history'
import { timeAgo } from '@browser-server/shared-utils'

export type HistorySearchColumn = 'title' | 'url' | 'all'

export interface GroupedHistoryEntry {
  url: string
  title: string
  count: number
  lastVisited: string
  lastVisitedLabel: string
  _lowerTitle: string
  _lowerUrl: string
  _lowerCombined: string
}

const PAGE_SIZE = 100
const DEBOUNCE_MS = 150

export function useHistoryView(client: Ref<BrowserServerClient | null>, userId: Ref<number>) {
  const entries = ref<History[]>([])
  const grouped = ref<GroupedHistoryEntry[]>([])
  const stats = ref<string>('Loading…')
  const errorMessage = ref<string | null>(null)
  const isLoading = ref(false)
  const searchQuery = ref('')
  const searchColumn = ref<HistorySearchColumn>('all')
  const currentPage = ref(1)

  const debouncedQuery = ref('')
  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  watch(searchQuery, (value) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      debouncedQuery.value = value
      currentPage.value = 1
    }, DEBOUNCE_MS)
  })

  function refreshGrouping() {
    grouped.value = summarizeHistory(entries.value).map<GroupedHistoryEntry>((entry) => ({
      ...entry,
      lastVisitedLabel: timeAgo(entry.lastVisited),
      _lowerTitle: entry.title.toLowerCase(),
      _lowerUrl: entry.url.toLowerCase(),
      _lowerCombined: `${entry.title} ${entry.url}`.toLowerCase(),
    }))
  }

  function matchesColumn(entry: GroupedHistoryEntry, col: HistorySearchColumn, term: string): boolean {
    if (col === 'title') return entry._lowerTitle.includes(term)
    if (col === 'url') return entry._lowerUrl.includes(term)
    return entry._lowerCombined.includes(term)
  }

  const filtered = computed(() => {
    const q = debouncedQuery.value.toLowerCase().trim()
    if (!q) return grouped.value
    const col = searchColumn.value
    const terms = q.split(/\s+/).filter(Boolean)
    return grouped.value.filter((entry) => terms.every((t) => matchesColumn(entry, col, t)))
  })

  const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / PAGE_SIZE)))

  const paginatedEntries = computed(() => {
    const start = (currentPage.value - 1) * PAGE_SIZE
    return filtered.value.slice(start, start + PAGE_SIZE)
  })

  watch(searchColumn, () => {
    currentPage.value = 1
  })

  function nextPage() {
    if (currentPage.value < totalPages.value) currentPage.value++
  }

  function prevPage() {
    if (currentPage.value > 1) currentPage.value--
  }

  function goToPage(page: number) {
    if (page >= 1 && page <= totalPages.value) currentPage.value = page
  }

  async function load(): Promise<void> {
    if (!client.value || !userId.value) {
      return
    }

    isLoading.value = true
    errorMessage.value = null

    try {
      entries.value = await client.value.getHistory(userId.value)
      refreshGrouping()
      const totalVisits = grouped.value.reduce((sum, entry) => sum + entry.count, 0)
      stats.value = `${grouped.value.length} pages · ${totalVisits} visits`
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      errorMessage.value = `Server not reachable. ${message}`
      stats.value = '0 pages · 0 visits'
      grouped.value = []
      entries.value = []
    } finally {
      isLoading.value = false
    }
  }

  async function clearAll(): Promise<void> {
    if (!client.value || !userId.value) {
      return
    }

    try {
      await Promise.all(entries.value.map((entry) => client.value!.deleteHistory(entry.id)))
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      console.debug('Clear history failed', message)
    }

    await load()
  }

  return {
    entries, grouped, filtered, paginatedEntries, stats, errorMessage, isLoading,
    searchQuery, searchColumn, currentPage, totalPages,
    load, clearAll, nextPage, prevPage, goToPage,
  }
}
