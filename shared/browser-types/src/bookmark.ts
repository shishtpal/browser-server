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

export interface CreateBookmarkInput {
  user_id: number
  title: string
  url: string
  description?: string
  tags?: string[]
  folder_path?: string
  capture_id?: string
}

export interface UpdateBookmarkInput {
  user_id: number
  title: string
  url: string
  description?: string
  tags?: string[]
  folder_path?: string
}
