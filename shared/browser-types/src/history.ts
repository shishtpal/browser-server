export interface History {
  id: number
  user_id: number
  url: string
  title: string
  visited_at: string
  duration: number
}

export interface HistoryImportResult {
  imported: number
  skipped: number
  errors: string[]
}

export type HistorySearchColumn = 'all' | 'title' | 'url'

export interface GroupedHistoryEntry {
  url: string
  title: string
  count: number
  total_duration: number
  last_visited: string
}

export interface GroupedHistoryResponse {
  entries: GroupedHistoryEntry[]
  total: number
  limit: number
  offset: number
}

export interface GroupedHistoryParams {
  user_id?: number
  domain?: string
  q?: string
  column?: HistorySearchColumn
  limit?: number
  offset?: number
}

export interface HistoryDomainSummary {
  domain: string
  visit_count: number
  url_count: number
  total_duration: number
  last_visited: string
}

export interface CreateHistoryInput {
  user_id: number
  url: string
  title: string
  duration?: number
}
