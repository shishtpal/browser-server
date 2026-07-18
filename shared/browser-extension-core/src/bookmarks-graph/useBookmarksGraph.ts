import type { BookmarkResponse } from '@browser-server/shared-client'
import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import { createApiClient, useExtensionSettings, useUserId, useBookmarksView } from '../composables/composables'
import { buildFlowGraph, buildFolderTree, getAllFolderIds, type GraphNode, type GraphEdge, type FolderTreeNode } from './buildGraph'

export interface UseBookmarksGraphReturn {
  isReady: ComputedRef<boolean>
  isLoading: Ref<boolean>
  errorMessage: Ref<string | null>
  bookmarks: Ref<BookmarkResponse[]>
  allTags: ComputedRef<string[]>
  nodes: ComputedRef<GraphNode[]>
  edges: ComputedRef<GraphEdge[]>
  folderTree: ComputedRef<FolderTreeNode[]>
  expandedFolders: Ref<Set<string>>
  searchQuery: Ref<string>
  activeTag: Ref<string>
  selectedBookmark: Ref<BookmarkResponse | null>
  load(): Promise<void>
  refresh(): Promise<void>
  toggleFolder(folderId: string): void
  expandAll(): void
  collapseAll(): void
  selectBookmark(bookmark: BookmarkResponse | null): void
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
 * Graph view-model using Vue Flow. Handles data fetching, filtering,
 * folder expand/collapse state, and bookmark CRUD operations.
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

  const expandedFolders = ref<Set<string>>(new Set())
  const selectedBookmark = ref<BookmarkResponse | null>(null)

  const isReady = computed(() => Boolean(client.value) && userId.value > 0)

  // When searching/filtering, auto-expand all folders so results are visible
  const effectiveExpanded = computed(() => {
    if (searchQuery.value.trim() || activeTag.value) {
      return new Set(getAllFolderIds(filtered.value))
    }
    return expandedFolders.value
  })

  const graphData = computed(() => buildFlowGraph(filtered.value, effectiveExpanded.value))

  const nodes = computed(() => graphData.value.nodes)
  const edges = computed(() => graphData.value.edges)
  const folderTree = computed(() => buildFolderTree(filtered.value))

  function toggleFolder(folderId: string): void {
    const next = new Set(expandedFolders.value)
    if (next.has(folderId)) {
      next.delete(folderId)
    } else {
      next.add(folderId)
    }
    expandedFolders.value = next
  }

  function expandAll(): void {
    expandedFolders.value = new Set(getAllFolderIds(filtered.value))
  }

  function collapseAll(): void {
    expandedFolders.value = new Set()
  }

  function selectBookmark(bookmark: BookmarkResponse | null): void {
    selectedBookmark.value = bookmark
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
    // Nothing to dispose — Vue's reactivity handles teardown.
  }

  watch([client, userId], () => {
    expandedFolders.value = new Set()
    selectedBookmark.value = null
    if (isReady.value) void load()
  }, { immediate: true })

  return {
    isReady,
    isLoading,
    errorMessage,
    bookmarks: items,
    allTags,
    nodes,
    edges,
    folderTree,
    expandedFolders,
    searchQuery,
    activeTag,
    selectedBookmark,
    load,
    refresh,
    toggleFolder,
    expandAll,
    collapseAll,
    selectBookmark,
    updateBookmark,
    moveBookmark,
    deleteBookmark,
    cleanup,
  }
}
