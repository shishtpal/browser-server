import type { History, BrowserServerClient } from '@browser-server/shared-client'
import { ref, type Ref } from 'vue'
import { summarizeHistory } from './history'
import { timeAgo } from '../lib/format'

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

  function refreshGrouping() {
    const mapped = summarizeHistory(entries.value).map<GroupedHistoryEntry>((entry) => ({
      ...entry,
      lastVisitedLabel: timeAgo(entry.lastVisited),
    }))
    grouped.value = mapped
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

  return { entries, grouped, stats, errorMessage, isLoading, load, clearAll }
}
