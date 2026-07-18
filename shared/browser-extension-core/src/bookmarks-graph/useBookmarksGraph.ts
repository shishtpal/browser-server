import type { BookmarkResponse } from '@browser-server/shared-client'
import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import { createApiClient, useExtensionSettings, useUserId, useBookmarksView } from '../composables/composables'
import { buildLayout, type GraphLayout, type PositionedNode } from './buildGraph'

export type SelectionKind = 'bookmark' | 'folder' | 'root' | null

export interface GraphSelection {
  kind: SelectionKind
  node: PositionedNode | null
}

export interface UseBookmarksGraphReturn {
  isReady: ComputedRef<boolean>
  isLoading: Ref<boolean>
  errorMessage: Ref<string | null>
  bookmarks: Ref<BookmarkResponse[]>
  allTags: ComputedRef<string[]>
  layout: ComputedRef<GraphLayout>
  selection: Ref<GraphSelection>
  searchQuery: Ref<string>
  activeTag: Ref<string>
  load(): Promise<void>
  refresh(): Promise<void>
  select(node: PositionedNode | null): void
  clearSelection(): void
  updateBookmark(id: number, data: {
    user_id: number
    title: string
    url: string
    description?: string
    tags?: string[]
    folder_path?: string
  }): Promise<void>
  moveBookmark(id: number, folderPath: string): Promise<void>
  deleteBookmark(id: number): Promise<void>
  cleanup(): void
}

/**
 * Graph view-model. Reuses `useBookmarksView` as the data layer (load,
 * filter, create/update/delete) and layers graph-specific concerns on top:
 * radial layout computed from the filtered set, selection state, and a
 * `moveBookmark` helper that updates only `folder_path`.
 */
export function useBookmarksGraph(): UseBookmarksGraphReturn {
  const { settings } = useExtensionSettings()
  const userId = useUserId(computed(() => settings.value))
  const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

  const {
    items,
    filtered,
    errorMessage,
    isLoading,
    searchQuery,
    activeTag,
    allTags,
    load,
    updateBookmark,
    deleteBookmark,
  } = useBookmarksView(client, userId)

  const selection = ref<GraphSelection>({ kind: null, node: null })

  const isReady = computed(() => Boolean(client.value) && userId.value > 0)

  const layout = computed(() => buildLayout(filtered.value))

  function select(node: PositionedNode | null): void {
    if (!node) {
      selection.value = { kind: null, node: null }
      return
    }
    selection.value = { kind: node.node.type, node }
  }

  function clearSelection(): void {
    selection.value = { kind: null, node: null }
  }

  async function moveBookmark(id: number, folderPath: string): Promise<void> {
    const target = items.value.find((b) => b.id === id)
    if (!target) return
    await updateBookmark(id, {
      user_id: target.user_id,
      title: target.title,
      url: target.url,
      description: target.description,
      tags: target.tags,
      folder_path: folderPath,
    })
  }

  async function refresh(): Promise<void> {
    await load()
  }

  function cleanup(): void {
    // Watchers live in useBookmarksView/useExtensionSettings; nothing local
    // to dispose. Kept for symmetry with useHistoryBrowser.
  }

  watch([client, userId], () => {
    clearSelection()
    if (isReady.value) void load()
  }, { immediate: true })

  return {
    isReady,
    isLoading,
    errorMessage,
    bookmarks: items,
    allTags,
    layout,
    selection,
    searchQuery,
    activeTag,
    load,
    refresh,
    select,
    clearSelection,
    updateBookmark,
    moveBookmark,
    deleteBookmark,
    cleanup,
  }
}
