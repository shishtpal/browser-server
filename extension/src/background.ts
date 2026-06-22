import type { OmniboxSearchResult } from '@browser-server/shared-types'
import { createBrowserServerClient } from '@browser-server/shared-client'
import { isTrackableUrl } from './lib/browser'
import { getSettings } from './lib/settings'
import { TimeTracker } from './lib/timeTracker'

const USAGE_FLUSH_ALARM = 'usage-flush'
const OMNIBOX_RESULT_LIMIT = 6

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

function formatOmniboxSuggestion(result: OmniboxSearchResult): chrome.omnibox.SuggestResult {
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

async function suggestOmniboxResults(text: string, suggest: (suggestResults: chrome.omnibox.SuggestResult[]) => void): Promise<void> {
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

function getActiveTab(): Promise<chrome.tabs.Tab | null> {
  return new Promise((resolve) => {
    chrome.tabs.query({ active: true, currentWindow: true }, ([tab]) => {
      resolve(tab ?? null)
    })
  })
}

function isCurrentWindowFocused(): Promise<boolean> {
  return new Promise((resolve) => {
    chrome.windows.getCurrent((window) => {
      resolve(Boolean(window?.focused))
    })
  })
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

chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (!tab.active || tab.id !== tabId) {
    return
  }

  if (changeInfo.url || changeInfo.status === 'complete') {
    void syncActiveTab()
  }
})

chrome.tabs.onActivated.addListener((activeInfo) => {
  void activeInfo
  void syncActiveTab()
})

chrome.windows.onFocusChanged.addListener((windowId) => {
  if (windowId === chrome.windows.WINDOW_ID_NONE) {
    tracker.stopTracking()
    return
  }

  void syncActiveTab()
})

chrome.idle.onStateChanged.addListener((newState) => {
  tracker.handleIdleState(newState)
  if (newState === 'active') {
    void syncActiveTab()
  }
})

chrome.runtime.onSuspend.addListener(() => {
  void tracker.flush()
})

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === USAGE_FLUSH_ALARM) {
    void tracker.flush()
  }
})

chrome.omnibox.onInputStarted.addListener(() => {
  chrome.omnibox.setDefaultSuggestion({
    description: 'Search Browser Server bookmarks and history',
  })
})

chrome.omnibox.onInputChanged.addListener((text, suggest) => {
  void suggestOmniboxResults(text, suggest)
})

chrome.omnibox.onInputEntered.addListener((text, disposition) => {
  const targetUrl = toNavigableUrl(text)

  switch (disposition) {
    case 'newForegroundTab':
      void chrome.tabs.create({ url: targetUrl, active: true })
      break
    case 'newBackgroundTab':
      void chrome.tabs.create({ url: targetUrl, active: false })
      break
    default:
      void chrome.tabs.update({ url: targetUrl })
      break
  }
})

chrome.runtime.onMessage.addListener((message: { type?: string }, _sender, sendResponse) => {
  if (message.type !== 'captureScreenshot') {
    return false
  }

  chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
    const windowId = tabs[0]?.windowId
    if (windowId === undefined) {
      sendResponse({ dataUrl: null })
      return
    }

    chrome.tabs.captureVisibleTab(windowId, { format: 'png' }, (dataUrl) => {
      sendResponse({ dataUrl })
    })
  })

  return true
})

chrome.idle.setDetectionInterval(15)
chrome.alarms.create(USAGE_FLUSH_ALARM, { periodInMinutes: 0.5 })

void tracker.restore().then(() => syncActiveTab())
