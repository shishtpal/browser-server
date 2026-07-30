import { createBrowserServerClient } from '@browser-server/shared-client'
import { authHeaders, getToken } from '../auth'

/** Base URL for the Browser Server REST API. */
export const API_BASE = 'http://localhost:9191'

/** Shared, token-aware API client used by all domain modules. */
export const client = createBrowserServerClient(API_BASE, { getToken })

// Re-exported so domain modules can `import { authHeaders } from './client'`.
export { authHeaders }
