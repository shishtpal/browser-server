import { createBrowserServerClient } from '@browser-server/shared-client'
import type { OmniboxSearchResult, WalletEntry } from '@browser-server/shared-types'
import { getBrowserApi } from './browserApi'
import { isTrackableUrl } from './lib/browser'
import { getSettings } from './lib/settings'
import { TimeTracker } from './lib/timeTracker'

const USAGE_FLUSH_ALARM = 'usage-flush'
const OMNIBOX_RESULT_LIMIT = 6

type OmniboxSuggestion = { content: string; description: string }

let lastRecordedUrl: string | null = null
const tracker = new TimeTracker()

function extractHostname(url: string): string | null {
  try {
    const u = new URL(url)
    return u.hostname || null
  } catch {
    return null
  }
}

function normalizeHostname(value: string): string | null {
  try {
    const url = new URL(value.includes('://') ? value : `https://${value}`)
    return url.hostname.toLowerCase().replace(/^www\./, '') || null
  } catch {
    return null
  }
}

function walletEntryMatchesHostname(entry: WalletEntry, hostname: string): boolean {
  const savedHostname = normalizeHostname(entry.website)
  const currentHostname = normalizeHostname(hostname)
  if (!savedHostname || !currentHostname) return false
  return currentHostname === savedHostname
}

async function getLoginProviderAccounts(hostname: string) {
  const settings = await getSettings()
  const userId = Number.parseInt(settings.userId, 10)
  if (!settings.apiToken || Number.isNaN(userId)) return []

  const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })
  const entries = await client.getWallet(userId)
  return entries.filter((entry) => walletEntryMatchesHostname(entry, hostname)).map((entry) => ({
    loginProvider: entry.login_provider || 'Password',
    username: entry.username,
  }))
}

async function postVisit(url: string, title: string | undefined): Promise<void> {
  if (!isTrackableUrl(url)) {
    return
  }

  const settings = await getSettings()
  const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })

  try {
    await client.createHistory({
      user_id: Number.parseInt(settings.userId, 10),
      url,
      title: title || url,
      duration: 0,
    })
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    console.debug('History sync failed (server offline?)', message)
  }
}

function escapeOmniboxText(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

function getDisplayUrl(url: string): string {
  try {
    const parsed = new URL(url)
    return `${parsed.hostname}${parsed.pathname === '/' ? '' : parsed.pathname}`
  } catch {
    return url
  }
}

function isLikelyUrl(value: string): boolean {
  const trimmed = value.trim()
  return /^https?:\/\//i.test(trimmed) || /^[\w-]+(\.[\w-]+)+([/:?#].*)?$/i.test(trimmed)
}

function toNavigableUrl(value: string): string {
  const trimmed = value.trim()
  if (/^https?:\/\//i.test(trimmed)) {
    return trimmed
  }
  if (isLikelyUrl(trimmed)) {
    return `https://${trimmed}`
  }
  return `https://www.google.com/search?q=${encodeURIComponent(trimmed)}`
}

function formatOmniboxSuggestion(result: OmniboxSearchResult): OmniboxSuggestion {
  const title = result.title || result.url
  const displayUrl = getDisplayUrl(result.url)

  if (result.source === 'history') {
    const count = result.visit_count ?? 0
    const countLabel = `${count} visit${count === 1 ? '' : 's'}`
    return {
      content: result.url,
      description: `<match>[History]</match> ${escapeOmniboxText(title)} <dim>- ${escapeOmniboxText(displayUrl)} - ${countLabel}</dim>`,
    }
  }

  const details = [
    result.folder_path,
    result.tags && result.tags.length > 0 ? result.tags.join(', ') : undefined,
    result.description,
  ].filter((detail): detail is string => Boolean(detail))

  const detailText = details.length > 0 ? ` - ${details.join(' - ')}` : ''
  return {
    content: result.url,
    description: `<match>[Bookmark]</match> ${escapeOmniboxText(title)} <dim>- ${escapeOmniboxText(displayUrl)}${escapeOmniboxText(detailText)}</dim>`,
  }
}

async function suggestOmniboxResults(
  text: string,
  suggest: (suggestResults: OmniboxSuggestion[]) => void,
): Promise<void> {
  const query = text.trim()
  if (!query) {
    suggest([])
    return
  }

  const settings = await getSettings()
  const userId = Number.parseInt(settings.userId, 10)
  if (!settings.apiToken || Number.isNaN(userId)) {
    suggest([])
    return
  }

  const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })

  try {
    const results = await client.searchOmnibox({ user_id: userId, q: query, limit: OMNIBOX_RESULT_LIMIT })
    suggest(results.map(formatOmniboxSuggestion))
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    console.debug('Omnibox search failed (server offline?)', message)
    suggest([])
  }
}

async function getActiveTab() {
  const [tab] = await getBrowserApi().tabs.query({ active: true, currentWindow: true })
  return tab ?? null
}

async function isCurrentWindowFocused(): Promise<boolean> {
  const window = await getBrowserApi().windows.getCurrent()
  return Boolean(window?.focused)
}

async function syncActiveTab(): Promise<void> {
  const focused = await isCurrentWindowFocused()
  if (!focused) {
    tracker.stopTracking()
    return
  }

  const tab = await getActiveTab()
  if (!tab?.url || !isTrackableUrl(tab.url)) {
    tracker.stopTracking()
    return
  }

  if (tab.url !== lastRecordedUrl) {
    lastRecordedUrl = tab.url
    void postVisit(tab.url, tab.title)
  }

  tracker.startTracking(extractHostname(tab.url))
}

export function initBackground(): void {
  const api = getBrowserApi()

  api.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    if (!tab.active || tab.id !== tabId) {
      return
    }

    if (changeInfo.url || changeInfo.status === 'complete') {
      void syncActiveTab()
    }
  })

  api.tabs.onActivated.addListener((activeInfo) => {
    void activeInfo
    void syncActiveTab()
  })

  api.windows.onFocusChanged.addListener((windowId) => {
    if (windowId === api.windows.WINDOW_ID_NONE) {
      tracker.stopTracking()
      return
    }

    void syncActiveTab()
  })

  api.idle.onStateChanged.addListener((newState) => {
    tracker.handleIdleState(newState)
    if (newState === 'active') {
      void syncActiveTab()
    }
  })

  api.runtime.onSuspend.addListener(() => {
    void tracker.flush()
  })

  api.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name === USAGE_FLUSH_ALARM) {
      void tracker.flush()
    }
  })

  api.omnibox.onInputStarted.addListener(() => {
    api.omnibox.setDefaultSuggestion({
      description: 'Search Browser Server bookmarks and history',
    })
  })

  api.omnibox.onInputChanged.addListener((text, suggest) => {
    void suggestOmniboxResults(text, suggest)
  })

  api.omnibox.onInputEntered.addListener((text, disposition) => {
    const targetUrl = toNavigableUrl(text)

    switch (disposition) {
      case 'newForegroundTab':
        void api.tabs.create({ url: targetUrl, active: true })
        break
      case 'newBackgroundTab':
        void api.tabs.create({ url: targetUrl, active: false })
        break
      default:
        void api.tabs.update({ url: targetUrl })
        break
    }
  })

  api.runtime.onMessage.addListener((message, _sender, sendResponse) => {
    if (typeof message !== 'object' || message === null || !('type' in message)) {
      return false
    }

    if (message.type === 'captureScreenshot') {
      void api.tabs.query({ active: true, currentWindow: true }).then(async (tabs) => {
        const windowId = tabs[0]?.windowId
        if (windowId === undefined) {
          sendResponse({ dataUrl: null })
          return
        }

        const dataUrl = await api.tabs.captureVisibleTab(windowId, { format: 'png' })
        sendResponse({ dataUrl })
      })
      return true
    }

    if (message.type === 'getLoginProviders' && 'hostname' in message && typeof message.hostname === 'string') {
      void getLoginProviderAccounts(message.hostname)
        .then((accounts) => sendResponse({ accounts }))
        .catch((error) => {
          const detail = error instanceof Error ? error.message : String(error)
          console.debug('Wallet provider lookup failed (server offline?)', detail)
          sendResponse({ accounts: [] })
        })
      return true
    }

    return false
  })

  api.idle.setDetectionInterval(15)
  api.alarms.create(USAGE_FLUSH_ALARM, { periodInMinutes: 0.5 })

  void tracker.restore().then(() => syncActiveTab())
}
