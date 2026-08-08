import type { User } from '@browser-server/shared-types'
import { API_BASE, authHeaders } from './client'

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
  return fetch(`${API_BASE}/api/users/${id}`, { method: 'DELETE', headers: authHeaders() }).then(
    (res) => {
      if (res.status === 204) return
      if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    },
  )
}
