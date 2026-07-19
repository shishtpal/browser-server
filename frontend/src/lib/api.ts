import { createBrowserServerClient } from '@browser-server/shared-client'
import type {
  AIConfig,
  AIConversation,
  AIConversationDetail,
  AIStreamEvent,
  AIToolDecisionResponse,
  AnalyticsSummary,
  AnalyticsSummaryParams,
  Bookmark,
  BookmarkResponse,
  CreateAIConversationInput,
  CreateHistoryInput,
  CreateTodoInput,
  HealthResponse,
  History,
  HistoryImportResult,
  ImportResult,
  Screenshot,
  SendAIMessageInput,
  SendAIMessageResponse,
  StopAIGenerationResponse,
  Todo,
  UpdateAIConversationInput,
  User,
  WalletEntry,
  WalletImportResult,
} from '@browser-server/shared-types'

import { authHeaders, getToken } from './auth'

const API_BASE = 'http://localhost:9191'

const client = createBrowserServerClient(API_BASE, { getToken })

// ─── Health ─────────────────────────────────────────────

export function checkHealth(): Promise<HealthResponse> {
  return client.health()
}

export async function isServerOnline(): Promise<boolean> {
  return client.ping()
}

// ─── Todos ───────────────────────────────────────────────

export function getTodos(userId?: number, domain?: string): Promise<Todo[]> {
  return client.getTodos(userId, domain)
}

export function getTodo(id: number): Promise<Todo> {
  return fetch(`${API_BASE}/api/todos/${id}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<Todo>
  })
}

export function createTodo(data: CreateTodoInput): Promise<Todo> {
  return client.createTodo(data)
}

export function updateTodo(id: number, data: Partial<Todo>): Promise<Todo> {
  return client.updateTodo(id, data as Parameters<typeof client.updateTodo>[1])
}

export function deleteTodo(id: number): Promise<void> {
  return client.deleteTodo(id)
}

export function uploadScreenshot(todoId: number, file: Blob): Promise<Screenshot> {
  return client.uploadScreenshot(todoId, file)
}

export function getScreenshotUrl(todoId: number): string {
  return client.getScreenshotUrl(todoId)
}

// ─── Bookmarks ───────────────────────────────────────────

export function getBookmarks(userId?: number, tags?: string): Promise<BookmarkResponse[]> {
  const params = new URLSearchParams()
  if (userId) params.set('user_id', String(userId))
  if (tags) params.set('tags', tags)
  const qs = params.toString()
  return fetch(`${API_BASE}/api/bookmarks${qs ? `?${qs}` : ''}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<BookmarkResponse[]>
  })
}

export function getBookmark(id: number): Promise<BookmarkResponse> {
  return fetch(`${API_BASE}/api/bookmarks/${id}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<BookmarkResponse>
  })
}

export function createBookmark(data: { user_id: number; title: string; url: string; description?: string; tags?: string[] }): Promise<BookmarkResponse> {
  return fetch(`${API_BASE}/api/bookmarks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) {
      return res.text().then((text) => {
        throw new Error(text || `Request failed: ${res.status}`)
      })
    }
    return res.json() as Promise<BookmarkResponse>
  })
}

export function updateBookmark(id: number, data: Partial<Bookmark>): Promise<BookmarkResponse> {
  return fetch(`${API_BASE}/api/bookmarks/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<BookmarkResponse>
  })
}

export function deleteBookmark(id: number): Promise<void> {
  return fetch(`${API_BASE}/api/bookmarks/${id}`, { method: 'DELETE', headers: authHeaders() }).then((res) => {
    if (res.status === 204) return
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  })
}

export function importBookmarks(userId: number, file: File): Promise<ImportResult> {
  const formData = new FormData()
  formData.append('file', file)
  return fetch(`${API_BASE}/api/bookmarks/import?user_id=${userId}`, {
    method: 'POST',
    headers: authHeaders(),
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

export function getHistory(userId?: number, url?: string, limit?: number, offset?: number): Promise<History[]> {
  return client.getHistory(userId, url, limit, offset)
}

export function getHistoryEntry(id: number): Promise<History> {
  return fetch(`${API_BASE}/api/history/${id}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<History>
  })
}

export function createHistory(data: CreateHistoryInput): Promise<History> {
  return client.createHistory(data)
}

export function deleteHistory(id: number): Promise<void> {
  return client.deleteHistory(id)
}

export function importHistory(userId: number, file: File): Promise<HistoryImportResult> {
  const formData = new FormData()
  formData.append('file', file)
  return fetch(`${API_BASE}/api/history/import?user_id=${userId}`, {
    method: 'POST',
    headers: authHeaders(),
    body: formData,
  }).then(async (res) => {
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || `Import failed: ${res.status}`)
    }
    return res.json()
  })
}

// ─── Wallet ──────────────────────────────────────────────

export function getWallet(userId?: number, website?: string): Promise<WalletEntry[]> {
  const params = new URLSearchParams()
  if (userId) params.set('user_id', String(userId))
  if (website) params.set('website', website)
  const qs = params.toString()
  return fetch(`${API_BASE}/api/wallet${qs ? `?${qs}` : ''}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<WalletEntry[]>
  })
}

export function revealWalletPassword(userId: number, id: number): Promise<string> {
  const params = new URLSearchParams({ user_id: String(userId), id: String(id) })
  return fetch(`${API_BASE}/api/wallet/reveal?${params.toString()}`, { headers: authHeaders() }).then(async (res) => {
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || `Request failed: ${res.status}`)
    }
    return (await res.json() as { password: string }).password
  })
}

export function importWallet(userId: number, file: File): Promise<WalletImportResult> {
  const formData = new FormData()
  formData.append('file', file)
  return fetch(`${API_BASE}/api/wallet/import?user_id=${userId}`, {
    method: 'POST',
    headers: authHeaders(),
    body: formData,
  }).then(async (res) => {
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || `Import failed: ${res.status}`)
    }
    return res.json() as Promise<WalletImportResult>
  })
}

export function getWalletEntry(id: number): Promise<WalletEntry> {
  return fetch(`${API_BASE}/api/wallet/${id}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<WalletEntry>
  })
}

export function createWalletEntry(data: { user_id: number; website: string; username: string; password: string; login_provider?: string; description?: string }): Promise<WalletEntry> {
  return fetch(`${API_BASE}/api/wallet`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<WalletEntry>
  })
}

export function updateWalletEntry(id: number, data: Partial<WalletEntry>): Promise<WalletEntry> {
  return fetch(`${API_BASE}/api/wallet/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<WalletEntry>
  })
}

export function deleteWalletEntry(id: number): Promise<void> {
  return fetch(`${API_BASE}/api/wallet/${id}`, { method: 'DELETE', headers: authHeaders() }).then((res) => {
    if (res.status === 204) return
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  })
}

// ─── Users ───────────────────────────────────────────────

export function getUsers(): Promise<User[]> {
  return fetch(`${API_BASE}/api/users`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<User[]>
  })
}

export function getUser(id: number): Promise<User> {
  return fetch(`${API_BASE}/api/users/${id}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<User>
  })
}

export function createUser(data: { username: string; email?: string }): Promise<User> {
  return fetch(`${API_BASE}/api/users`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<User>
  })
}

export function deleteUser(id: number): Promise<void> {
  return fetch(`${API_BASE}/api/users/${id}`, { method: 'DELETE', headers: authHeaders() }).then((res) => {
    if (res.status === 204) return
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  })
}

// ─── Analytics ───────────────────────────────────────────

export function getAnalyticsSummary(params: AnalyticsSummaryParams): Promise<AnalyticsSummary> {
  return client.getAnalyticsSummary(params)
}

// ─── AI Chat ────────────────────────────────────────────

export function getAIConfig(): Promise<AIConfig> {
  return client.getAIConfig()
}

export function listAIConversations(query?: string, limit?: number): Promise<AIConversation[]> {
  return client.listAIConversations(query, limit)
}

export function createAIConversation(data: CreateAIConversationInput = {}): Promise<AIConversation> {
  return client.createAIConversation(data)
}

export function getAIConversation(id: string): Promise<AIConversationDetail> {
  return client.getAIConversation(id)
}

export function updateAIConversation(id: string, data: UpdateAIConversationInput): Promise<AIConversation> {
  return client.updateAIConversation(id, data)
}

export function deleteAIConversation(id: string): Promise<void> {
  return client.deleteAIConversation(id)
}

export function sendAIMessage(id: string, data: SendAIMessageInput): Promise<SendAIMessageResponse> {
  return client.sendAIMessage(id, data)
}

export function sendAIMessageStream(
  id: string,
  data: SendAIMessageInput,
  onEvent: (event: AIStreamEvent) => void,
  onError?: (err: Error) => void,
): AbortController {
  return client.sendAIMessageStream(id, data, onEvent, onError)
}

export function regenerateAIMessage(id: string): Promise<SendAIMessageResponse> {
  return client.regenerateAIMessage(id)
}

export function decideAIToolCall(id: string, callId: string, approved: boolean): Promise<AIToolDecisionResponse> {
  return client.decideAIToolCall(id, callId, approved)
}

export function stopAIGeneration(id: string): Promise<StopAIGenerationResponse> {
  return client.stopAIGeneration(id)
}
