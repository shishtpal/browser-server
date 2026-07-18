import type { BookmarkResponse } from '@browser-server/shared-client'

export type GraphNodeType = 'root' | 'folder' | 'bookmark'

export interface GraphNode {
  id: string
  type: GraphNodeType
  name: string
  path: string
  bookmark: BookmarkResponse | null
  children: GraphNode[]
  /** number of bookmark leaves under this node (folders only) */
  leafCount: number
}

export interface PositionedNode {
  node: GraphNode
  x: number
  y: number
  /** screen-space angle for radial layout */
  angle: number
  depth: number
}

export interface PositionedEdge {
  from: PositionedNode
  to: PositionedNode
}

export interface GraphLayout {
  nodes: PositionedNode[]
  edges: PositionedEdge[]
  /** bounding box of all node centers */
  bounds: { minX: number; minY: number; maxX: number; maxY: number }
}

const UNFILED_KEY = '__unfiled__'

/**
 * Build a nested graph tree from bookmarks grouped by `folder_path`.
 * Mirrors the split-on-"/" hierarchy used by the web app's bookmark tree,
 * but returns a plain recursive structure suitable for graph layout.
 */
export function buildGraphTree(bookmarks: BookmarkResponse[]): GraphNode {
  const root: GraphNode = {
    id: 'root',
    type: 'root',
    name: 'Bookmarks',
    path: '',
    bookmark: null,
    children: [],
    leafCount: 0,
  }

  const folderIndex = new Map<string, GraphNode>()
  folderIndex.set('', root)

  function ensureFolder(path: string): GraphNode {
    const existing = folderIndex.get(path)
    if (existing) return existing
    const parts = path.split('/').filter(Boolean)
    const name = parts[parts.length - 1] ?? path
    const parentPath = parts.slice(0, -1).join('/')
    const parent = ensureFolder(parentPath)
    const node: GraphNode = {
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

  // Unfiled bucket for bookmarks with no folder_path.
  let unfiled: GraphNode | null = null
  function getUnfiled(): GraphNode {
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
    const leaf: GraphNode = {
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

  // Compute leaf counts bottom-up and drop empty folders.
  function pruneAndCount(node: GraphNode): number {
    node.children = node.children.filter((c) => c.type === 'bookmark' || pruneAndCount(c) > 0)
    // Keep folder ordering stable; bookmarks after sub-folders for readability.
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
  pruneAndCount(root)

  // Collapse a lone Unfiled bucket: if it's the only top-level folder, hoist
  // its bookmarks directly under root for a cleaner graph.
  if (root.children.length === 1 && root.children[0].id === `folder:${UNFILED_KEY}`) {
    root.children = root.children[0].children
  }

  return root
}

export interface LayoutOptions {
  /** pixels between radial rings */
  ringGap?: number
  /** minimum angular slice (radians) per leaf */
  minLeafAngle?: number
  /** starting radius for first ring */
  baseRadius?: number
}

/**
 * Radial layered ("mind-map") layout: root at center, folders on concentric
 * rings, bookmarks as leaves on the outer ring. Each subtree is allotted an
 * angular slice proportional to its leaf count so siblings never overlap.
 */
export function layoutTree(root: GraphNode, options: LayoutOptions = {}): GraphLayout {
  const ringGap = options.ringGap ?? 220
  const baseRadius = options.baseRadius ?? 0
  const minLeafAngle = options.minLeafAngle ?? 0.08

  const nodes: PositionedNode[] = []
  const edges: PositionedEdge[] = []

  const rootPos: PositionedNode = {
    node: root,
    x: 0,
    y: 0,
    angle: 0,
    depth: 0,
  }
  nodes.push(rootPos)

  function totalLeafWeight(node: GraphNode): number {
    return Math.max(1, node.leafCount || node.children.length || 1)
  }

  function placeChildren(
    parent: PositionedNode,
    startAngle: number,
    endAngle: number,
  ) {
    const children = parent.node.children
    if (children.length === 0) return

    const depth = parent.depth + 1
    const radius = baseRadius + depth * ringGap
    const span = endAngle - startAngle
    const totalWeight = children.reduce(
      (sum, c) => sum + (c.type === 'bookmark' ? 1 : totalLeafWeight(c)),
      0,
    )
    let cursor = startAngle

    for (const child of children) {
      const weight = child.type === 'bookmark' ? 1 : totalLeafWeight(child)
      const slice = Math.max(span * (weight / totalWeight), minLeafAngle)
      const mid = cursor + slice / 2
      const x = parent.x + Math.cos(mid) * radius
      const y = parent.y + Math.sin(mid) * radius
      const childPos: PositionedNode = { node: child, x, y, angle: mid, depth }
      nodes.push(childPos)
      edges.push({ from: parent, to: childPos })

      if (child.type === 'folder') {
        // Recurse, reserving the inner part of the slice for grandchildren.
        placeChildren(childPos, mid - slice / 2, mid + slice / 2)
      }
      cursor += slice
    }
  }

  placeChildren(rootPos, 0, Math.PI * 2)

  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const n of nodes) {
    if (n.x < minX) minX = n.x
    if (n.y < minY) minY = n.y
    if (n.x > maxX) maxX = n.x
    if (n.y > maxY) maxY = n.y
  }

  return {
    nodes,
    edges,
    bounds: { minX, minY, maxX, maxY },
  }
}

export function buildLayout(bookmarks: BookmarkResponse[], options?: LayoutOptions): GraphLayout {
  return layoutTree(buildGraphTree(bookmarks), options)
}
