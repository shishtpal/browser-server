/**
 * DOM automation executor that runs inside the content script (page context).
 * The background resolves a tab, forwards the command here, and forwards the
 * result back to the server bus.
 *
 * Message contract (from background):
 *   { type: 'browserCommand', command_id, session_id, action, params }
 * Response:
 *   { ok: boolean, result?: { page?: {url,title}, scrape?: unknown, data?: string, navigating?: boolean }, error?: string }
 */

import { DEFAULT_STEP_EVAL_TIMEOUT_MS, wrapEvalExpression } from './cdpEval'

export interface BrowserCommandMessage {
  type: 'browserCommand'
  command_id: string
  session_id: string
  action: string
  params?: Record<string, unknown>
}

export interface AutomationResult {
  ok: boolean
  result?: {
    page?: { url: string; title: string }
    scrape?: unknown
    data?: string
    via?: string
    navigating?: boolean
  }
  error?: string
}

interface SelectorRef {
  css?: string
  xpath?: string
  text?: string
}

const MAX_LINKS = 200
const DEFAULT_SELECTOR_TIMEOUT_MS = 10_000
const EVAL_RESULT_EVENT = 'browser-server-eval-result'

function pageInfo(): { url: string; title: string } {
  return { url: location.href, title: document.title }
}

function resolveSelector(ref: SelectorRef): Element | null {
  if (ref?.css) {
    return document.querySelector(ref.css)
  }
  if (ref?.xpath) {
    const result = document.evaluate(
      ref.xpath,
      document,
      null,
      XPathResult.FIRST_ORDERED_NODE_TYPE,
      null,
    )
    return result.singleNodeValue instanceof Element ? result.singleNodeValue : null
  }
  if (ref?.text) {
    const text = ref.text.trim().toLowerCase()
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_ELEMENT)
    let node = walker.nextNode()
    while (node) {
      const el = node as HTMLElement
      const direct = Array.from(el.childNodes)
        .filter((child) => child.nodeType === Node.TEXT_NODE)
        .map((child) => child.textContent ?? '')
        .join(' ')
        .trim()
      if (direct.toLowerCase() === text || el.textContent?.trim().toLowerCase() === text) {
        return el
      }
      node = walker.nextNode()
    }
  }
  return null
}

function waitForSelector(ref: SelectorRef, timeoutMs: number): Promise<Element> {
  const deadline = Date.now() + timeoutMs
  return new Promise((resolve, reject) => {
    const attempt = () => {
      const el = resolveSelector(ref)
      if (el) {
        resolve(el)
        return
      }
      if (Date.now() > deadline) {
        reject(new Error('selector not found'))
        return
      }
      setTimeout(attempt, 200)
    }
    attempt()
  })
}

function fireInputEvents(el: HTMLInputElement | HTMLTextAreaElement, value: string, clear: boolean): void {
  el.focus()
  if (clear) {
    el.value = ''
  } else if (el instanceof HTMLInputElement && ['checkbox', 'radio'].includes(el.type)) {
    // Typing into checkboxes/radios is a no-op; callers should use click.
  }
  const setter = Object.getOwnPropertyDescriptor(
    el instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype,
    'value',
  )?.set
  if (setter) {
    setter.call(el, clear ? '' : value)
  } else {
    el.value = clear ? '' : value
  }
  el.dispatchEvent(new Event('input', { bubbles: true }))
  el.dispatchEvent(new Event('change', { bubbles: true }))
}

function keyNameToKey(name: string): string {
  const map: Record<string, string> = {
    enter: 'Enter',
    tab: 'Tab',
    escape: 'Escape',
    esc: 'Escape',
    backspace: 'Backspace',
    delete: 'Delete',
    arrowup: 'ArrowUp',
    arrowdown: 'ArrowDown',
    arrowleft: 'ArrowLeft',
    arrowright: 'ArrowRight',
    space: ' ',
    home: 'Home',
    end: 'End',
    pageup: 'PageUp',
    pagedown: 'PageDown',
  }
  return map[name.toLowerCase()] ?? name
}

function sendKey(keys: string): void {
  const parts = keys.split('+').map((p) => p.trim())
  const key = keyNameToKey(parts[parts.length - 1])
  const ctrl = parts.slice(0, -1).some((p) => /^ctrl$/i.test(p))
  const shift = parts.slice(0, -1).some((p) => /^shift$/i.test(p))
  const alt = parts.slice(0, -1).some((p) => /^alt$/i.test(p))
  const meta = parts.slice(0, -1).some((p) => /^meta$/i.test(p))
  const target = document.activeElement ?? document.body
  target.dispatchEvent(
    new KeyboardEvent('keydown', { key, code: key, bubbles: true, cancelable: true, ctrlKey: ctrl, shiftKey: shift, altKey: alt, metaKey: meta }),
  )
  if (!/^(?:Shift|Control|Alt|Meta)$/.test(key)) {
    target.dispatchEvent(
      new KeyboardEvent('keypress', { key, bubbles: true, cancelable: true, ctrlKey: ctrl, shiftKey: shift, altKey: alt, metaKey: meta }),
    )
  }
  target.dispatchEvent(
    new KeyboardEvent('keyup', { key, code: key, bubbles: true, cancelable: true, ctrlKey: ctrl, shiftKey: shift, altKey: alt, metaKey: meta }),
  )
}

function clickAt(el: Element, xOffset = 0, yOffset = 0): void {
  const rect = el.getBoundingClientRect()
  el.dispatchEvent(
    new MouseEvent('click', {
      bubbles: true,
      cancelable: true,
      view: window,
      clientX: rect.left + rect.width / 2 + xOffset,
      clientY: rect.top + rect.height / 2 + yOffset,
    }),
  )
}

// -------------------------------------------------------------------------
// Scrape
// -------------------------------------------------------------------------

function scrape(params: Record<string, unknown>): unknown {
  const extract = Array.isArray(params.extract)
    ? (params.extract as string[])
    : ['text', 'links']
  const scope = typeof params.scope === 'string' ? params.scope : 'page'
  const maxLinks = typeof params.max_links === 'number' ? params.max_links : 100
  const includeHidden = params.include_hidden === true
  const rowsToCSV = params.rows_to_csv === true

  let root: Document | Element = document
  if (scope === 'element' && params.selector) {
    const el = resolveSelector(params.selector as SelectorRef)
    if (!el) {
      throw new Error('element scope: selector matched nothing')
    }
    root = el
  } else if (scope === 'table' && params.selector) {
    const el = resolveSelector(params.selector as SelectorRef)
    if (!el) {
      throw new Error('table scope: selector matched nothing')
    }
    root = el
  }

  const visible = (el: Element): boolean => {
    if (includeHidden) {
      return true
    }
    const style = getComputedStyle(el)
    return style.display !== 'none' && style.visibility !== 'hidden'
  }
  const queryAll = (sel: string): Element[] => {
    if (root === document && scope === 'page') {
      return Array.from(document.querySelectorAll(sel))
    }
    if (root instanceof Element) {
      return Array.from(root.querySelectorAll(sel))
    }
    return Array.from(document.querySelectorAll(sel))
  }

  const out: Record<string, unknown> = {}

  if (extract.includes('text')) {
    const text = queryAll('h1, h2, h3, p, li, td, th, a, button, label, span')
      .filter(visible)
      .map((el) => (el.textContent ?? '').replace(/\s+/g, ' ').trim())
      .filter(Boolean)
    out.text = text.slice(0, 500)
  }

  if (extract.includes('links')) {
    const links = queryAll('a[href]')
      .filter(visible)
      .map((a) => ({
        text: (a.textContent ?? '').replace(/\s+/g, ' ').trim(),
        href: (a as HTMLAnchorElement).href,
      }))
      .filter((l) => l.text && l.href && /^https?:/i.test(l.href))
    out.links = links.slice(0, Math.min(maxLinks, MAX_LINKS))
  }

  if (extract.includes('attributes')) {
    out.attributes = {
      title: document.title,
      meta_description:
        document.querySelector('meta[name="description"]')?.getAttribute('content') ?? '',
      lang: document.documentElement.lang || '',
    }
  }

  if (extract.includes('forms')) {
    out.forms = queryAll('form').map((form, index) => ({
      index,
      action: (form as HTMLFormElement).action || '',
      method: (form as HTMLFormElement).method || 'get',
      fields: Array.from(form.querySelectorAll('input, textarea, select'))
        .map((el) => {
          const input = el as HTMLInputElement
          return {
            name: input.name || input.id || '',
            type: input.type || '',
            value: input.type === 'password' ? '***' : input.value || '',
          }
        })
        .filter((f) => f.name),
    }))
  }

  if (extract.includes('table')) {
    const table = scope === 'table' && root instanceof Element
      ? root.querySelector('table') ?? (root.tagName.toLowerCase() === 'table' ? root : null)
      : root.querySelector('table')
    if (table) {
      const rows = Array.from(table.querySelectorAll('tr')).map((row) =>
        Array.from(row.querySelectorAll('th, td')).map((cell) =>
          (cell.textContent ?? '').replace(/\s+/g, ' ').trim(),
        ),
      )
      out.table = rows
      if (rowsToCSV) {
        out.table_csv = rows
          .map((cells) =>
            cells
              .map((c) => (c.includes(',') || c.includes('"') ? `"${c.replace(/"/g, '""')}"` : c))
              .join(','),
          )
          .join('\n')
      }
    }
  }

  return out
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

// -------------------------------------------------------------------------
// Actions
// -------------------------------------------------------------------------

/**
 * Executes one automation action in the current page. Returns a serializable
 * result; throws on failure so the caller can map it to an error.
 */
export async function executeAutomationAction(action: string, params: Record<string, unknown>): Promise<AutomationResult['result']> {
  switch (action) {
    case 'navigate': {
      const url = String(params.url ?? '')
      if (!/^https?:\/\//i.test(url)) {
        throw new Error('navigate requires an http(s) url')
      }
      location.href = url
      return { page: pageInfo(), navigating: true }
    }

    case 'click': {
      const el = await waitForSelector(
        params.selector as SelectorRef,
        typeof params.selector_timeout_ms === 'number' ? params.selector_timeout_ms : DEFAULT_SELECTOR_TIMEOUT_MS,
      )
      clickAt(el)
      return { page: pageInfo() }
    }

    case 'focus': {
      const el = await waitForSelector(params.selector as SelectorRef, DEFAULT_SELECTOR_TIMEOUT_MS)
      if (el instanceof HTMLElement) {
        el.focus()
      }
      return { page: pageInfo() }
    }

    case 'type': {
      const el = await waitForSelector(
        params.selector as SelectorRef,
        typeof params.selector_timeout_ms === 'number' ? params.selector_timeout_ms : DEFAULT_SELECTOR_TIMEOUT_MS,
      )
      if (!(el instanceof HTMLInputElement) && !(el instanceof HTMLTextAreaElement)) {
        const inner = el.querySelector('input, textarea')
        if (inner instanceof HTMLInputElement || inner instanceof HTMLTextAreaElement) {
          fireInputEvents(inner, String(params.text ?? ''), params.clear === true)
          return { page: pageInfo() }
        }
        throw new Error('type target is not an input or textarea')
      }
      fireInputEvents(el, String(params.text ?? ''), params.clear === true)
      return { page: pageInfo() }
    }

    case 'select': {
      const el = await waitForSelector(params.selector as SelectorRef, DEFAULT_SELECTOR_TIMEOUT_MS)
      if (!(el instanceof HTMLSelectElement)) {
        throw new Error('select target is not a <select>')
      }
      const value = typeof params.value === 'string' ? params.value : null
      const label = typeof params.label === 'string' ? params.label : null
      let matched = -1
      for (let i = 0; i < el.options.length; i += 1) {
        const opt = el.options.item(i)
        if (!opt) {
          continue
        }
        if (value !== null && opt.value === value) {
          matched = i
          break
        }
        if (label !== null && opt.textContent?.trim() === label) {
          matched = i
          break
        }
      }
      if (matched < 0) {
        throw new Error('select option not found')
      }
      el.selectedIndex = matched
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      return { page: pageInfo() }
    }

    case 'check': {
      const el = await waitForSelector(params.selector as SelectorRef, DEFAULT_SELECTOR_TIMEOUT_MS)
      if (!(el instanceof HTMLInputElement) || (el.type !== 'checkbox' && el.type !== 'radio')) {
        throw new Error('check target is not a checkbox or radio')
      }
      const want = typeof params.checked === 'boolean' ? params.checked : true
      if (el.checked !== want) {
        el.checked = want
        el.dispatchEvent(new Event('input', { bubbles: true }))
        el.dispatchEvent(new Event('change', { bubbles: true }))
      }
      return { page: pageInfo() }
    }

    case 'press': {
      sendKey(String(params.keys ?? ''))
      return { page: pageInfo() }
    }

    case 'hover': {
      const el = await waitForSelector(params.selector as SelectorRef, DEFAULT_SELECTOR_TIMEOUT_MS)
      const rect = el.getBoundingClientRect()
      const x = rect.left + rect.width / 2
      const y = rect.top + rect.height / 2
      el.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, view: window, clientX: x, clientY: y }))
      el.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true, view: window, clientX: x, clientY: y }))
      el.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, view: window, clientX: x, clientY: y }))
      return { page: pageInfo() }
    }

    case 'drag': {
      const src = await waitForSelector(params.source as SelectorRef, DEFAULT_SELECTOR_TIMEOUT_MS)
      const srcRect = src.getBoundingClientRect()
      const sx = srcRect.left + srcRect.width / 2
      const sy = srcRect.top + srcRect.height / 2

      let tx = sx
      let ty = sy
      if (params.to && typeof params.to === 'object' && !Array.isArray(params.to)) {
        const to = params.to as SelectorRef & { x?: number; y?: number }
        if (typeof to.x === 'number' && typeof to.y === 'number') {
          tx = to.x
          ty = to.y
        } else {
          const dst = await waitForSelector(to as SelectorRef, DEFAULT_SELECTOR_TIMEOUT_MS)
          const dstRect = dst.getBoundingClientRect()
          tx = dstRect.left + dstRect.width / 2
          ty = dstRect.top + dstRect.height / 2
        }
      }

      src.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, view: window, clientX: sx, clientY: sy }))
      window.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, view: window, clientX: tx, clientY: ty }))
      document.elementFromPoint(tx, ty)?.dispatchEvent(
        new MouseEvent('mouseup', { bubbles: true, view: window, clientX: tx, clientY: ty }),
      )
      return { page: pageInfo() }
    }

    case 'scroll': {
      const ref = params.selector as SelectorRef | undefined
      if (ref && (ref.css || ref.xpath || ref.text)) {
        const el = await waitForSelector(ref, DEFAULT_SELECTOR_TIMEOUT_MS)
        el.scrollIntoView({ behavior: 'smooth', block: 'center' })
      } else {
        const direction = String(params.direction ?? 'down')
        if (direction === 'top') {
          window.scrollTo({ top: 0, behavior: 'smooth' })
        } else if (direction === 'bottom') {
          window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' })
        } else {
          const amount =
            typeof params.amount === 'number'
              ? params.amount
              : window.innerHeight * (direction === 'up' ? -1 : 1)
          window.scrollBy({ top: amount, behavior: 'smooth' })
        }
      }
      return { page: pageInfo() }
    }

    case 'wait': {
      const waitMs = typeof params.wait_ms === 'number' ? params.wait_ms : 0
      if (params.selector) {
        await waitForSelector(
          params.selector as SelectorRef,
          typeof params.selector_timeout_ms === 'number' ? params.selector_timeout_ms : DEFAULT_SELECTOR_TIMEOUT_MS,
        )
      } else if (waitMs > 0) {
        await sleep(waitMs)
      }
      return { page: pageInfo() }
    }

    case 'scrape': {
      return { page: pageInfo(), scrape: scrape(params ?? {}) }
    }

    case 'storage': {
      // localStorage/sessionStorage live in the page's origin, so storage
      // operations run in the page main world via the same eval machinery as
      // browser_eval (inject <script>, CDP fallback when the CSP blocks it).
      const type = typeof params.type === 'string' ? params.type : ''
      const storageAction = typeof params.action === 'string' ? params.action : ''
      const key = typeof params.key === 'string' ? params.key : ''
      const value = typeof params.value === 'string' ? params.value : ''
      const raw = params.raw === true
      const expression = buildStorageExpression(type, storageAction, key, value, raw)
      const timeoutMs =
        typeof params.timeout_ms === 'number' && params.timeout_ms > 0
          ? params.timeout_ms
          : DEFAULT_STEP_EVAL_TIMEOUT_MS
      const evalResult = await evalWithCdpFallback(expression, timeoutMs)
      if (evalResult.error) {
        throw new Error(evalResult.error)
      }
      return { page: pageInfo(), data: evalResult.data ?? '', via: evalResult.via }
    }

    case 'eval':
      // Eval is handled before this switch: browser_eval commands are routed
      // by handleAutomationMessage (inject mode) or the background CDP path
      // (cdp mode); browser_execute eval steps run in runOrchestrate.
      throw new Error('eval is not a primitive action')

    default:
      throw new Error(`unknown action ${action}`)
  }
}

// -------------------------------------------------------------------------
// Orchestration ("browser_execute")
// -------------------------------------------------------------------------

interface OrchestrateStep {
  action: string
  params?: Record<string, unknown>
  selector?: unknown
  screenshot?: boolean
}

/**
 * Runs an orchestration payload inside the page. Background-level steps
 * (screenshot, new_tab, cookies, pdf) are reported as a structured failure
 * so the model can re-issue them as top-level tools.
 *
 * Eval steps run in the mode selected by browser_execute: "inject" (default,
 * main-world <script> injection — no debugger permission or infobar) or "cdp"
 * (CDP Runtime.evaluate from the background; bypasses page CSP). A per-step
 * params.mode overrides the flow's mode.
 *
 * After a navigate step the whole flow is aborted and reported as successful
 * so far (the content script will be re-injected on the new document).
 */
async function runOrchestrate(
  steps: OrchestrateStep[],
  opts: { mode: 'inject' | 'cdp'; screenshotAfter: boolean; rollbackUrl: string },
): Promise<AutomationResult['result']> {
  const stepResults: Array<Record<string, unknown>> = []
  let aborted = false
  let abortReason = ''
  let lastPage = pageInfo()

  for (let i = 0; i < steps.length; i += 1) {
    const s = steps[i]
    const params: Record<string, unknown> = { ...(s.params ?? {}) }
    if (s.selector && !params.selector) {
      params.selector = s.selector
    }
    if (s.screenshot) {
      stepResults.push({ index: i, action: s.action, ok: false, error: 'screenshots are a background action; split them into a browser_screenshot tool call' })
      aborted = true
      abortReason = `step ${i} (${s.action}): cannot screenshot inside orchestration`
      break
    }
    if (s.action === 'eval') {
      const expression = typeof params.expression === 'string' ? params.expression : ''
      if (!expression.trim()) {
        stepResults.push({ index: i, action: s.action, ok: false, error: 'eval requires an expression' })
        aborted = true
        abortReason = `step ${i} (${s.action}): eval requires an expression`
        break
      }
      const mode = params.mode === 'cdp' ? 'cdp' : opts.mode
      const evalTimeoutMs =
        typeof params.timeout_ms === 'number' && params.timeout_ms > 0
          ? params.timeout_ms
          : DEFAULT_STEP_EVAL_TIMEOUT_MS
      const evalResult: EvalResult =
        mode === 'cdp'
          ? await evalViaCdpBridge(expression, evalTimeoutMs)
          : await evalWithCdpFallback(expression, evalTimeoutMs)
      if (evalResult.error) {
        stepResults.push({ index: i, action: s.action, ok: false, error: evalResult.error })
        aborted = true
        abortReason = `step ${i} (${s.action}): ${evalResult.error}`
        break
      }
      const evalEntry: Record<string, unknown> = { index: i, action: s.action, ok: true, data: evalResult.data ?? '' }
      if (evalResult.via) {
        evalEntry.via = evalResult.via
      }
      stepResults.push(evalEntry)
      continue
    }
    try {
      const result = await executeAutomationAction(s.action, params)
      const entry: Record<string, unknown> = { index: i, action: s.action, ok: true }
      if (result?.page) {
        entry.page = result.page
        lastPage = result.page
      }
      if (result?.scrape !== undefined) {
        entry.scrape = result.scrape
      }
      if (result?.data !== undefined) {
        entry.data = result.data
      }
      stepResults.push(entry)
      if (result?.navigating) {
        aborted = true
        abortReason = `step ${i} (${s.action}) triggered navigation; reissue remaining steps after the page loads`
        break
      }
    } catch (error) {
      const msg = error instanceof Error ? error.message : String(error)
      stepResults.push({ index: i, action: s.action, ok: false, error: msg })
      aborted = true
      abortReason = `step ${i} (${s.action}): ${msg}`
      break
    }
  }

  // Best-effort rollback when a step failed and the caller asked for one.
  if (aborted && abortReason && opts.rollbackUrl) {
    try {
      await executeAutomationAction('navigate', { url: opts.rollbackUrl })
    } catch {
      // best effort
    }
  }

  const out: Record<string, unknown> = { steps: stepResults, page: lastPage }
  if (opts.screenshotAfter) {
    // Content scripts cannot capture the viewport; the AI follows with a
    // browser_screenshot call when it needs a frame saved.
    // eslint-disable-next-line no-console
    console.debug('browser_execute: screenshot_after requested; call browser_screenshot afterwards')
  }
  if (aborted && abortReason && stepResults.some((s) => s.ok === false)) {
    out.error = abortReason
  }
  return out
}

/**
 * Registers the automation message listener. Call once from the content script
 * entry point. Content scripts can touch the chrome/browser global directly,
 * so no adapter is needed here.
 */
export function initAutomationContentScript(): void {
  const globals = globalThis as { chrome?: { runtime?: unknown }; browser?: { runtime?: unknown } }
  const runtime = (globals.chrome?.runtime ?? globals.browser?.runtime) as
    | {
        onMessage?: {
          addListener(callback: (message: unknown, sender: unknown, sendResponse: (response: unknown) => void) => boolean | void): void
        }
      }
    | undefined
  if (!runtime?.onMessage) {
    return
  }
  runtime.onMessage.addListener((message: unknown, _sender, sendResponse) => {
    const msg = message as BrowserCommandMessage
    if (msg?.type !== 'browserCommand') {
      return false
    }
    void handleAutomationMessage(msg).then(sendResponse)
    return true
  })
}

/**
 * Builds the <script> source for inject-mode eval. The injected script runs in
 * the page's main world (content-script-added <script> elements execute with
 * full page-global access), evaluates the wrapped expression, and reports the
 * JSON result back to the content script via a CustomEvent.
 *
 * Assembled with plain string concatenation so a user expression containing
 * backticks or `${...}` cannot break the generated source. textContent (not
 * innerHTML) is used so `</script>` in the expression is inert.
 */
function buildInjectSource(expression: string, id: string): string {
  const wrapped = wrapEvalExpression(expression)
  return (
    '(function () {\n' +
    '  var __bsId = ' +
    JSON.stringify(id) +
    ';\n' +
    '  Promise.resolve((' +
    wrapped +
    ')).then(function (__bsJson) {\n' +
    "    document.dispatchEvent(new CustomEvent('" +
    EVAL_RESULT_EVENT +
    "', { detail: { id: __bsId, ok: true, data: __bsJson } }));\n" +
    '  }).catch(function (__bsErr) {\n' +
    "    document.dispatchEvent(new CustomEvent('" +
    EVAL_RESULT_EVENT +
    "', { detail: { id: __bsId, ok: false, error: String((__bsErr && __bsErr.message) || __bsErr) } }));\n" +
    '  });\n' +
    '})();'
  )
}

type EvalResult = { data?: string; error?: string; via?: 'cdp-fallback' }

// Web Storage caps. Applied inside the eval expression (before JSON transport)
// unless the caller passed raw:true, so oversized localStorage values never
// cross the content-script bridge in full.
const STORAGE_GET_CAP = 8 * 1024
const STORAGE_LIST_VALUE_CAP = 512
const STORAGE_LIST_ENTRY_CAP = 200

/**
 * Builds the eval expression for a browser_storage command. Runs in the page
 * main world (via the same inject/CDP eval machinery as browser_eval) and
 * returns a JSON-serializable object shaped per action:
 *   get    → { key, found, value }        (value capped at STORAGE_GET_CAP)
 *   set    → { ok, key, value }
 *   remove → { ok, key, found }
 *   list   → { count, entries, truncated }(entry values capped at
 *                                          STORAGE_LIST_VALUE_CAP, entry count
 *                                          at STORAGE_LIST_ENTRY_CAP)
 * Assembled with plain string concatenation and JSON.stringify for key/value
 * so arbitrary storage contents (backticks, ${...}, `</script>`) cannot break
 * the generated source. raw:true disables the value caps.
 */
function buildStorageExpression(type: string, action: string, key: string, value: string, raw: boolean): string {
  const area = type === 'session' ? 'sessionStorage' : 'localStorage'
  const k = JSON.stringify(key)
  const v = JSON.stringify(value)
  const src: string[] = ['(async () => {', `  const s = ${area};`]
  if (action === 'get') {
    src.push(
      `  const rawValue = s.getItem(${k});`,
      `  if (rawValue === null) return { key: ${k}, found: false, value: null };`,
      `  const cap = ${raw ? 'Infinity' : String(STORAGE_GET_CAP)};`,
      `  const out = rawValue.length > cap ? rawValue.slice(0, cap) + '...[' + (rawValue.length - cap) + ' more chars]' : rawValue;`,
      `  return { key: ${k}, found: true, value: out };`,
    )
  } else if (action === 'set') {
    src.push(
      `  s.setItem(${k}, ${v});`,
      `  return { ok: true, key: ${k}, value: ${v} };`,
    )
  } else if (action === 'remove') {
    src.push(
      `  const found = s.getItem(${k}) !== null;`,
      `  s.removeItem(${k});`,
      `  return { ok: true, key: ${k}, found: found };`,
    )
  } else if (action === 'list') {
    src.push(
      `  const entries = Object.create(null);`,
      `  let truncated = false;`,
      `  const cap = ${raw ? 'Infinity' : String(STORAGE_LIST_VALUE_CAP)};`,
      `  for (let i = 0; i < s.length; i++) {`,
      `    if (i >= ${String(STORAGE_LIST_ENTRY_CAP)}) { truncated = true; break; }`,
      `    const kk = s.key(i);`,
      `    const vv = s.getItem(kk);`,
      `    entries[kk] = vv !== null && vv.length > cap ? vv.slice(0, cap) + '...[' + (vv.length - cap) + ' more chars]' : vv;`,
      `  }`,
      `  return { count: s.length, entries: entries, truncated: truncated };`,
    )
  } else {
    src.push(`  throw new Error('unknown storage action');`)
  }
  src.push('})()')
  return src.join('\n')
}

/**
 * Evaluates an expression in the page main world by injecting a <script> tag,
 * then waits for the CustomEvent the injected script dispatches. Never rejects.
 * When the page CSP blocks inline scripts the script element fires its `error`
 * event instead of executing, which is detected immediately
 * (eval_inject_csp_blocked) rather than waiting out the timeout. Errors are
 * returned as `{ error }`; see evalWithCdpFallback for the automatic CDP retry.
 */
function mainWorldEvalInject(expression: string, timeoutMs: number): Promise<{ data?: string; error?: string }> {
  const id = 'bs_' + Date.now().toString(36) + '_' + Math.random().toString(36).slice(2, 10)
  return new Promise((resolve) => {
    let settled = false
    let script: HTMLScriptElement | null = null
    let timer = 0
    const finish = (result: { data?: string; error?: string }) => {
      if (settled) {
        return
      }
      settled = true
      window.clearTimeout(timer)
      document.removeEventListener(EVAL_RESULT_EVENT, listener)
      script?.remove()
      resolve(result)
    }
    const listener = (event: Event) => {
      const detail = (event as CustomEvent<{ id?: string; ok?: boolean; data?: string; error?: string }>).detail
      if (!detail || detail.id !== id) {
        return
      }
      if (detail.ok === true) {
        finish({ data: detail.data ?? '' })
      } else {
        finish({ error: detail.error || 'eval failed' })
      }
    }
    timer = window.setTimeout(() => {
      finish({
        error:
          `eval_inject_timed_out: no result within ${timeoutMs}ms; the page CSP may block injected <script> tags, or the expression hung — retry with mode "cdp"`,
      })
    }, timeoutMs)
    document.addEventListener(EVAL_RESULT_EVENT, listener)
    script = document.createElement('script')
    script.textContent = buildInjectSource(expression, id)
    script.onerror = () => {
      // A CSP that blocks inline scripts fires `error` on the script element
      // instead of executing it — fail fast rather than waiting out the whole
      // timeout (see evalWithCdpFallback for the automatic CDP retry).
      finish({
        error:
          'eval_inject_csp_blocked: the page CSP blocked the injected <script> tag (inline script execution is not allowed); retrying via CDP (mode "cdp")',
      })
    }
    const root = document.documentElement || document.head
    if (!root) {
      finish({ error: 'eval_inject: no document root to inject into' })
      return
    }
    root.appendChild(script)
  })
}

/**
 * Runs an eval with inject mode first, then — when the page CSP blocked the
 * injected script or nothing came back — automatically retries via CDP
 * (mode "cdp"). Expression-level errors (the script ran and threw) are returned
 * as-is: a CDP retry would fail identically. On a successful CDP fallback the
 * result carries `via: "cdp-fallback"` so callers can surface that the
 * debugging infobar appeared.
 */
async function evalWithCdpFallback(expression: string, timeoutMs: number): Promise<EvalResult> {
  const inject = await mainWorldEvalInject(expression, timeoutMs)
  if (!inject.error) {
    return inject
  }
  if (!inject.error.startsWith('eval_inject_csp_blocked') && !inject.error.startsWith('eval_inject_timed_out')) {
    return inject
  }
  const cdp = await evalViaCdpBridge(expression, timeoutMs)
  if (!cdp.error) {
    return { data: cdp.data, via: 'cdp-fallback' }
  }
  return { error: `${inject.error} (CDP fallback also failed: ${cdp.error})` }
}

/** Content scripts reach the background runtime directly (no adapter here). */
function contentRuntime(): { sendMessage(message: unknown): Promise<unknown> } | undefined {
  const globals = globalThis as { chrome?: { runtime?: unknown }; browser?: { runtime?: unknown } }
  return (globals.chrome?.runtime ?? globals.browser?.runtime) as
    | { sendMessage(message: unknown): Promise<unknown> }
    | undefined
}

/**
 * Routes an eval step to the background for CDP evaluation (mode "cdp"). The
 * background attaches the debugger, evaluates, and replies with the JSON string.
 * Never rejects: failures/timeouts are returned as `{ error }`.
 */
function evalViaCdpBridge(expression: string, timeoutMs: number): Promise<{ data?: string; error?: string }> {
  return new Promise((resolve) => {
    const runtime = contentRuntime()
    if (!runtime) {
      resolve({ error: 'eval_cdp: extension runtime unavailable from the content script' })
      return
    }
    const timer = window.setTimeout(() => {
      resolve({
        error:
          `eval_cdp_timed_out: no background response within ${timeoutMs}ms; check that the debugger permission is granted (browser_eval works?)`,
      })
    }, timeoutMs)
    Promise.resolve(runtime.sendMessage({ type: 'browserEvalCdp', expression, timeout_ms: timeoutMs }))
      .then((response) => {
        window.clearTimeout(timer)
        const res = response as { data?: string; error?: string } | undefined
        resolve(res ?? { error: 'eval_cdp: empty background response' })
      })
      .catch((error) => {
        window.clearTimeout(timer)
        const detail = error instanceof Error ? error.message : String(error)
        resolve({ error: `eval_cdp: ${detail}` })
      })
  })
}

async function handleAutomationMessage(message: BrowserCommandMessage): Promise<AutomationResult> {
  try {
    if (message.action === 'orchestrate') {
      const params = (message.params ?? {}) as {
        steps?: OrchestrateStep[]
        mode?: string
        screenshot_after?: boolean
        rollback_url?: string
      }
      const steps = Array.isArray(params.steps) ? params.steps : []
      const result = await runOrchestrate(steps, {
        mode: params.mode === 'cdp' ? 'cdp' : 'inject',
        screenshotAfter: params.screenshot_after === true,
        rollbackUrl: typeof params.rollback_url === 'string' ? params.rollback_url : '',
      })
      return { ok: true, result }
    }
    if (message.action === 'eval') {
      // browser_eval in "inject" mode: the background routed the command here
      // so the content script evaluates in the page main world via a <script>
      // tag (no debugger permission or infobar). When the page CSP blocks the
      // injected script, evalWithCdpFallback automatically retries via the CDP
      // bridge and marks the result with via: "cdp-fallback".
      const params = (message.params ?? {}) as { expression?: string; mode?: string; timeout_ms?: number }
      const expression = typeof params.expression === 'string' ? params.expression : ''
      if (!expression.trim()) {
        throw new Error('eval requires an expression')
      }
      const timeoutMs =
        typeof params.timeout_ms === 'number' && params.timeout_ms > 0
          ? params.timeout_ms
          : DEFAULT_STEP_EVAL_TIMEOUT_MS
      const result = await evalWithCdpFallback(expression, timeoutMs)
      if (result.error) {
        throw new Error(result.error)
      }
      return { ok: true, result: { page: pageInfo(), data: result.data ?? '', via: result.via } }
    }
    const result = await executeAutomationAction(message.action, message.params ?? {})
    return { ok: true, result }
  } catch (error) {
    const detail = error instanceof Error ? error.message : String(error)
    return { ok: false, error: detail }
  }
}
