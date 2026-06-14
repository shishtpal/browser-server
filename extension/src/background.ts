import { createBrowserServerClient } from '@browser-server/shared-client'
import { isTrackableUrl } from './lib/browser'
import { getSettings } from './lib/settings'
import { TimeTracker } from './lib/timeTracker'

let lastUrl: string | null = null
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
  const client = createBrowserServerClient(settings.apiBase)

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

chrome.tabs.onUpdated.addListener((_tabId, changeInfo, tab) => {
  if (changeInfo.url && changeInfo.url !== lastUrl) {
    lastUrl = changeInfo.url
    void postVisit(changeInfo.url, tab.title)
    tracker.startTracking(extractHostname(changeInfo.url))
    return
  }

  if (changeInfo.status === 'complete' && tab.url && tab.url !== lastUrl) {
    lastUrl = tab.url
    void postVisit(tab.url, tab.title)
    tracker.startTracking(extractHostname(tab.url))
  }
})

chrome.tabs.onActivated.addListener((activeInfo) => {
  chrome.tabs.get(activeInfo.tabId, (tab) => {
    if (tab.url && tab.url !== lastUrl) {
      lastUrl = tab.url
      void postVisit(tab.url, tab.title)
      tracker.startTracking(extractHostname(tab.url))
    }
  })
})

chrome.idle.onStateChanged.addListener((newState) => {
  tracker.handleIdleState(newState)
})

chrome.runtime.onSuspend.addListener(() => {
  tracker.stopPeriodicFlush()
  void tracker.flush()
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

void tracker.restore()
tracker.startPeriodicFlush()
