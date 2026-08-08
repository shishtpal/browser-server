import { createBrowserServerClient } from '@browser-server/shared-client'
import { authHeaders, getToken } from '../auth'

function resolveApiBase(): string {
  if (typeof window === 'undefined') return 'http://localhost:9191'

  const { protocol, host } = window.location
  return `${protocol}//${host}`
}

/** Base URL for the Browser Server REST API. */
export const API_BASE = resolveApiBase()

/** Shared, token-aware API client used by all domain modules. */
export const client = createBrowserServerClient(API_BASE, { getToken })

// Re-exported so domain modules can `import { authHeaders } from './client'`.
export { authHeaders }
