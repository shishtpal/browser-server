import type { Todo, Bookmark, BookmarkResponse, History, WalletEntry, User } from '../types'

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
  return apiFetch<Todo[]>('GET', `/todos${qs}`)
}

export function getTodo(id: number): Promise<Todo> {
  return apiFetch<Todo>('GET', `/todos/${id}`)
}

export function createTodo(data: { user_id: number; title: string; description?: string }): Promise<Todo> {
  return apiFetch<Todo>('POST', '/todos', { ...data, completed: false })
}

export function updateTodo(id: number, data: Partial<Todo>): Promise<Todo> {
  return apiFetch<Todo>('PUT', `/todos/${id}`, data)
}

export function deleteTodo(id: number): Promise<void> {
  return apiFetch<void>('DELETE', `/todos/${id}`)
}

// ─── Bookmarks ───────────────────────────────────────────

export function getBookmarks(userId?: number, tags?: string): Promise<BookmarkResponse[]> {
  const params = new URLSearchParams()
  if (userId) params.set('user_id', String(userId))
  if (tags) params.set('tags', tags)
  const qs = params.toString()
  return apiFetch<BookmarkResponse[]>('GET', `/bookmarks${qs ? '?' + qs : ''}`)
}

export function getBookmark(id: number): Promise<BookmarkResponse> {
  return apiFetch<BookmarkResponse>('GET', `/bookmarks/${id}`)
}

export function createBookmark(data: { user_id: number; title: string; url: string; description?: string; tags?: string[] }): Promise<BookmarkResponse> {
  return apiFetch<BookmarkResponse>('POST', '/bookmarks', data)
}

export function updateBookmark(id: number, data: Partial<Bookmark>): Promise<BookmarkResponse> {
  return apiFetch<BookmarkResponse>('PUT', `/bookmarks/${id}`, data)
}

export function deleteBookmark(id: number): Promise<void> {
  return apiFetch<void>('DELETE', `/bookmarks/${id}`)
}

// ─── History ─────────────────────────────────────────────

export function getHistory(userId?: number, url?: string): Promise<History[]> {
  const params = new URLSearchParams()
  if (userId) params.set('user_id', String(userId))
  if (url) params.set('url', url)
  const qs = params.toString()
  return apiFetch<History[]>('GET', `/history${qs ? '?' + qs : ''}`)
}

export function getHistoryEntry(id: number): Promise<History> {
  return apiFetch<History>('GET', `/history/${id}`)
}

export function createHistory(data: { user_id: number; url: string; title: string; duration?: number }): Promise<History> {
  return apiFetch<History>('POST', '/history', data)
}

export function deleteHistory(id: number): Promise<void> {
  return apiFetch<void>('DELETE', `/history/${id}`)
}

// ─── Wallet ──────────────────────────────────────────────

export function getWallet(userId?: number, website?: string): Promise<WalletEntry[]> {
  const params = new URLSearchParams()
  if (userId) params.set('user_id', String(userId))
  if (website) params.set('website', website)
  const qs = params.toString()
  return apiFetch<WalletEntry[]>('GET', `/wallet${qs ? '?' + qs : ''}`)
}

export function getWalletEntry(id: number): Promise<WalletEntry> {
  return apiFetch<WalletEntry>('GET', `/wallet/${id}`)
}

export function createWalletEntry(data: { user_id: number; website: string; username: string; password: string; description?: string }): Promise<WalletEntry> {
  return apiFetch<WalletEntry>('POST', '/wallet', data)
}

export function updateWalletEntry(id: number, data: Partial<WalletEntry>): Promise<WalletEntry> {
  return apiFetch<WalletEntry>('PUT', `/wallet/${id}`, data)
}

export function deleteWalletEntry(id: number): Promise<void> {
  return apiFetch<void>('DELETE', `/wallet/${id}`)
}

// ─── Users ───────────────────────────────────────────────

export function getUsers(): Promise<User[]> {
  return apiFetch<User[]>('GET', '/users')
}

export function getUser(id: number): Promise<User> {
  return apiFetch<User>('GET', `/users/${id}`)
}

export function createUser(data: { username: string; email?: string }): Promise<User> {
  return apiFetch<User>('POST', '/users', data)
}

export function deleteUser(id: number): Promise<void> {
  return apiFetch<void>('DELETE', `/users/${id}`)
}
