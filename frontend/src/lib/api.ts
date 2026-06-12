import type { Todo, Bookmark, BookmarkResponse, History, WalletEntry, User, ImportResult } from '../types'

const API_BASE = 'http://localhost:8080'

async function apiFetch<T>(method: string, path: string, body?: unknown): Promise<T> {
  const opts: RequestInit = {
    method,
    headers: { 'Content-Type': 'application/json' },
  }
  if (body !== undefined) {
    opts.body = JSON.stringify(body)
  }
  const res = await fetch(`${API_BASE}${path}`, opts)
  if (res.status === 204) return undefined as T
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `Request failed: ${res.status}`)
  }
  return res.json()
}

// ─── Todos ───────────────────────────────────────────────

export function getTodos(userId?: number): Promise<Todo[]> {
  const qs = userId ? `?user_id=${userId}` : ''
  return apiFetch<Todo[]>('GET', `/api/todos${qs}`)
}

export function getTodo(id: number): Promise<Todo> {
  return apiFetch<Todo>('GET', `/api/todos/${id}`)
}

export function createTodo(data: { user_id: number; title: string; description?: string }): Promise<Todo> {
  return apiFetch<Todo>('POST', '/api/todos', { ...data, completed: false })
}

export function updateTodo(id: number, data: Partial<Todo>): Promise<Todo> {
  return apiFetch<Todo>('PUT', `/api/todos/${id}`, data)
}

export function deleteTodo(id: number): Promise<void> {
  return apiFetch<void>('DELETE', `/api/todos/${id}`)
}

// ─── Bookmarks ───────────────────────────────────────────

export function getBookmarks(userId?: number, tags?: string): Promise<BookmarkResponse[]> {
  const params = new URLSearchParams()
  if (userId) params.set('user_id', String(userId))
  if (tags) params.set('tags', tags)
  const qs = params.toString()
  return apiFetch<BookmarkResponse[]>('GET', `/api/bookmarks${qs ? '?' + qs : ''}`)
}

export function getBookmark(id: number): Promise<BookmarkResponse> {
  return apiFetch<BookmarkResponse>('GET', `/api/bookmarks/${id}`)
}

export function createBookmark(data: { user_id: number; title: string; url: string; description?: string; tags?: string[] }): Promise<BookmarkResponse> {
  return apiFetch<BookmarkResponse>('POST', '/api/bookmarks', data)
}

export function updateBookmark(id: number, data: Partial<Bookmark>): Promise<BookmarkResponse> {
  return apiFetch<BookmarkResponse>('PUT', `/api/bookmarks/${id}`, data)
}

export function deleteBookmark(id: number): Promise<void> {
  return apiFetch<void>('DELETE', `/api/bookmarks/${id}`)
}

export function importBookmarks(userId: number, file: File): Promise<ImportResult> {
  const formData = new FormData()
  formData.append('file', file)
  return fetch(`${API_BASE}/api/bookmarks/import?user_id=${userId}`, {
    method: 'POST',
    body: formData,
  }).then(async (res) => {
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || `Import failed: ${res.status}`)
    }
    return res.json()
  })
}

// ─── History ─────────────────────────────────────────────

export function getHistory(userId?: number, url?: string): Promise<History[]> {
  const params = new URLSearchParams()
  if (userId) params.set('user_id', String(userId))
  if (url) params.set('url', url)
  const qs = params.toString()
  return apiFetch<History[]>('GET', `/api/history${qs ? '?' + qs : ''}`)
}

export function getHistoryEntry(id: number): Promise<History> {
  return apiFetch<History>('GET', `/api/history/${id}`)
}

export function createHistory(data: { user_id: number; url: string; title: string; duration?: number }): Promise<History> {
  return apiFetch<History>('POST', '/api/history', data)
}

export function deleteHistory(id: number): Promise<void> {
  return apiFetch<void>('DELETE', `/api/history/${id}`)
}

// ─── Wallet ──────────────────────────────────────────────

export function getWallet(userId?: number, website?: string): Promise<WalletEntry[]> {
  const params = new URLSearchParams()
  if (userId) params.set('user_id', String(userId))
  if (website) params.set('website', website)
  const qs = params.toString()
  return apiFetch<WalletEntry[]>('GET', `/api/wallet${qs ? '?' + qs : ''}`)
}

export function getWalletEntry(id: number): Promise<WalletEntry> {
  return apiFetch<WalletEntry>('GET', `/api/wallet/${id}`)
}

export function createWalletEntry(data: { user_id: number; website: string; username: string; password: string; description?: string }): Promise<WalletEntry> {
  return apiFetch<WalletEntry>('POST', '/api/wallet', data)
}

export function updateWalletEntry(id: number, data: Partial<WalletEntry>): Promise<WalletEntry> {
  return apiFetch<WalletEntry>('PUT', `/api/wallet/${id}`, data)
}

export function deleteWalletEntry(id: number): Promise<void> {
  return apiFetch<void>('DELETE', `/api/wallet/${id}`)
}

// ─── Users ───────────────────────────────────────────────

export function getUsers(): Promise<User[]> {
  return apiFetch<User[]>('GET', '/api/users')
}

export function getUser(id: number): Promise<User> {
  return apiFetch<User>('GET', `/api/users/${id}`)
}

export function createUser(data: { username: string; email?: string }): Promise<User> {
  return apiFetch<User>('POST', '/api/users', data)
}

export function deleteUser(id: number): Promise<void> {
  return apiFetch<void>('DELETE', `/api/users/${id}`)
}
