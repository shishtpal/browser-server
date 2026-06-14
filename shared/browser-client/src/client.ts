import type {
  CreateHistoryInput,
  CreateTodoInput,
  History,
  Screenshot,
  Todo,
  UpdateTodoInput,
  WalletEntry,
} from '@browser-server/shared-types'

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

function normalizeBaseUrl(baseUrl: string): string {
  return baseUrl.replace(/\/+$/, '')
}

function buildQuery(params: Record<string, string | number | undefined>): string {
  const searchParams = new URLSearchParams()

  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') {
      searchParams.set(key, String(value))
    }
  }

  const query = searchParams.toString()
  return query ? `?${query}` : ''
}

async function apiFetch<T>(baseUrl: string, method: HttpMethod, path: string, body?: unknown): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, {
    method,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })

  if (response.status === 204) {
    return undefined as T
  }

  if (!response.ok) {
    const text = await response.text()
    throw new Error(text || `Request failed: ${response.status}`)
  }

  return response.json() as Promise<T>
}

export type BrowserServerClient = ReturnType<typeof createBrowserServerClient>

export function createBrowserServerClient(baseUrl: string) {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl)

  return {
    getHistory(userId?: number, url?: string): Promise<History[]> {
      return apiFetch<History[]>(normalizedBaseUrl, 'GET', `/api/history${buildQuery({ user_id: userId, url })}`)
    },

    createHistory(data: CreateHistoryInput): Promise<History> {
      return apiFetch<History>(normalizedBaseUrl, 'POST', '/api/history', data)
    },

    deleteHistory(id: number): Promise<void> {
      return apiFetch<void>(normalizedBaseUrl, 'DELETE', `/api/history/${id}`)
    },

    getTodos(userId?: number, domain?: string): Promise<Todo[]> {
      return apiFetch<Todo[]>(normalizedBaseUrl, 'GET', `/api/todos${buildQuery({ user_id: userId, domain })}`)
    },

    createTodo(data: CreateTodoInput): Promise<Todo> {
      return apiFetch<Todo>(normalizedBaseUrl, 'POST', '/api/todos', { ...data, completed: false })
    },

    updateTodo(id: number, data: UpdateTodoInput): Promise<Todo> {
      return apiFetch<Todo>(normalizedBaseUrl, 'PUT', `/api/todos/${id}`, data)
    },

    deleteTodo(id: number): Promise<void> {
      return apiFetch<void>(normalizedBaseUrl, 'DELETE', `/api/todos/${id}`)
    },

    async uploadScreenshot(todoId: number, file: Blob): Promise<Screenshot> {
      const formData = new FormData()
      formData.append('file', file, 'screenshot.png')

      const response = await fetch(`${normalizedBaseUrl}/api/screenshots?todo_id=${todoId}`, {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) {
        const text = await response.text()
        throw new Error(text || `Upload failed: ${response.status}`)
      }

      return response.json() as Promise<Screenshot>
    },

    getScreenshotUrl(todoId: number): string {
      return `${normalizedBaseUrl}/api/screenshots/${todoId}`
    },

    getWallet(userId?: number, website?: string): Promise<WalletEntry[]> {
      return apiFetch<WalletEntry[]>(normalizedBaseUrl, 'GET', `/api/wallet${buildQuery({ user_id: userId, website })}`)
    },

    async revealWalletPassword(userId: number, website: string, username: string): Promise<string> {
      const result = await apiFetch<{ password: string }>(
        normalizedBaseUrl,
        'GET',
        `/api/wallet/reveal${buildQuery({ user_id: userId, website, username })}`,
      )
      return result.password
    },
  }
}
