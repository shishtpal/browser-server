import type { History, BrowserServerClient } from '@browser-server/shared-client'
import { computed, ref, type Ref } from 'vue'
import { summarizeHistory } from './history'
import { timeAgo } from '@browser-server/shared-utils'

export type HistorySearchColumn = 'title' | 'url' | 'all'

export interface GroupedHistoryEntry {
  url: string
  title: string
  count: number
  lastVisited: string
  lastVisitedLabel: string
}

export function useHistoryView(client: Ref<BrowserServerClient | null>, userId: Ref<number>) {
  const entries = ref<History[]>([])
  const grouped = ref<GroupedHistoryEntry[]>([])
  const stats = ref<string>('Loading…')
  const errorMessage = ref<string | null>(null)
  const isLoading = ref(false)
  const searchQuery = ref('')
  const searchColumn = ref<HistorySearchColumn>('all')

  function refreshGrouping() {
    const mapped = summarizeHistory(entries.value).map<GroupedHistoryEntry>((entry) => ({
      ...entry,
      lastVisitedLabel: timeAgo(entry.lastVisited),
    }))
    grouped.value = mapped
  }

  function matchesColumn(entry: GroupedHistoryEntry, col: HistorySearchColumn, term: string): boolean {
    if (col === 'title') return entry.title.toLowerCase().includes(term)
    if (col === 'url') return entry.url.toLowerCase().includes(term)
    return `${entry.title} ${entry.url}`.toLowerCase().includes(term)
  }

  const filtered = computed(() => {
    const q = searchQuery.value.toLowerCase().trim()
    if (!q) return grouped.value
    const col = searchColumn.value
    const terms = q.split(/\s+/).filter(Boolean)
    return grouped.value.filter((entry) => terms.every((t) => matchesColumn(entry, col, t)))
  })

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

  return { entries, grouped, filtered, stats, errorMessage, isLoading, searchQuery, searchColumn, load, clearAll }
}
