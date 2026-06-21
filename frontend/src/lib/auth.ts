// Client-side storage for the operator API token. The token is entered by the
// user in the UI and persisted in localStorage, then attached as a Bearer
// header to every API request.

const TOKEN_KEY = 'api_token'

export function getToken(): string | null {
  if (typeof localStorage === 'undefined') return null
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  if (typeof localStorage === 'undefined') return
  const trimmed = token.trim()
  if (trimmed) {
    localStorage.setItem(TOKEN_KEY, trimmed)
  } else {
    localStorage.removeItem(TOKEN_KEY)
  }
  window.dispatchEvent(new CustomEvent('api-token-changed'))
}

export function clearToken(): void {
  if (typeof localStorage === 'undefined') return
  localStorage.removeItem(TOKEN_KEY)
  window.dispatchEvent(new CustomEvent('api-token-changed'))
}

export function hasToken(): boolean {
  return Boolean(getToken())
}

/** Authorization header object for raw fetch calls, empty when no token set. */
export function authHeaders(): Record<string, string> {
  const token = getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}
