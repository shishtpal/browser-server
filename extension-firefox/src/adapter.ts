import type { BrowserApi, BrowserTab, ContextMenuClickInfo } from '@browser-server/extension-core'
import type browserType from 'webextension-polyfill'

declare const browser: typeof browserType

type OmniboxSuggestion = { content: string; description: string }

function stripOmniboxMarkup(description: string): string {
  return description.replace(/<\/?(?:match|dim)>/g, '')
}

export class FirefoxAdapter implements BrowserApi {
  storage = {
    local: {
      get: (key: string) => browser.storage.local.get(key) as Promise<Record<string, unknown>>,
      set: (items: Record<string, unknown>) => browser.storage.local.set(items),
      remove: (key: string) => browser.storage.local.remove(key),
    },
    onChanged: {
      addListener: (callback: (changes: Record<string, { newValue?: unknown }>) => void) =>
        browser.storage.onChanged.addListener(callback),
      removeListener: (callback: (changes: Record<string, { newValue?: unknown }>) => void) =>
        browser.storage.onChanged.removeListener(callback),
    },
  }

  tabs = {
    query: (queryInfo: { active?: boolean; currentWindow?: boolean }) => browser.tabs.query(queryInfo),
    update: (updateProperties: { url?: string }) =>
      browser.tabs.update(updateProperties).then(() => undefined),
    create: (createProperties: { url: string; active: boolean }) =>
      browser.tabs.create(createProperties).then(() => undefined),
    captureVisibleTab: (windowId: number, options: { format: string }) =>
      browser.tabs.captureVisibleTab(windowId, options as browserType.ExtensionTypes.ImageDetails),
    onUpdated: {
      addListener: (
        callback: (
          tabId: number,
          changeInfo: { url?: string; status?: string },
          tab: { active?: boolean; id?: number },
        ) => void,
      ) => browser.tabs.onUpdated.addListener(callback),
    },
    onActivated: {
      addListener: (callback: (activeInfo: { tabId: number }) => void) =>
        browser.tabs.onActivated.addListener(callback),
    },
  }

  windows = {
    getCurrent: () => browser.windows.getCurrent(),
    onFocusChanged: {
      addListener: (callback: (windowId: number) => void) =>
        browser.windows.onFocusChanged.addListener(callback),
    },
    WINDOW_ID_NONE: browser.windows.WINDOW_ID_NONE,
  }

  idle = {
    setDetectionInterval: (intervalInSeconds: number) => {
      if (browser.idle) {
        browser.idle.setDetectionInterval(intervalInSeconds)
      }
    },
    onStateChanged: {
      addListener: (callback: (newState: 'active' | 'idle' | 'locked') => void) => {
        if (browser.idle) {
          browser.idle.onStateChanged.addListener(callback)
        }
      },
    },
  }

  alarms = {
    create: (name: string, alarmInfo: { periodInMinutes: number }) => {
      void browser.alarms.create(name, alarmInfo)
    },
    onAlarm: {
      addListener: (callback: (alarm: { name: string }) => void) =>
        browser.alarms.onAlarm.addListener(callback),
    },
  }

  omnibox = {
    setDefaultSuggestion: (suggestion: { description: string }) =>
      browser.omnibox.setDefaultSuggestion(suggestion),
    onInputStarted: {
      addListener: (callback: () => void) => browser.omnibox.onInputStarted.addListener(callback),
    },
    onInputChanged: {
      addListener: (
        callback: (text: string, suggest: (results: OmniboxSuggestion[]) => void) => void,
      ) => {
        browser.omnibox.onInputChanged.addListener((text, suggest) => {
          callback(text, (results) => {
            suggest(
              results.map((result) => ({
                ...result,
                description: stripOmniboxMarkup(result.description),
              })),
            )
          })
        })
      },
    },
    onInputEntered: {
      addListener: (callback: (text: string, disposition: string) => void) =>
        browser.omnibox.onInputEntered.addListener(callback),
    },
  }

  contextMenus = {
    removeAll: () => browser.menus.removeAll(),
    create: (properties: {
      id: string
      parentId?: string
      title: string
      contexts: Array<'page' | 'selection'>
    }) => {
      browser.menus.create(properties)
    },
    onClicked: {
      addListener: (callback: (info: ContextMenuClickInfo, tab?: BrowserTab) => void) =>
        browser.menus.onClicked.addListener((info, tab) => callback(info, tab)),
    },
  }

  commands = {
    onCommand: {
      addListener: (callback: (command: string, tab?: BrowserTab) => void) =>
        browser.commands.onCommand.addListener((command, tab) => callback(command, tab)),
    },
  }

  notifications = {
    create: (options: { type: 'basic'; iconUrl: string; title: string; message: string }) =>
      browser.notifications.create(options),
  }

  runtime = {
    onInstalled: {
      addListener: (callback: () => void) => browser.runtime.onInstalled.addListener(callback),
    },
    onMessage: {
      addListener: (
        callback: (
          message: unknown,
          sender: unknown,
          sendResponse: (response: unknown) => void,
        ) => boolean | void,
      ) =>
        browser.runtime.onMessage.addListener(
          ((message, sender, sendResponse) =>
            callback(message, sender, sendResponse) === true ? true : undefined) as Parameters<
            typeof browser.runtime.onMessage.addListener
          >[0],
        ),
    },
    onSuspend: {
      addListener: (callback: () => void) => browser.runtime.onSuspend.addListener(callback),
    },
    sendMessage: (message: unknown) => browser.runtime.sendMessage(message),
    openOptionsPage: () => {
      void browser.runtime.openOptionsPage()
    },
  }

  declarativeNetRequest = typeof browser.declarativeNetRequest !== 'undefined'
    ? {
        updateSessionRules: (options: {
          removeRuleIds?: number[]
          addRules?: Array<{
            id: number
            priority: number
            condition: Record<string, unknown>
            action: Record<string, unknown>
          }>
        }) => browser.declarativeNetRequest.updateSessionRules(options as Parameters<typeof browser.declarativeNetRequest.updateSessionRules>[0]),
      }
    : undefined
}
