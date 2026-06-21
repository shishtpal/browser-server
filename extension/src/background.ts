import { createBrowserServerClient } from '@browser-server/shared-client'
import { isTrackableUrl } from './lib/browser'
import { getSettings } from './lib/settings'
import { TimeTracker } from './lib/timeTracker'

const USAGE_FLUSH_ALARM = 'usage-flush'

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
