export interface HealthResponse {
  status: string
  uptime_seconds: number
}

export interface Todo {
  id: number
  user_id: number
  title: string
  description: string
  domain: string
  screenshot_path: string
  completed: boolean
  created_at: string
  updated_at: string
}

export interface Screenshot {
  id: number
  todo_id: number
  filename: string
  created_at: string
}

export interface Bookmark {
  id: number
  user_id: number
  title: string
  url: string
  description: string
  tags: string[]
  folder_path: string
  created_at: string
  updated_at: string
}

export interface BookmarkResponse {
  id: number
  user_id: number
  title: string
  url: string
  description: string
  tags: string[]
  folder_path: string
  created_at: string
  updated_at: string
}

export interface ImportResult {
  imported: number
  skipped: number
  bookmarks: BookmarkResponse[]
}

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
  q?: string
  column?: HistorySearchColumn
  limit?: number
  offset?: number
}

export interface OmniboxSearchResult {
  source: 'history' | 'bookmark'
  title: string
  url: string
  description?: string
  tags?: string[]
  folder_path?: string
  visit_count?: number
  last_visited?: string
  updated_at?: string
}

export interface OmniboxSearchParams {
  user_id?: number
  q: string
  limit?: number
}

export interface WalletEntry {
  id: number
  user_id: number
  username: string
  password: string
  website: string
  login_provider: string
  description: string
  created_at: string
  updated_at: string
}

export interface WalletImportResult {
  imported: number
  skipped: number
  errors: string[]
}

export interface CreateWalletInput {
  user_id: number
  website: string
  login_provider?: string
  username: string
  password: string
  description?: string
}

export interface User {
  id: number
  username: string
  email: string
}

export interface Route {
  method: string
  path: string
  description: string
}

export interface CreateTodoInput {
  user_id: number
  title: string
  description?: string
  domain?: string
  capture_id?: string
}

export interface UpdateTodoInput {
  user_id?: number
  title?: string
  description?: string
  domain?: string
  completed?: boolean
  screenshot_path?: string
}

export interface CreateHistoryInput {
  user_id: number
  url: string
  title: string
  duration?: number
}

export interface CreateBookmarkInput {
  user_id: number
  title: string
  url: string
  description?: string
  tags?: string[]
  folder_path?: string
  capture_id?: string
}

export interface UpdateWalletInput {
  username?: string
  password?: string
  website?: string
  login_provider?: string
  description?: string
}

export interface UsageEntry {
  domain: string
  date: string
  seconds: number
}

export interface UsageBatchRequest {
  user_id: number
  entries: UsageEntry[]
}

export interface UsageBatchResponse {
  upserted: number
}

export interface DomainUsage {
  domain: string
  total_seconds: number
  percentage: number
}

export interface TimelinePoint {
  period: string
  total_seconds: number
}

export interface AnalyticsSummary {
  total_seconds: number
  domains: DomainUsage[]
  timeline: TimelinePoint[]
}

export interface AnalyticsSummaryParams {
  user_id: number
  start_date: string
  end_date: string
  group_by?: 'day' | 'week' | 'month'
  limit?: number
}
