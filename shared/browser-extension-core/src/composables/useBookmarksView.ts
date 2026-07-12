import type { BookmarkResponse, BrowserServerClient, CreateBookmarkInput } from '@browser-server/shared-client'
import { computed, ref, watch, type Ref } from 'vue'

export type BookmarkSearchColumn = 'title' | 'url' | 'all'

interface BookmarkedEntry extends BookmarkResponse {
  _lowerTitle: string
  _lowerUrl: string
  _lowerCombined: string
}

const PAGE_SIZE = 100
const DEBOUNCE_MS = 150

export function useBookmarksView(client: Ref<BrowserServerClient | null>, userId: Ref<number>) {
  const items = ref<BookmarkedEntry[]>([])
  const errorMessage = ref<string | null>(null)
  const isLoading = ref(false)
  const searchQuery = ref('')
  const searchColumn = ref<BookmarkSearchColumn>('all')
  const activeTag = ref('')
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

  watch([searchColumn, activeTag], () => {
    currentPage.value = 1
  })

  function toEntry(bookmark: BookmarkResponse): BookmarkedEntry {
    return {
      ...bookmark,
      _lowerTitle: bookmark.title.toLowerCase(),
      _lowerUrl: bookmark.url.toLowerCase(),
      _lowerCombined: `${bookmark.title} ${bookmark.url}`.toLowerCase(),
    }
  }

  function matchesColumn(entry: BookmarkedEntry, col: BookmarkSearchColumn, term: string): boolean {
    if (col === 'title') return entry._lowerTitle.includes(term)
    if (col === 'url') return entry._lowerUrl.includes(term)
    return entry._lowerCombined.includes(term)
  }

  const allTags = computed(() => {
    const tags = new Set<string>()
    for (const item of items.value) {
      for (const tag of item.tags) {
        tags.add(tag)
      }
    }
    return Array.from(tags).sort()
  })

  const filtered = computed(() => {
    let result = items.value

    if (activeTag.value) {
      result = result.filter((entry) => entry.tags.includes(activeTag.value))
    }

    const q = debouncedQuery.value.toLowerCase().trim()
    if (q) {
      const col = searchColumn.value
      const terms = q.split(/\s+/).filter(Boolean)
      result = result.filter((entry) => terms.every((t) => matchesColumn(entry, col, t)))
    }

    return result
  })

  const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / PAGE_SIZE)))

  const paginatedEntries = computed(() => {
    const start = (currentPage.value - 1) * PAGE_SIZE
    return filtered.value.slice(start, start + PAGE_SIZE)
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
    if (!client.value || !userId.value) return

    isLoading.value = true
    errorMessage.value = null

    try {
      const raw = await client.value.getBookmarks(userId.value)
      items.value = raw.map(toEntry)
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      errorMessage.value = `Server not reachable. ${message}`
      items.value = []
    } finally {
      isLoading.value = false
    }
  }

  async function addBookmark(data: CreateBookmarkInput): Promise<void> {
    if (!client.value) return
    try {
      await client.value.createBookmark(data)
      await load()
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      console.debug('Create bookmark failed', message)
    }
  }

  async function deleteBookmark(id: number): Promise<void> {
    if (!client.value) return
    try {
      await client.value.deleteBookmark(id)
      await load()
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      console.debug('Delete bookmark failed', message)
    }
  }

  return {
    items, filtered, paginatedEntries, errorMessage, isLoading,
    searchQuery, searchColumn, activeTag, allTags,
    currentPage, totalPages,
    load, addBookmark, deleteBookmark, nextPage, prevPage, goToPage,
  }
}
