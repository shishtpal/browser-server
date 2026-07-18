import type { BookmarkResponse } from '@browser-server/shared-client'
import type { Node, Edge } from '@vue-flow/core'

// ─── Node data types ─────────────────────────────────────────────────────────

export interface FolderNodeData {
  type: 'folder'
  label: string
  path: string
  leafCount: number
  expanded: boolean
}

export interface BookmarkNodeData {
  type: 'bookmark'
  label: string
  bookmark: BookmarkResponse
  faviconUrl: string
}

export interface RootNodeData {
  type: 'root'
  label: string
  leafCount: number
}

export type GraphNodeData = FolderNodeData | BookmarkNodeData | RootNodeData

export type GraphNode = Node<GraphNodeData>
export type GraphEdge = Edge

// ─── Internal tree structure ─────────────────────────────────────────────────

export interface TreeNode {
  id: string
  name: string
  path: string
  type: 'root' | 'folder' | 'bookmark'
  bookmark: BookmarkResponse | null
  children: TreeNode[]
  leafCount: number
}

const UNFILED_KEY = '__unfiled__'

/**
 * Build a tree from the flat bookmark list, grouped by folder_path.
 */
function buildTree(bookmarks: BookmarkResponse[]): TreeNode {
  const root: TreeNode = {
    id: 'root',
    name: 'Bookmarks',
    path: '',
    type: 'root',
    bookmark: null,
    children: [],
    leafCount: 0,
  }

  const folderIndex = new Map<string, TreeNode>()
  folderIndex.set('', root)

  function ensureFolder(path: string): TreeNode {
    const existing = folderIndex.get(path)
    if (existing) return existing
    const parts = path.split('/').filter(Boolean)
    const name = parts[parts.length - 1] ?? path
    const parentPath = parts.slice(0, -1).join('/')
    const parent = ensureFolder(parentPath)
    const node: TreeNode = {
      id: `folder:${path}`,
      type: 'folder',
      name,
      path,
      bookmark: null,
      children: [],
      leafCount: 0,
    }
    parent.children.push(node)
    folderIndex.set(path, node)
    return node
  }

  let unfiled: TreeNode | null = null
  function getUnfiled(): TreeNode {
    if (unfiled) return unfiled
    unfiled = {
      id: `folder:${UNFILED_KEY}`,
      type: 'folder',
      name: 'Unfiled',
      path: '',
      bookmark: null,
      children: [],
      leafCount: 0,
    }
    root.children.push(unfiled)
    folderIndex.set(UNFILED_KEY, unfiled)
    return unfiled
  }

  for (const b of bookmarks) {
    const folderPath = b.folder_path?.trim() ?? ''
    const parent = folderPath ? ensureFolder(folderPath) : getUnfiled()
    const leaf: TreeNode = {
      id: `bookmark:${b.id}`,
      type: 'bookmark',
      name: b.title || b.url,
      path: folderPath,
      bookmark: b,
      children: [],
      leafCount: 0,
    }
    parent.children.push(leaf)
  }

  // Compute leaf counts and sort
  function computeLeafCount(node: TreeNode): number {
    node.children = node.children.filter(
      (c) => c.type === 'bookmark' || computeLeafCount(c) > 0,
    )
    node.children.sort((a, b) => {
      if (a.type !== b.type) return a.type === 'folder' ? -1 : 1
      return a.name.localeCompare(b.name)
    })
    const leaves = node.children.reduce(
      (sum, c) => sum + (c.type === 'bookmark' ? 1 : c.leafCount),
      0,
    )
    node.leafCount = leaves
    return leaves
  }
  computeLeafCount(root)

  // Collapse lone Unfiled bucket
  if (root.children.length === 1 && root.children[0].id === `folder:${UNFILED_KEY}`) {
    root.children = root.children[0].children
  }

  return root
}

// ─── Layout ──────────────────────────────────────────────────────────────────

export interface LayoutOptions {
  /** horizontal gap between tree levels */
  levelGap?: number
  /** vertical gap between sibling nodes */
  nodeGap?: number
}

/**
 * Convert the internal tree into Vue Flow nodes and edges.
 * Only shows expanded folders and their direct children.
 * When no folders are expanded (initial state), shows root + top-level folders only.
 */
export function buildFlowGraph(
  bookmarks: BookmarkResponse[],
  expandedFolders: Set<string>,
  options: LayoutOptions = {},
): { nodes: GraphNode[]; edges: GraphEdge[]; tree: TreeNode } {
  const tree = buildTree(bookmarks)
  const levelGap = options.levelGap ?? 300
  const nodeGap = options.nodeGap ?? 80

  const nodes: GraphNode[] = []
  const edges: GraphEdge[] = []

  // Track the Y position for each level for vertical distribution
  let globalY = 0

  function layoutNode(treeNode: TreeNode, depth: number, parentId: string | null): void {
    const x = depth * levelGap
    const y = globalY
    globalY += nodeGap

    if (treeNode.type === 'root') {
      nodes.push({
        id: treeNode.id,
        type: 'root',
        position: { x, y },
        data: {
          type: 'root',
          label: treeNode.name,
          leafCount: treeNode.leafCount,
        } as RootNodeData,
      })
    } else if (treeNode.type === 'folder') {
      const isExpanded = expandedFolders.has(treeNode.id)
      nodes.push({
        id: treeNode.id,
        type: 'folder',
        position: { x, y },
        data: {
          type: 'folder',
          label: treeNode.name,
          path: treeNode.path,
          leafCount: treeNode.leafCount,
          expanded: isExpanded,
        } as FolderNodeData,
      })
    } else {
      // bookmark
      const b = treeNode.bookmark!
      nodes.push({
        id: treeNode.id,
        type: 'bookmark',
        position: { x, y },
        data: {
          type: 'bookmark',
          label: treeNode.name,
          bookmark: b,
          faviconUrl: `https://www.google.com/s2/favicons?domain=${new URL(b.url).hostname}&sz=32`,
        } as BookmarkNodeData,
      })
    }

    if (parentId) {
      edges.push({
        id: `${parentId}->${treeNode.id}`,
        source: parentId,
        target: treeNode.id,
        type: 'smoothstep',
        animated: treeNode.type === 'folder',
      })
    }

    // Determine which children to show
    if (treeNode.type === 'root') {
      // Always show top-level children (folders)
      for (const child of treeNode.children) {
        if (child.type === 'folder') {
          layoutNode(child, depth + 1, treeNode.id)
        } else if (child.type === 'bookmark') {
          // Bookmarks directly under root (no folder) — only show if root is considered expanded
          layoutNode(child, depth + 1, treeNode.id)
        }
      }
    } else if (treeNode.type === 'folder' && expandedFolders.has(treeNode.id)) {
      for (const child of treeNode.children) {
        layoutNode(child, depth + 1, treeNode.id)
      }
    }
  }

  layoutNode(tree, 0, null)

  return { nodes, edges, tree }
}

/**
 * Get all folder IDs from a bookmark set (for expand-all functionality).
 */
export function getAllFolderIds(bookmarks: BookmarkResponse[]): string[] {
  const tree = buildTree(bookmarks)
  const ids: string[] = []
  function collect(node: TreeNode): void {
    if (node.type === 'folder') ids.push(node.id)
    for (const child of node.children) collect(child)
  }
  collect(tree)
  return ids
}

/**
 * Build a folder-only tree (no bookmark leaves) for the sidebar panel.
 */
export interface FolderTreeNode {
  id: string
  name: string
  path: string
  leafCount: number
  children: FolderTreeNode[]
}

export function buildFolderTree(bookmarks: BookmarkResponse[]): FolderTreeNode[] {
  const tree = buildTree(bookmarks)

  function toFolderTree(node: TreeNode): FolderTreeNode | null {
    if (node.type === 'bookmark') return null
    const children: FolderTreeNode[] = []
    for (const child of node.children) {
      const folderChild = toFolderTree(child)
      if (folderChild) children.push(folderChild)
    }
    return {
      id: node.id,
      name: node.name,
      path: node.path,
      leafCount: node.leafCount,
      children,
    }
  }

  // Return the root's folder children (skip root itself for the sidebar)
  const folders: FolderTreeNode[] = []
  for (const child of tree.children) {
    const f = toFolderTree(child)
    if (f) folders.push(f)
  }
  return folders
}
