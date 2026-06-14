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

export interface WalletEntry {
  id: number
  user_id: number
  username: string
  password: string
  website: string
  description: string
  created_at: string
  updated_at: string
}

export interface WalletImportResult {
  imported: number
  skipped: number
  errors: string[]
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
