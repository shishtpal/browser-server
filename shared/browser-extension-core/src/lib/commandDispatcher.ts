import { createBrowserServerClient } from '@browser-server/shared-client'
import type { BrowserCommand, BrowserCommandResult } from '@browser-server/shared-types'
import { getBrowserApi } from '../browserApi'
import { isTrackableUrl } from './browser'
import { DEFAULT_EVAL_TIMEOUT_MS, evalInTabViaCdp, printToPdfViaCdp } from './cdpEval'
import { getSettings } from './settings'
import type { TabRegistry } from './tabRegistry'
import type { AutomationResult } from './automationExecutor'

const NAVIGATION_READY_TIMEOUT_MS = 30_000

interface Report {
  (status: 'succeeded' | 'failed', result?: BrowserCommandResult, error?: string): Promise<void>
}


/**
 * CommandDispatcher turns a queued command into a real browser action:
 *
 *   1. arm-switch check (allowAiControl)
 *   2. acknowledge processing to the server
 *   3. resolve tab_uuid → native tab id
 *   4. dispatch to the content script (or handle screenshot/new_tab/cookies/
 *      pdf/eval directly in the background, since those need extension APIs)
 *   5. report the final result to the server bus
 */
export class CommandDispatcher {
  private inFlight = new Set<string>()

  constructor(private readonly tabs: TabRegistry) {}

  async dispatch(command: BrowserCommand): Promise<void> {
    if (this.inFlight.has(command.command_id)) {
      return
    }
    this.inFlight.add(command.command_id)
    try {
      await this.handle(command)
    } finally {
      this.inFlight.delete(command.command_id)
    }
  }

  private async handle(command: BrowserCommand): Promise<void> {
    const settings = await getSettings()
    if (!settings.apiToken) {
      return
    }
    const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })
    const report: Report = async (status, result, error) => {
      await client
        .browserPostResult({ command_id: command.command_id, status, result, error })
        .catch(() => undefined)
    }

    // Master arm switch: the user must explicitly enable AI control.
    if (!settings.allowAiControl) {
      await report('failed', undefined, 'ai_control_disabled: enable "Allow AI control" in the extension options')
      return
    }

    // Acknowledge so the server marks the command sent (and the CLI poll sees
    // progress rather than hanging).
    await client
      .browserPostResult({ command_id: command.command_id, status: 'processing' })
      .catch(() => undefined)

    try {
      switch (command.action) {
        case 'screenshot': {
          const dataUrl = await this.capture(command)
          await report('succeeded', { data: dataUrl })
          return
        }
        case 'close_tab': {
          await this.closeTab(command, report)
          return
        }
        case 'new_tab': {
          await this.createTab(command, report)
          return
        }
        case 'bring_to_front': {
          await this.bringToFront(command, report)
          return
        }
        case 'get_cookies': {
          await this.getCookies(command, report)
          return
        }
        case 'set_cookie': {
          await this.setCookie(command, report)
          return
        }
        case 'pdf': {
          await this.pdf(command, report)
          return
        }
        case 'eval': {
          await this.runEval(command, report)
          return
        }
        default: {
          const result = await this.runInTab(command)
          if (result.navigating) {
            await this.waitForNavigation(command)
            const page = await this.currentPage(command.tab_uuid)
            await report('succeeded', { page })
            return
          }
          await report('succeeded', result.result)
          return
        }
      }
    } catch (error) {
      const detail = error instanceof Error ? error.message : String(error)
      await report('failed', undefined, detail)
    }
  }

  private async capture(command: BrowserCommand): Promise<string> {
    const tabId = await this.requireTabId(command.tab_uuid)
    const tab = await getBrowserApi().tabs.get(tabId)
    if (tab.windowId === undefined) {
      throw new Error('tab has no window')
    }
    // tabs.captureVisibleTab captures the ACTIVE tab of the window, not the tab
    // this command targeted. Bring the requested tab to the front of its window
    // first so the capture matches the target (a background tab has no visible
    // area to snapshot).
    if (!tab.active) {
      await getBrowserApi().tabs.updateTab(tabId, { active: true })
    }
    return getBrowserApi().tabs.captureVisibleTab(tab.windowId, { format: 'png' })
  }

  private async closeTab(command: BrowserCommand, report: Report): Promise<void> {
    const tabId = await this.resolveTabId(command.tab_uuid)
    if (tabId === undefined) {
      await report('failed', undefined, 'tab_not_found')
      return
    }
    await getBrowserApi().tabs.remove(tabId)
    await report('succeeded', {})
  }

  private async createTab(command: BrowserCommand, report: Report): Promise<void> {
    const params = (command.params ?? {}) as { url?: string }
    const tab = await getBrowserApi().tabs.create({
      url: params.url || 'about:blank',
      active: true,
    })
    if (tab.id === undefined) {
      await report('failed', undefined, 'tab creation returned no id')
      return
    }
    const tabUuid = this.tabs.assign(tab.id)
    await this.tabs.refreshSnapshot()
    await this.waitForTabReady(tab.id, params.url || undefined)
    const page = await this.currentPage(tabUuid)
    await report('succeeded', { page })
  }

  private async bringToFront(command: BrowserCommand, report: Report): Promise<void> {
    const tabId = await this.resolveTabId(command.tab_uuid)
    if (tabId === undefined) {
      await report('failed', undefined, 'tab_not_found')
      return
    }
    const tab = await getBrowserApi().tabs.get(tabId)
    await getBrowserApi().tabs.focus(tabId, tab.windowId)
    await report('succeeded', {})
  }

  private async getCookies(command: BrowserCommand, report: Report): Promise<void> {
    const api = getBrowserApi()
    if (!api.cookies) {
      await report('failed', undefined, 'cookies_api_unavailable: the extension has no cookies permission')
      return
    }
    const tabId = await this.requireTabId(command.tab_uuid)
    const tab = await api.tabs.get(tabId)
    const params = (command.params ?? {}) as { domain?: string; name?: string; url?: string }
    const origin = tab.url ? new URL(tab.url).origin : ''
    const filter: Record<string, unknown> = {}
    if (params.url) {
      filter.url = params.url
    } else if (params.domain) {
      filter.domain = params.domain
    } else if (origin) {
      filter.url = origin
    }
    if (params.name) {
      filter.name = params.name
    }
    const cookies = await api.cookies.getAll(filter as never)
    await report('succeeded', { scrape: cookies })
  }

  private async setCookie(command: BrowserCommand, report: Report): Promise<void> {
    const api = getBrowserApi()
    if (!api.cookies) {
      await report('failed', undefined, 'cookies_api_unavailable: the extension has no cookies permission')
      return
    }
    const tabId = await this.requireTabId(command.tab_uuid)
    const tab = await api.tabs.get(tabId)
    const params = (command.params ?? {}) as {
      name?: string
      value?: string
      url?: string
      path?: string
      domain?: string
      secure?: boolean
      http_only?: boolean
    }
    if (!params.name || params.value === undefined) {
      await report('failed', undefined, 'set_cookie requires name and value')
      return
    }
    const url = params.url ?? (tab.url ? new URL(tab.url).origin : '')
    if (!url) {
      await report('failed', undefined, 'set_cookie requires a target url (tab has none)')
      return
    }
    await api.cookies.set({
      url,
      name: params.name,
      value: params.value,
      path: params.path ?? '/',
      domain: params.domain,
      secure: params.secure,
      httpOnly: params.http_only,
    } as never)
    await report('succeeded', {})
  }

  private async pdf(command: BrowserCommand, report: Report): Promise<void> {
    const tabId = await this.requireTabId(command.tab_uuid)
    const tab = await getBrowserApi().tabs.get(tabId)
    if (tab.url && !isTrackableUrl(tab.url)) {
      await report('failed', undefined, 'unsupported_page: cannot export browser-internal pages to PDF')
      return
    }
    const timeoutMs =
      typeof command.timeout_ms === 'number' && command.timeout_ms > 0 ? command.timeout_ms : DEFAULT_EVAL_TIMEOUT_MS
    const result = await printToPdfViaCdp(getBrowserApi(), tabId, { printBackground: true }, timeoutMs)
    if (result.error) {
      await report('failed', undefined, result.error)
      return
    }
    await report('succeeded', { data: 'data:application/pdf;base64,' + (result.data ?? ''), via: 'cdp' })
  }

  /**
   * Runs a JavaScript expression in the page's main world via CDP
   * (chrome.debugger / browser.debugger → Runtime.evaluate). CDP evaluation is
   * exempt from both the extension's and the page's Content Security Policy,
   * so it works on every page regardless of `unsafe-eval` — the same mechanism
   * Playwright/Puppeteer use.
   */
  private async runEval(command: BrowserCommand, report: Report): Promise<void> {
    const api = getBrowserApi()
    const tabId = await this.requireTabId(command.tab_uuid)
    const tab = await api.tabs.get(tabId)
    if (tab.url && !isTrackableUrl(tab.url)) {
      await report('failed', undefined, 'unsupported_page: cannot evaluate on browser-internal pages')
      return
    }
    const params = (command.params ?? {}) as { expression?: string; mode?: string }
    const expression = typeof params.expression === 'string' ? params.expression : ''
    if (!expression.trim()) {
      await report('failed', undefined, 'eval requires an expression')
      return
    }
    const timeoutMs =
      typeof command.timeout_ms === 'number' && command.timeout_ms > 0 ? command.timeout_ms : DEFAULT_EVAL_TIMEOUT_MS

    // mode "inject" (default): evaluate in the page main world via a <script>
    // tag injected from the content script — no debugger permission or infobar.
    // If the page CSP blocks the injected script, the content script falls back
    // to the CDP bridge automatically and the result reports via "cdp-fallback".
    if (params.mode !== 'cdp') {
      try {
        const injectCommand = {
          ...command,
          params: { ...(command.params ?? {}), mode: 'inject', timeout_ms: timeoutMs },
        }
        const response = await this.runInTab(injectCommand)
        await report('succeeded', {
          page: response.result?.page,
          data: response.result?.data,
          via: response.result?.via,
        })
      } catch (error) {
        const detail = error instanceof Error ? error.message : String(error)
        await report('failed', undefined, detail)
      }
      return
    }

    // mode "cdp": CDP Runtime.evaluate via the debugger API (bypasses CSP).
    const result = await evalInTabViaCdp(api, tabId, expression, timeoutMs)
    if (result.error) {
      await report('failed', undefined, result.error)
      return
    }
    const page = await this.currentPage(command.tab_uuid)
    await report('succeeded', { page, data: result.data })
  }

  // ---------------------------------------------------------------------
  // Shared helpers
  // ---------------------------------------------------------------------

  private async runInTab(command: BrowserCommand): Promise<{ result?: BrowserCommandResult; navigating?: boolean }> {
    const tabId = await this.requireTabId(command.tab_uuid)
    const tab = await getBrowserApi().tabs.get(tabId)
    if (tab.url && !isTrackableUrl(tab.url)) {
      throw new Error('unsupported_page: cannot automate browser-internal pages')
    }

    let response: unknown
    try {
      response = await getBrowserApi().tabs.sendMessage(tabId, {
        type: 'browserCommand',
        command_id: command.command_id,
        session_id: command.session_id,
        action: command.action,
        params: command.params ?? {},
      })
    } catch {
      // Content script not injected (e.g. extension reloaded mid-flight). Try
      // injecting on demand via scripting API, then retry once.
      await this.injectContentScript(tabId)
      response = await getBrowserApi().tabs.sendMessage(tabId, {
        type: 'browserCommand',
        command_id: command.command_id,
        session_id: command.session_id,
        action: command.action,
        params: command.params ?? {},
      })
    }

    const typed = response as AutomationResult | undefined
    if (!typed?.ok) {
      throw new Error(typed?.error || 'unknown content-script error')
    }
    const result = typed.result
    return {
      result: result ? { page: result.page, scrape: result.scrape, data: result.data, via: result.via } : {},
      navigating: Boolean(result?.navigating),
    }
  }

  private async requireTabId(tabUuid: string): Promise<number> {
    const tabId = await this.resolveTabId(tabUuid)
    if (tabId === undefined) {
      throw new Error('tab_not_found: tab is no longer open; list tabs with browser_list_tabs')
    }
    return tabId
  }

  private async resolveTabId(tabUuid: string): Promise<number | undefined> {
    const known = this.tabs.tabIdFor(tabUuid)
    if (known !== undefined) {
      return known
    }
    // Registry may be stale (e.g. after a restart); refresh and retry.
    await this.tabs.refreshSnapshot()
    return this.tabs.tabIdFor(tabUuid)
  }

  private async injectContentScript(tabId: number): Promise<void> {
    const api = getBrowserApi()
    if (!api.scripting) {
      throw new Error('content script unavailable')
    }
    await api.scripting.executeScript({
      target: { tabId },
      files: ['dist/contentScript.js'],
    })
  }

  private async waitForNavigation(command: BrowserCommand): Promise<void> {
    const tabId = await this.resolveTabId(command.tab_uuid)
    if (tabId === undefined) {
      return
    }
    await new Promise<void>((resolve) => {
      let settled = false
      const api = getBrowserApi()
      const listener = (id: number, changeInfo: { status?: string }) => {
        if (id === tabId && changeInfo.status === 'complete') {
          done()
        }
      }
      const done = () => {
        if (settled) {
          return
        }
        settled = true
        clearTimeout(timeout)
        api.tabs.onUpdated.removeListener(listener)
        resolve()
      }
      const timeout = setTimeout(done, NAVIGATION_READY_TIMEOUT_MS)
      api.tabs.onUpdated.addListener(listener)
    })
  }

  private async currentPage(tabUuid: string): Promise<BrowserCommandResult['page']> {
    const tabId = await this.resolveTabId(tabUuid)
    if (tabId === undefined) {
      return { url: '', title: '' }
    }
    const tab = await getBrowserApi().tabs.get(tabId)
    return { url: tab.url ?? '', title: tab.title ?? '' }
  }

  private async waitForTabReady(tabId: number, url?: string): Promise<void> {
    const deadline = Date.now() + NAVIGATION_READY_TIMEOUT_MS
    while (Date.now() < deadline) {
      try {
        const tab = await getBrowserApi().tabs.get(tabId)
        if (tab.status === 'complete' || !url) {
          // Give the content script a beat to inject at document_idle.
          await new Promise((r) => setTimeout(r, 300))
          return
        }
      } catch {
        return
      }
      await new Promise((r) => setTimeout(r, 250))
    }
  }
}

// Re-exported helper so CommandDispatcher can keep one report path.
export type { BrowserCommand, BrowserCommandResult }
