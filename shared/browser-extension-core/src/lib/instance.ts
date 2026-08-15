import { ApiError, createBrowserServerClient } from '@browser-server/shared-client'
import { getBrowserApi } from '../browserApi'
import { getSettings } from './settings'

const INSTANCE_ID_KEY = 'bs_instance_id'

/**
 * Detects the browser family from the user agent so the server can label the
 * instance. Returns a stable family string (chrome/firefox/edge/other).
 */
export function detectBrowserFamily(): string {
  const ua = navigator.userAgent || ''
  if (/edg\//i.test(ua)) return 'edge'
  if (/firefox|fxios/i.test(ua)) return 'firefox'
  if (/chrome|crios/i.test(ua)) return 'chrome'
  if (/safari/i.test(ua)) return 'safari'
  return 'other'
}

function generateInstanceId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  const random = Math.random().toString(36).slice(2)
  const time = Date.now().toString(36)
  return `inst_${random}${time}`
}

/**
 * Returns this extension instance's stable id, creating and persisting one on
 * first use. The id never changes, so the server can correlate heartbeats,
 * tab snapshots, and command results across browser restarts.
 */
export async function getOrCreateInstanceId(): Promise<string> {
  const stored = await getBrowserApi().storage.local.get(INSTANCE_ID_KEY)
  const existing = stored[INSTANCE_ID_KEY] as string | undefined
  if (existing) {
    return existing
  }
  const fresh = generateInstanceId()
  await getBrowserApi().storage.local.set({ [INSTANCE_ID_KEY]: fresh })
  return fresh
}

/**
 * Registers this instance with the server (idempotent) and marks it online.
 * Safe to call on every background start.
 */
export async function registerInstance(): Promise<void> {
  const settings = await getSettings()
  const userId = Number.parseInt(settings.userId, 10)
  if (!settings.apiToken || Number.isNaN(userId)) {
    return
  }
  const instanceId = await getOrCreateInstanceId()
  const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })
  try {
    await client.browserRegister({
      instance_id: instanceId,
      user_id: userId,
      browser: detectBrowserFamily(),
      version: chromeVersion(),
      label: settings.instanceLabel || undefined,
    })
  } catch {
    // Server may be offline; the next heartbeat/tab sync will retry.
  }
}

function chromeVersion(): string {
  const match = /(?:Chrome|Firefox)\/(\d+(?:\.\d+)*)/.exec(navigator.userAgent || '')
  return match ? match[1] : ''
}

/**
 * Refreshes the server's online TTL for this instance. Called from the same
 * 30-second alarm that flushes usage, so the extension adds no new timers.
 */
export async function heartbeat(): Promise<void> {
  const settings = await getSettings()
  if (!settings.apiToken) {
    return
  }
  const instanceId = await getOrCreateInstanceId()
  const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })
  try {
    await client.browserHeartbeat(instanceId)
  } catch (error) {
    // A 404 means the server restarted and lost this instance's registration.
    // Re-register (idempotent) so the instance comes back online without
    // waiting for a tab event. Network failures (server offline) don't reach
    // here as ApiError, so this never spams a down server.
    if (error instanceof ApiError && error.status === 404) {
      await registerInstance()
    }
  }
}
