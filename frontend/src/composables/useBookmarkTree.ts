import { ref, computed, type Ref } from 'vue'
import type { BookmarkResponse } from '../types'

export interface TreeFolder {
  name: string
  children: Map<string, TreeFolder>
  bookmarks: BookmarkResponse[]
  count: number
}

export interface FlatTreeEntry {
  key: string
  name: string
  depth: number
  type: 'folder' | 'bookmark'
  expanded: boolean
  visible: boolean
  bookmark: BookmarkResponse | null
  count: number
}

function buildTree(bms: BookmarkResponse[]): Map<string, TreeFolder> {
  const root = new Map<string, TreeFolder>()
  for (const b of bms) {
    const parts = b.folder_path ? b.folder_path.split('/').filter(Boolean) : []
    if (parts.length === 0) {
      const f = root.get('\0') || { name: 'Unfiled', children: new Map(), bookmarks: [] as BookmarkResponse[], count: 0 }
      f.bookmarks.push(b)
      f.count++
      root.set('\0', f)
      continue
    }
    let cur = root
    for (let i = 0; i < parts.length; i++) {
      const p = parts[i]
      if (!cur.has(p)) cur.set(p, { name: p, children: new Map(), bookmarks: [] as BookmarkResponse[], count: 0 })
      const f = cur.get(p)!
      f.count++
      if (i === parts.length - 1) f.bookmarks.push(b)
      cur = f.children
    }
  }
  return root
}

function flattenFolder(
  node: TreeFolder,
  depth: number,
  parentVisible: boolean,
  parentKey: string,
  result: FlatTreeEntry[],
  searching: boolean,
  expandedFolders: Set<string>,
) {
  if (node.name === 'Unfiled' && depth === 0) {
    for (const bm of node.bookmarks) {
      result.push(makeBookmarkEntry(bm, 0, true))
    }
    return
  }
  const key = parentKey ? parentKey + '/' + node.name : node.name
  const hasBms = node.bookmarks.length > 0
  let expanded = expandedFolders.has(key)
  if (searching && hasBms) expanded = true
  const vis = parentVisible
  if (hasBms || node.children.size > 0) {
    result.push({ key, name: node.name, depth, type: 'folder', expanded, visible: vis, bookmark: null, count: node.count })
  }
  const childKeys = [...node.children.keys()].sort()
  for (const k of childKeys) {
    flattenFolder(node.children.get(k)!, depth + 1, vis && expanded, key, result, searching, expandedFolders)
  }
  if (vis && expanded) {
    for (const bm of node.bookmarks) {
      result.push(makeBookmarkEntry(bm, depth + 1, true))
    }
  }
}

function makeBookmarkEntry(bm: BookmarkResponse, depth: number, visible: boolean): FlatTreeEntry {
  return { key: 'bm-' + bm.id, name: bm.title, depth, type: 'bookmark', expanded: false, visible, bookmark: bm, count: 0 }
}

export function useBookmarkTree(
  filteredBookmarks: Ref<BookmarkResponse[]>,
  searchQuery: Ref<string>,
) {
  const viewMode = ref<'flat' | 'tree'>('flat')
  const expandedFolders = ref<Set<string>>(new Set())

  const treeNodes = computed(() => {
    const tree = buildTree(filteredBookmarks.value)
    const searching = searchQuery.value.trim().length > 0
    const result: FlatTreeEntry[] = []
    const entries = [...tree.entries()]
    entries.sort((a, b) => {
      if (a[0] === '\0') return 1
      if (b[0] === '\0') return -1
      return a[0].localeCompare(b[0])
    })
    for (const [, folder] of entries) {
      flattenFolder(folder, 0, true, '', result, searching, expandedFolders.value)
    }
    return result.filter(n => n.visible)
  })

  const treeCount = computed(() =>
    treeNodes.value.filter(n => n.type === 'bookmark' && n.visible).length
  )

  const toggleTreeFolder = (key: string) => {
    const s = new Set(expandedFolders.value)
    if (s.has(key)) {
      s.delete(key)
    } else {
      s.add(key)
    }
    expandedFolders.value = s
  }

  return {
    viewMode,
    treeNodes,
    treeCount,
    toggleTreeFolder,
  }
}
