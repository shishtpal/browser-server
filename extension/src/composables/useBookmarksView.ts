import type { BookmarkResponse, BrowserServerClient, CreateBookmarkInput } from '@browser-server/shared-client'
import { computed, ref, type Ref } from 'vue'

export type BookmarkSearchColumn = 'title' | 'url' | 'all'

export function useBookmarksView(client: Ref<BrowserServerClient | null>, userId: Ref<number>) {
  const items = ref<BookmarkResponse[]>([])
  const errorMessage = ref<string | null>(null)
  const isLoading = ref(false)
  const searchQuery = ref('')
  const searchColumn = ref<BookmarkSearchColumn>('all')

  function matchesColumn(entry: BookmarkResponse, col: BookmarkSearchColumn, term: string): boolean {
    if (col === 'title') return entry.title.toLowerCase().includes(term)
    if (col === 'url') return entry.url.toLowerCase().includes(term)
    return `${entry.title} ${entry.url}`.toLowerCase().includes(term)
  }

  const filtered = computed(() => {
    const q = searchQuery.value.toLowerCase().trim()
    if (!q) return items.value
    const col = searchColumn.value
    const terms = q.split(/\s+/).filter(Boolean)
    return items.value.filter((entry) => terms.every((t) => matchesColumn(entry, col, t)))
  })

  async function load(): Promise<void> {
    if (!client.value || !userId.value) return

    isLoading.value = true
    errorMessage.value = null

    try {
      items.value = await client.value.getBookmarks(userId.value)
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

  return { items, filtered, errorMessage, isLoading, searchQuery, searchColumn, load, addBookmark, deleteBookmark }
}
