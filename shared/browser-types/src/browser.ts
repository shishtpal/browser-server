/**
 * Browser automation types shared between the extension, the server API, and
 * the AI tools (extension command channel, Path 1).
 */

/** One browser profile (one extension instance). */
export interface BrowserInstance {
  instance_id: string
  user_id: number
  browser: string
  version: string
  label: string
  online: boolean
  last_seen_at: string
  created_at: string
}

/** One tab within a browser profile, addressed by its stable tab_uuid. */
export interface BrowserTabInfo {
  tab_uuid: string
  instance_id: string
  tab_id: number
  window_id: number
  url: string
  title: string
  active: boolean
  /** ISO timestamp of last activation, or null if the tab has never been active. */
  last_active_at: string | null
  updated_at: string
}

/** Level-1 browser profile selection. */
export interface BrowserTargetBrowser {
  instance_id?: string
  label?: string
  first_online?: boolean
}

/** Level-2 tab selection. */
export interface BrowserTargetTab {
  uuid?: string
  url?: string
  title?: string
  role?: 'active' | 'new'
}

/** The targeting model every browser command carries. */
export interface BrowserTarget {
  browser?: BrowserTargetBrowser
  tab?: BrowserTargetTab
  session_id?: string
}

/** Page snapshot returned after an action. */
export interface BrowserPageInfo {
  url: string
  title: string
}

/** Structured result payload (action-specific). */
export interface BrowserCommandResult {
  page?: BrowserPageInfo
  scrape?: unknown
  data?: string
  path?: string
  /** How the result was produced when it differs from the requested mode, e.g. "cdp-fallback" for eval. */
  via?: string
}

/** A command travelling from the server bus to the extension. */
export interface BrowserCommand {
  command_id: string
  instance_id: string
  tab_uuid: string
  session_id: string
  action: string
  params?: Record<string, unknown>
  status: 'queued' | 'sent' | 'succeeded' | 'failed' | 'timed_out'
  timeout_ms?: number
  created_at: string
  sent_at?: string
  finished_at?: string
  result?: BrowserCommandResult
  error?: string
}

/** The extension's acknowledgement / final result for one command. */
export interface BrowserResultRequest {
  command_id: string
  status: 'processing' | 'succeeded' | 'failed'
  result?: BrowserCommandResult
  error?: string
}

/** Request body for /api/browser/register and /api/browser/tabs. */
export interface BrowserRegisterInput {
  instance_id: string
  user_id: number
  browser: string
  version: string
  label?: string
}

export interface BrowserTabsInput {
  instance_id: string
  patch?: boolean
  tabs: BrowserTabInfo[]
  removed?: string[]
}

/** Request body for /api/browser/cmd. */
export interface BrowserCreateCommandInput {
  target: BrowserTarget
  action: string
  params?: Record<string, unknown>
  timeout_ms?: number
}
