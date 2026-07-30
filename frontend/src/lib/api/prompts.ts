import type { CreatePromptInput, PromptResponse, UpdatePromptInput } from '@browser-server/shared-types'
import { API_BASE, authHeaders } from './client'

export function getPrompts(userId: number, query?: string): Promise<PromptResponse[]> {
  const params = new URLSearchParams({ user_id: String(userId) })
  if (query) params.set('q', query)
  const qs = params.toString()
  return fetch(`${API_BASE}/api/prompts?${qs}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<PromptResponse[]>
  })
}

export function createPrompt(data: CreatePromptInput): Promise<PromptResponse> {
  return fetch(`${API_BASE}/api/prompts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<PromptResponse>
  })
}

export function updatePrompt(id: number, data: UpdatePromptInput): Promise<PromptResponse> {
  return fetch(`${API_BASE}/api/prompts/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<PromptResponse>
  })
}

export function deletePrompt(id: number): Promise<void> {
  return fetch(`${API_BASE}/api/prompts/${id}`, { method: 'DELETE', headers: authHeaders() }).then((res) => {
    if (res.status === 204) return
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  })
}

export function searchPrompts(userId: number, query: string, limit = 10): Promise<PromptResponse[]> {
  const params = new URLSearchParams({ user_id: String(userId), q: query, limit: String(limit) })
  return fetch(`${API_BASE}/api/prompts/search?${params.toString()}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<PromptResponse[]>
  })
}
