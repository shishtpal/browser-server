/**
 * Browser Server API client.
 *
 * This module assembles domain-specific method groups into a single unified
 * client object. Each domain lives in its own file under `./domains/` for
 * maintainability — but consumers see one flat namespace.
 */

import { type TokenProvider, normalizeBaseUrl } from './internals'
import { createAIMethods } from './domains/ai'
import { createAnalyticsMethods } from './domains/analytics'
import { createBookmarkMethods } from './domains/bookmarks'
import { createHealthMethods } from './domains/health'
import { createHistoryMethods } from './domains/history'
import { createScreenshotMethods } from './domains/screenshots'
import { createTodoMethods } from './domains/todos'
import { createWalletMethods } from './domains/wallet'

// Re-export public types consumers may need.
export type { TokenProvider } from './internals'
export { ApiError } from './internals'

export interface BrowserServerClientOptions {
  /** Called on every request to obtain the bearer token to send. */
  getToken?: TokenProvider
}

export type BrowserServerClient = ReturnType<typeof createBrowserServerClient>

export function createBrowserServerClient(baseUrl: string, options: BrowserServerClientOptions = {}) {
  const normalized = normalizeBaseUrl(baseUrl)
  const { getToken } = options

  return {
    ...createHealthMethods(normalized, getToken),
    ...createTodoMethods(normalized, getToken),
    ...createScreenshotMethods(normalized, getToken),
    ...createWalletMethods(normalized, getToken),
    ...createBookmarkMethods(normalized, getToken),
    ...createHistoryMethods(normalized, getToken),
    ...createAnalyticsMethods(normalized, getToken),
    ...createAIMethods(normalized, getToken),
  }
}
