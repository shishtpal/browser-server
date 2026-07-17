import type {
  AnalyticsSummary,
  AnalyticsSummaryParams,
  BookmarkResponse,
  CreateBookmarkInput,
  CreateHistoryInput,
  CreateTodoInput,
  CreateWalletInput,
  GroupedHistoryParams,
  GroupedHistoryResponse,
  HealthResponse,
  History,
  HistoryDomainSummary,
  OmniboxSearchParams,
  OmniboxSearchResult,
  Screenshot,
  Todo,
  UpdateTodoInput,
  UpdateWalletInput,
  UsageBatchRequest,
  UsageBatchResponse,
  WalletEntry,
} from '@browser-server/shared-types'

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

/** Resolves the current API token (or null/undefined when none is set). */
export type TokenProvider = () => string | null | undefined

export interface BrowserServerClientOptions {
  /** Called on every request to obtain the bearer token to send. */
  getToken?: TokenProvider
}

/** Error thrown for non-OK API responses, carrying the HTTP status code. */
export class ApiError extends Error {
  readonly status: number
  /** Field-level validation errors keyed by JSON field name, when present. */
  readonly fields?: Record<string, string>

  constructor(status: number, message: string, fields?: Record<string, string>) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.fields = fields
  }
}

/**
 * Builds an ApiError from a raw response body. The server returns a standard
 * JSON envelope ({ error, fields? }); fall back to the raw text for anything
 * that isn't that shape.
 */
function apiErrorFromBody(status: number, body: string, fallback: string): ApiError {
  if (body) {
    try {
      const parsed = JSON.parse(body) as { error?: string; fields?: Record<string, string> }
      if (parsed && typeof parsed.error === 'string') {
        return new ApiError(status, parsed.error, parsed.fields)
      }
    } catch {
      // Not JSON; use the raw text below.
    }
    return new ApiError(status, body)
  }
  return new ApiError(status, fallback)
}

function normalizeBaseUrl(baseUrl: string): string {
  return baseUrl.replace(/\/+$/, '')
}

function authHeader(getToken?: TokenProvider): Record<string, string> {
  const token = getToken?.()
  return token ? { Authorization: `Bearer ${token}` } : {}
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

async function apiFetch<T>(
  baseUrl: string,
  method: HttpMethod,
  path: string,
  body?: unknown,
  getToken?: TokenProvider,
): Promise<T> {
  const headers: Record<string, string> = { ...authHeader(getToken) }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  const response = await fetch(`${baseUrl}${path}`, {
    method,
    headers: Object.keys(headers).length > 0 ? headers : undefined,
    body: body === undefined ? undefined : JSON.stringify(body),
  })

  if (response.status === 204) {
    return undefined as T
  }

  if (!response.ok) {
    const text = await response.text()
    throw apiErrorFromBody(response.status, text, `Request failed: ${response.status}`)
  }

  return response.json() as Promise<T>
}

export type BrowserServerClient = ReturnType<typeof createBrowserServerClient>

export function createBrowserServerClient(baseUrl: string, options: BrowserServerClientOptions = {}) {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl)
  const { getToken } = options

  return {
    async ping(): Promise<boolean> {
      try {
        const controller = new AbortController()
        const timeout = setTimeout(() => controller.abort(), 3000)
        const response = await fetch(`${normalizedBaseUrl}/health`, {
          method: 'GET',
          signal: controller.signal,
        })
        clearTimeout(timeout)
        return response.ok
      } catch {
        return false
      }
    },

    async health(): Promise<HealthResponse> {
      return apiFetch<HealthResponse>(normalizedBaseUrl, 'GET', '/health')
    },

    searchOmnibox(params: OmniboxSearchParams): Promise<OmniboxSearchResult[]> {
      return apiFetch<OmniboxSearchResult[]>(
        normalizedBaseUrl,
        'GET',
        `/api/search/omnibox${buildQuery(params as unknown as Record<string, string | number | undefined>)}`,
        undefined,
        getToken,
      )
    },

    getHistory(userId?: number, url?: string, limit?: number, offset?: number): Promise<History[]> {
      return apiFetch<History[]>(normalizedBaseUrl, 'GET', `/api/history${buildQuery({ user_id: userId, url, limit, offset })}`, undefined, getToken)
    },

    getGroupedHistory(params: GroupedHistoryParams): Promise<GroupedHistoryResponse> {
      return apiFetch<GroupedHistoryResponse>(
        normalizedBaseUrl,
        'GET',
        `/api/history/grouped${buildQuery(params as unknown as Record<string, string | number | undefined>)}`,
        undefined,
        getToken,
      )
    },

    getHistoryDomains(userId?: number, query?: string): Promise<HistoryDomainSummary[]> {
      return apiFetch<HistoryDomainSummary[]>(
        normalizedBaseUrl,
        'GET',
        `/api/history/domains${buildQuery({ user_id: userId, q: query })}`,
        undefined,
        getToken,
      )
    },

    createHistory(data: CreateHistoryInput): Promise<History> {
      return apiFetch<History>(normalizedBaseUrl, 'POST', '/api/history', data, getToken)
    },

    deleteHistory(id: number): Promise<void> {
      return apiFetch<void>(normalizedBaseUrl, 'DELETE', `/api/history/${id}`, undefined, getToken)
    },

    getTodos(userId?: number, domain?: string): Promise<Todo[]> {
      return apiFetch<Todo[]>(normalizedBaseUrl, 'GET', `/api/todos${buildQuery({ user_id: userId, domain })}`, undefined, getToken)
    },

    createTodo(data: CreateTodoInput): Promise<Todo> {
      return apiFetch<Todo>(normalizedBaseUrl, 'POST', '/api/todos', { ...data, completed: false }, getToken)
    },

    updateTodo(id: number, data: UpdateTodoInput): Promise<Todo> {
      return apiFetch<Todo>(normalizedBaseUrl, 'PUT', `/api/todos/${id}`, data, getToken)
    },

    deleteTodo(id: number): Promise<void> {
      return apiFetch<void>(normalizedBaseUrl, 'DELETE', `/api/todos/${id}`, undefined, getToken)
    },

    async uploadScreenshot(todoId: number, file: Blob, captureId?: string): Promise<Screenshot> {
      const formData = new FormData()
      formData.append('file', file, 'screenshot.png')

      const response = await fetch(`${normalizedBaseUrl}/api/screenshots${buildQuery({ todo_id: todoId, capture_id: captureId })}`, {
        method: 'POST',
        headers: authHeader(getToken),
        body: formData,
      })

      if (!response.ok) {
        const text = await response.text()
        throw apiErrorFromBody(response.status, text, `Upload failed: ${response.status}`)
      }

      return response.json() as Promise<Screenshot>
    },

    getScreenshotUrl(todoId: number): string {
      // Screenshots load via <img src>, which can't set an Authorization
      // header, so the token is passed as a query param instead.
      const token = getToken?.()
      const suffix = token ? `?token=${encodeURIComponent(token)}` : ''
      return `${normalizedBaseUrl}/api/screenshots/${todoId}${suffix}`
    },

    getWallet(userId?: number, website?: string): Promise<WalletEntry[]> {
      return apiFetch<WalletEntry[]>(normalizedBaseUrl, 'GET', `/api/wallet${buildQuery({ user_id: userId, website })}`, undefined, getToken)
    },

    createWallet(data: CreateWalletInput): Promise<WalletEntry> {
      return apiFetch<WalletEntry>(normalizedBaseUrl, 'POST', '/api/wallet', data, getToken)
    },

    async revealWalletPassword(userId: number, id: number): Promise<string> {
      const result = await apiFetch<{ password: string }>(
        normalizedBaseUrl,
        'GET',
        `/api/wallet/reveal${buildQuery({ user_id: userId, id })}`,
        undefined,
        getToken,
      )
      return result.password
    },

    updateWallet(id: number, data: UpdateWalletInput): Promise<WalletEntry> {
      return apiFetch<WalletEntry>(normalizedBaseUrl, 'PUT', `/api/wallet/${id}`, data, getToken)
    },

    getBookmarks(userId?: number, tags?: string, folderPath?: string): Promise<BookmarkResponse[]> {
      return apiFetch<BookmarkResponse[]>(
        normalizedBaseUrl,
        'GET',
        `/api/bookmarks${buildQuery({ user_id: userId, tags, folder_path: folderPath })}`,
        undefined,
        getToken,
      )
    },

    createBookmark(data: CreateBookmarkInput): Promise<BookmarkResponse> {
      return apiFetch<BookmarkResponse>(normalizedBaseUrl, 'POST', '/api/bookmarks', data, getToken)
    },

    deleteBookmark(id: number): Promise<void> {
      return apiFetch<void>(normalizedBaseUrl, 'DELETE', `/api/bookmarks/${id}`, undefined, getToken)
    },

    batchUpsertUsage(data: UsageBatchRequest): Promise<UsageBatchResponse> {
      return apiFetch<UsageBatchResponse>(normalizedBaseUrl, 'POST', '/api/analytics/usage', data, getToken)
    },

    getAnalyticsSummary(params: AnalyticsSummaryParams): Promise<AnalyticsSummary> {
      return apiFetch<AnalyticsSummary>(
        normalizedBaseUrl,
        'GET',
        `/api/analytics/summary${buildQuery(params as unknown as Record<string, string | number | undefined>)}`,
        undefined,
        getToken,
      )
    },
  }
}
