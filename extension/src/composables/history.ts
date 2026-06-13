import type { History } from '@browser-server/shared-client'

export interface GroupedHistory {
  url: string
  title: string
  count: number
  lastVisited: string
}

export function summarizeHistory(entries: History[]): GroupedHistory[] {
  const grouped = new Map<string, GroupedHistory>()

  for (const entry of entries) {
    const current = grouped.get(entry.url)
    if (!current) {
      grouped.set(entry.url, {
        url: entry.url,
        title: entry.title || entry.url,
        count: 1,
        lastVisited: entry.visited_at,
      })
      continue
    }

    current.count += 1
    if (Date.parse(entry.visited_at) > Date.parse(current.lastVisited)) {
      current.lastVisited = entry.visited_at
      current.title = entry.title || current.title
    }
  }

  return Array.from(grouped.values()).sort(
    (left, right) => Date.parse(right.lastVisited) - Date.parse(left.lastVisited),
  )
}
