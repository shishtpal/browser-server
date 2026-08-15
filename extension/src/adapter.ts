import type { BrowserApi, BrowserTab, ContextMenuClickInfo, CookieInfo, CookieSetInput } from '@browser-server/extension-core'

export class ChromeAdapter implements BrowserApi {
  storage = {
    local: {
      get: (key: string | string[]) => chrome.storage.local.get(key) as Promise<Record<string, unknown>>,
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
    get: (tabId: number) => chrome.tabs.get(tabId),
    update: (updateProperties: chrome.tabs.UpdateProperties) =>
      chrome.tabs.update(updateProperties).then(() => undefined),
    updateTab: (tabId: number, updateProperties: { active?: boolean; url?: string }) =>
      chrome.tabs.update(tabId, updateProperties as chrome.tabs.UpdateProperties).then(() => undefined),
    create: (createProperties: chrome.tabs.CreateProperties) => chrome.tabs.create(createProperties),
    remove: (tabId: number) => chrome.tabs.remove(tabId).then(() => undefined),
    sendMessage: (tabId: number, message: unknown) => chrome.tabs.sendMessage(tabId, message),
    captureVisibleTab: (windowId: number, options: { format: string }) =>
      chrome.tabs.captureVisibleTab(windowId, options as chrome.extensionTypes.ImageDetails),
    onUpdated: chrome.tabs.onUpdated,
    onActivated: chrome.tabs.onActivated,
    onRemoved: chrome.tabs.onRemoved,
    focus: (tabId: number, windowId?: number) => {
      const win = windowId !== undefined ? chrome.windows.update(windowId, { focused: true }) : Promise.resolve()
      return Promise.all([chrome.tabs.update(tabId, { active: true }), win]).then(() => undefined)
    },
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

  contextMenus = {
    removeAll: () => new Promise<void>((resolve, reject) => {
      chrome.contextMenus.removeAll(() => {
        const error = chrome.runtime.lastError
        if (error) reject(new Error(error.message))
        else resolve()
      })
    }),
    create: (properties: {
      id: string
      parentId?: string
      title: string
      contexts: Array<'page' | 'selection'>
    }) => {
      chrome.contextMenus.create(properties as chrome.contextMenus.CreateProperties)
    },
    onClicked: {
      addListener: (callback: (info: ContextMenuClickInfo, tab?: BrowserTab) => void) =>
        chrome.contextMenus.onClicked.addListener((info, tab) => callback(info, tab)),
    },
  }

  commands = {
    onCommand: {
      addListener: (callback: (command: string, tab?: BrowserTab) => void) =>
        chrome.commands.onCommand.addListener((command, tab) => callback(command, tab)),
    },
  }

  notifications = {
    create: (options: { type: 'basic'; iconUrl: string; title: string; message: string }) =>
      new Promise<string>((resolve, reject) => {
        chrome.notifications.create(options, (notificationId) => {
          const error = chrome.runtime.lastError
          if (error) reject(new Error(error.message))
          else resolve(notificationId)
        })
      }),
  }

  runtime = {
    onInstalled: chrome.runtime.onInstalled,
    onMessage: chrome.runtime.onMessage,
    onSuspend: chrome.runtime.onSuspend,
    sendMessage: (message: unknown) => chrome.runtime.sendMessage(message),
    openOptionsPage: () => chrome.runtime.openOptionsPage(),
    getURL: (path: string) => chrome.runtime.getURL(path),
  }

  scripting = chrome.scripting
    ? {
        executeScript: (script: { target: { tabId: number }; files?: string[] }) =>
          chrome.scripting.executeScript(script as unknown as Parameters<typeof chrome.scripting.executeScript>[0]),
      }
    : undefined

  debugger = chrome.debugger
    ? {
        attach: (target: { tabId: number }, requiredVersion: string) =>
          chrome.debugger.attach(target, requiredVersion),
        detach: (target: { tabId: number }) => chrome.debugger.detach(target),
        sendCommand: (target: { tabId: number }, method: string, params?: Record<string, unknown>) =>
          chrome.debugger.sendCommand(target, method, params as { [key: string]: unknown }),
      }
    : undefined

  cookies = chrome.cookies
    ? {
        getAll: (filter: { domain?: string; name?: string; url?: string }) =>
          chrome.cookies.getAll(filter) as Promise<CookieInfo[]>,
        set: (cookie: CookieSetInput) => chrome.cookies.set(cookie).then(() => undefined),
      }
    : undefined

  declarativeNetRequest = chrome.declarativeNetRequest
    ? {
        updateSessionRules: (options: {
          removeRuleIds?: number[]
          addRules?: Array<{
            id: number
            priority: number
            condition: Record<string, unknown>
            action: Record<string, unknown>
          }>
        }) =>
          chrome.declarativeNetRequest.updateSessionRules(
            options as chrome.declarativeNetRequest.UpdateRuleOptions,
          ),
      }
    : undefined
}
