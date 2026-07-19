/**
 * Shared fetch utilities used by all domain modules.
 * Not part of the public API — consumed only within this package.
 */

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

/** Resolves the current API token (or null/undefined when none is set). */
export type TokenProvider = () => string | null | undefined

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
export function apiErrorFromBody(status: number, body: string, fallback: string): ApiError {
  if (body) {
    try {
      const parsed = JSON.parse(body) as { error?: string | { message?: string }; fields?: Record<string, string> }
      if (parsed && typeof parsed.error === 'string') {
        return new ApiError(status, parsed.error, parsed.fields)
      }
      if (parsed && typeof parsed.error === 'object' && typeof parsed.error.message === 'string') {
        return new ApiError(status, parsed.error.message, parsed.fields)
      }
    } catch {
      // Not JSON; use the raw text below.
    }
    return new ApiError(status, body)
  }
  return new ApiError(status, fallback)
}

export function normalizeBaseUrl(baseUrl: string): string {
  return baseUrl.replace(/\/+$/, '')
}

export function authHeader(getToken?: TokenProvider): Record<string, string> {
  const token = getToken?.()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export function buildQuery(params: Record<string, string | number | undefined>): string {
  const searchParams = new URLSearchParams()

  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') {
      searchParams.set(key, String(value))
    }
  }

  const query = searchParams.toString()
  return query ? `?${query}` : ''
}

export async function apiFetch<T>(
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
