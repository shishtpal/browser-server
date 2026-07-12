import type { BrowserApi } from '@browser-server/extension-core'

export class ChromeAdapter implements BrowserApi {
  storage = {
    local: {
      get: (key: string) => chrome.storage.local.get(key) as Promise<Record<string, unknown>>,
      set: (items: Record<string, unknown>) => chrome.storage.local.set(items),
      remove: (key: string) => chrome.storage.local.remove(key),
    },
    onChanged: {
      addListener: (callback: (changes: Record<string, { newValue?: unknown }>) => void) =>
        chrome.storage.onChanged.addListener(callback),
      removeListener: (callback: (changes: Record<string, { newValue?: unknown }>) => void) =>
        chrome.storage.onChanged.removeListener(callback),
    },
  }

  tabs = {
    query: (queryInfo: chrome.tabs.QueryInfo) => chrome.tabs.query(queryInfo),
    update: (updateProperties: chrome.tabs.UpdateProperties) =>
      chrome.tabs.update(updateProperties).then(() => undefined),
    create: (createProperties: chrome.tabs.CreateProperties) =>
      chrome.tabs.create(createProperties).then(() => undefined),
    captureVisibleTab: (windowId: number, options: { format: string }) =>
      chrome.tabs.captureVisibleTab(windowId, options as chrome.extensionTypes.ImageDetails),
    onUpdated: chrome.tabs.onUpdated,
    onActivated: chrome.tabs.onActivated,
  }

  windows = {
    getCurrent: () => chrome.windows.getCurrent(),
    onFocusChanged: chrome.windows.onFocusChanged,
    WINDOW_ID_NONE: chrome.windows.WINDOW_ID_NONE,
  }

  idle = {
    setDetectionInterval: (intervalInSeconds: number) =>
      chrome.idle.setDetectionInterval(intervalInSeconds),
    onStateChanged: chrome.idle.onStateChanged,
  }

  alarms = {
    create: (name: string, alarmInfo: chrome.alarms.AlarmCreateInfo) =>
      chrome.alarms.create(name, alarmInfo),
    onAlarm: chrome.alarms.onAlarm,
  }

  omnibox = {
    setDefaultSuggestion: (suggestion: chrome.omnibox.DefaultSuggestResult) =>
      chrome.omnibox.setDefaultSuggestion(suggestion),
    onInputStarted: chrome.omnibox.onInputStarted,
    onInputChanged: chrome.omnibox.onInputChanged,
    onInputEntered: chrome.omnibox.onInputEntered,
  }

  runtime = {
    onMessage: chrome.runtime.onMessage,
    onSuspend: chrome.runtime.onSuspend,
    sendMessage: (message: unknown) => chrome.runtime.sendMessage(message),
    openOptionsPage: () => chrome.runtime.openOptionsPage(),
  }
}
