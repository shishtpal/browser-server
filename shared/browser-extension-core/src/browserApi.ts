/**
 * BrowserApi — the central abstraction for all browser-specific APIs used by
 * the shared extension core.  All shared logic calls through this interface
 * instead of referencing `chrome.*` or `browser.*` directly.
 *
 * Concrete implementations are provided by the browser-specific wrapper
 * packages:
 *   - ChromeAdapter  (extension/src/adapter.ts)   → chrome.*
 *   - FirefoxAdapter (extension-firefox/src/adapter.ts) → browser.*
 */
export interface BrowserTab {
  id?: number
  url?: string
  title?: string
  windowId?: number
  active?: boolean
}

export interface ContextMenuClickInfo {
  menuItemId: string | number
  pageUrl?: string
  selectionText?: string
}

export interface BrowserApi {
  storage: {
    local: {
      get(key: string): Promise<Record<string, unknown>>
      set(items: Record<string, unknown>): Promise<void>
      remove(key: string): Promise<void>
    }
    onChanged: {
      addListener(callback: (changes: Record<string, { newValue?: unknown }>) => void): void
      removeListener(callback: (changes: Record<string, { newValue?: unknown }>) => void): void
    }
  }

  tabs: {
    query(queryInfo: { active?: boolean; currentWindow?: boolean }): Promise<BrowserTab[]>
    update(updateProperties: { url?: string }): Promise<void>
    create(createProperties: { url: string; active: boolean }): Promise<void>
    captureVisibleTab(windowId: number, options: { format: string }): Promise<string>
    onUpdated: {
      addListener(
        callback: (
          tabId: number,
          changeInfo: { url?: string; status?: string },
          tab: { active?: boolean; id?: number },
        ) => void,
      ): void
    }
    onActivated: {
      addListener(callback: (activeInfo: { tabId: number }) => void): void
    }
  }

  windows: {
    getCurrent(): Promise<{ focused?: boolean } | undefined>
    onFocusChanged: {
      addListener(callback: (windowId: number) => void): void
    }
    WINDOW_ID_NONE: number
  }

  idle: {
    setDetectionInterval(intervalInSeconds: number): void
    onStateChanged: {
      addListener(callback: (newState: 'active' | 'idle' | 'locked') => void): void
    }
  }

  alarms: {
    create(name: string, alarmInfo: { periodInMinutes: number }): void
    onAlarm: {
      addListener(callback: (alarm: { name: string }) => void): void
    }
  }

  omnibox: {
    setDefaultSuggestion(suggestion: { description: string }): void
    onInputStarted: {
      addListener(callback: () => void): void
    }
    onInputChanged: {
      addListener(
        callback: (
          text: string,
          suggest: (results: Array<{ content: string; description: string }>) => void,
        ) => void,
      ): void
    }
    onInputEntered: {
      addListener(callback: (text: string, disposition: string) => void): void
    }
  }

  contextMenus: {
    removeAll(): Promise<void>
    create(properties: {
      id: string
      parentId?: string
      title: string
      contexts: Array<'page' | 'selection'>
    }): void
    onClicked: {
      addListener(callback: (info: ContextMenuClickInfo, tab?: BrowserTab) => void): void
    }
  }

  commands: {
    onCommand: {
      addListener(callback: (command: string, tab?: BrowserTab) => void): void
    }
  }

  notifications: {
    create(options: {
      type: 'basic'
      iconUrl: string
      title: string
      message: string
    }): Promise<string>
  }

  runtime: {
    onInstalled: {
      addListener(callback: () => void): void
    }
    onMessage: {
      addListener(
        callback: (
          message: unknown,
          sender: unknown,
          sendResponse: (response: unknown) => void,
        ) => boolean | void,
      ): void
    }
    onSuspend: {
      addListener(callback: () => void): void
    }
    sendMessage(message: unknown): Promise<unknown>
    openOptionsPage(): void
  }

  declarativeNetRequest?: {
    updateSessionRules(options: {
      removeRuleIds?: number[]
      addRules?: Array<{
        id: number
        priority: number
        condition: Record<string, unknown>
        action: Record<string, unknown>
      }>
    }): Promise<void>
  }
}

// ---------------------------------------------------------------------------
// Module-level singleton registry
// ---------------------------------------------------------------------------

let _browserApi: BrowserApi | null = null

/**
 * Register the active `BrowserApi` implementation.
 *
 * Each browser wrapper's `background.ts` entry point MUST call this as its
 * very first executable statement — before any shared module can invoke
 * `getBrowserApi()`.
 */
export function setBrowserApi(impl: BrowserApi): void {
  _browserApi = impl
}

/**
 * Retrieve the registered `BrowserApi` implementation.
 *
 * @throws {Error} If called before `setBrowserApi` has been invoked.
 */
export function getBrowserApi(): BrowserApi {
  if (_browserApi === null) {
    throw new Error('BrowserApi not initialized')
  }
  return _browserApi
}

/**
 * Reset the registry to its initial (uninitialized) state.
 *
 * **For test use only.** Call this in `beforeEach` / `afterEach` to isolate
 * test cases that exercise the registry (see task 2.3).
 *
 * @internal
 */
export function _resetBrowserApi(): void {
  _browserApi = null
}
