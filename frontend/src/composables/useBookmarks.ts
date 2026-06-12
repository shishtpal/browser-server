import { ref, computed, watch, type Ref } from 'vue'
import { getBookmarks, createBookmark, updateBookmark, deleteBookmark } from '../lib/api'
import type { BookmarkResponse } from '../types'

export function useBookmarks(selectedUserId: Ref<number | null>) {
  const bookmarks = ref<BookmarkResponse[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const activeTagFilter = ref<string | null>(null)
  const searchQuery = ref('')

  const newTitle = ref('')
  const newUrl = ref('')
  const newDescription = ref('')
  const newTags = ref('')

  const editing = ref<BookmarkResponse | null>(null)
  const editForm = ref({ title: '', url: '', description: '', tagsStr: '' })

  const allTags = computed(() =>
    Array.from(new Set(bookmarks.value.flatMap(b => b.tags))).sort()
  )

  const filteredBookmarks = computed(() => {
    const q = searchQuery.value.toLowerCase().trim()
    if (!q) return bookmarks.value
    const terms = q.split(/\s+/).filter(Boolean)
    return bookmarks.value.filter(b => {
      const haystack = [b.title, b.url, b.description, b.folder_path, ...b.tags]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
      return terms.every(t => haystack.includes(t))
    })
  })

  const loadBookmarks = async () => {
    if (!selectedUserId.value) return
    isLoading.value = true
    error.value = null
    try {
      bookmarks.value = await getBookmarks(selectedUserId.value, activeTagFilter.value || undefined)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load bookmarks'
    } finally {
      isLoading.value = false
    }
  }

  const addBookmark = async () => {
    if (!selectedUserId.value || !newTitle.value.trim() || !newUrl.value.trim()) return
    const tagList = newTags.value
      ? newTags.value.split(',').map(t => t.trim()).filter(Boolean)
      : []
    try {
      await createBookmark({
        user_id: selectedUserId.value,
        title: newTitle.value.trim(),
        url: newUrl.value.trim(),
        description: newDescription.value.trim() || undefined,
        tags: tagList.length ? tagList : undefined,
      })
      newTitle.value = ''
      newUrl.value = ''
      newDescription.value = ''
      newTags.value = ''
      await loadBookmarks()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add bookmark'
    }
  }

  const filterByTag = (tag: string) => {
    activeTagFilter.value = tag
  }

  const openEdit = (b: BookmarkResponse) => {
    editing.value = b
    editForm.value = {
      title: b.title,
      url: b.url,
      description: b.description,
      tagsStr: b.tags.join(', '),
    }
  }

  const saveEdit = async () => {
    if (!editing.value) return
    const tagList = editForm.value.tagsStr
      ? editForm.value.tagsStr.split(',').map(t => t.trim()).filter(Boolean)
      : []
    try {
      await updateBookmark(editing.value.id, {
        user_id: editing.value.user_id,
        title: editForm.value.title,
        url: editForm.value.url,
        description: editForm.value.description,
        tags: tagList,
      })
      editing.value = null
      await loadBookmarks()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update bookmark'
    }
  }

  const removeBookmark = async (id: number) => {
    if (!confirm('Delete this bookmark?')) return
    try {
      await deleteBookmark(id)
      await loadBookmarks()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete bookmark'
    }
  }

  watch(activeTagFilter, () => {
    if (selectedUserId.value) loadBookmarks()
  })

  return {
    bookmarks,
    isLoading,
    error,
    activeTagFilter,
    searchQuery,
    newTitle,
    newUrl,
    newDescription,
    newTags,
    editing,
    editForm,
    allTags,
    filteredBookmarks,
    loadBookmarks,
    addBookmark,
    filterByTag,
    openEdit,
    saveEdit,
    removeBookmark,
  }
}

export function getInitial(value: string): string {
  return value.trim().charAt(0).toUpperCase() || 'B'
}

export function formatHost(url: string): string {
  try {
    return new URL(url).host
  } catch {
    return url
  }
}
