import type {
  BrowserServerClient,
  GroupedHistoryEntry,
  HistorySearchColumn,
} from '@browser-server/shared-client'
import { computed, ref, watch, type Ref } from 'vue'
import { timeAgo } from '@browser-server/shared-utils'

export type { HistorySearchColumn }

export interface HistoryView {
  url: string
  title: string
  count: number
  totalDuration: number
  lastVisited: string
  lastVisitedLabel: string
}

const PAGE_SIZE = 100
const DEBOUNCE_MS = 200

function toView(entry: GroupedHistoryEntry): HistoryView {
  return {
    url: entry.url,
    title: entry.title || entry.url,
    count: entry.count,
    totalDuration: entry.total_duration,
    lastVisited: entry.last_visited,
    lastVisitedLabel: timeAgo(entry.last_visited),
  }
}

export function useHistoryView(client: Ref<BrowserServerClient | null>, userId: Ref<number>) {
  const paginatedEntries = ref<HistoryView[]>([])
  const totalCount = ref(0)
  const errorMessage = ref<string | null>(null)
  const isLoading = ref(false)
  const searchQuery = ref('')
  const searchColumn = ref<HistorySearchColumn>('all')
  const currentPage = ref(1)

  const debouncedQuery = ref('')
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  // Guards against an earlier slow request overwriting a newer one.
  let requestSeq = 0

  const totalPages = computed(() => Math.max(1, Math.ceil(totalCount.value / PAGE_SIZE)))

  async function load(): Promise<void> {
    if (!client.value || !userId.value) {
      return
    }

    const seq = ++requestSeq
    isLoading.value = true
    errorMessage.value = null

    try {
      const response = await client.value.getGroupedHistory({
        user_id: userId.value,
        q: debouncedQuery.value.trim() || undefined,
        column: searchColumn.value,
        limit: PAGE_SIZE,
        offset: (currentPage.value - 1) * PAGE_SIZE,
      })
      if (seq !== requestSeq) return // a newer request superseded this one
      paginatedEntries.value = response.entries.map(toView)
      totalCount.value = response.total
    } catch (error) {
      if (seq !== requestSeq) return
      const message = error instanceof Error ? error.message : 'Unknown error'
      errorMessage.value = `Server not reachable. ${message}`
      paginatedEntries.value = []
      totalCount.value = 0
    } finally {
      if (seq === requestSeq) isLoading.value = false
    }
  }

  // Reset to the first page and reload whenever the search changes.
  watch(searchQuery, (value) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      debouncedQuery.value = value
      currentPage.value = 1
      void load()
    }, DEBOUNCE_MS)
  })

  watch(searchColumn, () => {
    currentPage.value = 1
    void load()
  })

  function nextPage() {
    if (currentPage.value < totalPages.value) {
      currentPage.value++
      void load()
    }
  }

  function prevPage() {
    if (currentPage.value > 1) {
      currentPage.value--
      void load()
    }
  }

  return {
    paginatedEntries,
    totalCount,
    errorMessage,
    isLoading,
    searchQuery,
    searchColumn,
    currentPage,
    totalPages,
    load,
    nextPage,
    prevPage,
  }
}
