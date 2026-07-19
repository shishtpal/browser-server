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
