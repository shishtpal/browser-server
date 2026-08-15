import type { BrowserApi } from '../browserApi'

/**
 * Eval execution modes for JS steps:
 *   - "inject" — main-world evaluation by injecting a <script> tag from the
 *     content script (no debugger permission, no debugging infobar). Fails on
 *     pages whose CSP blocks inline scripts.
 *   - "cdp" — CDP Runtime.evaluate via the debugger API from the background
 *     (bypasses page CSP, needs the debugger permission, shows the infobar).
 */
export type EvalMode = 'inject' | 'cdp'

/** Standalone browser_eval fallback budget — matches the server-side default. */
export const DEFAULT_EVAL_TIMEOUT_MS = 60_000
/** Default budget for a single eval step inside browser_execute. */
export const DEFAULT_STEP_EVAL_TIMEOUT_MS = 30_000

export interface CdpEvalResult {
  data?: string
  error?: string
}

export interface CdpPdfResult {
  /** Base64-encoded PDF (no data-URL prefix). */
  data?: string
  error?: string
}

/**
 * Wraps the user expression so CDP returns a JSON string. Serialization rules
 * (match the old content-script serializer, tightened per test findings):
 *   - functions/symbols become their String() form;
 *   - undefined becomes null (so `undefined` vs the string `"undefined"` stays
 *     unambiguous);
 *   - circular/complex results (e.g. `window`) throw a descriptive error
 *     instead of degrading to a useless `"[object Object]"`.
 *
 * The wrapper is assembled with plain string concatenation — never a template
 * literal — so a user expression containing backticks or `${...}` cannot break
 * out of the generated source. `awaitPromise` handles async expressions.
 */
export function wrapEvalExpression(expression: string): string {
  return (
    '(async () => {\n' +
    '  const __bsValue = await (' +
    expression +
    ');\n' +
    '  let __bsOut;\n' +
    '  try {\n' +
    '    __bsOut = JSON.stringify(__bsValue, (__bsKey, __bsVal) => {\n' +
    "      if (typeof __bsVal === 'function' || typeof __bsVal === 'symbol') return String(__bsVal);\n" +
    "      if (typeof __bsVal === 'undefined') return null;\n" +
    '      return __bsVal;\n' +
    '    });\n' +
    '  } catch (__bsErr) {\n' +
    "    throw new Error('eval result is not JSON-serializable (' + (__bsErr && __bsErr.message || 'circular or complex value') + '); return plain data (strings, numbers, arrays, plain objects) instead');\n" +
    '  }\n' +
    "  return __bsOut === undefined ? 'null' : __bsOut;\n" +
    '})()'
  )
}

/**
 * Evaluates an expression in a tab's main world via the CDP debugger API.
 *
 * The server-side wait already enforces `timeout_ms`, but that only marks the
 * command timed_out — the in-flight Runtime.evaluate keeps running and the
 * debugger stays attached (a hung expression holds the "debugging this tab"
 * infobar indefinitely). A local watchdog matches the budget so we detach and
 * fail at the same deadline. Late reports are ignored by the bus for
 * already-timed-out commands, so this is safe to call after the server gave up.
 */
export async function evalInTabViaCdp(
  api: BrowserApi,
  tabId: number,
  expression: string,
  timeoutMs: number = DEFAULT_EVAL_TIMEOUT_MS,
): Promise<CdpEvalResult> {
  const dbg = api.debugger
  if (!dbg) {
    return {
      error:
        'debugger_unavailable: browser_eval requires the "debugger" permission and a browser that exposes the debugger API',
    }
  }
  const target = { tabId }
  let timedOut = false
  const watchdog = setTimeout(() => {
    timedOut = true
    // Detaching cancels the in-flight evaluation and releases the tab.
    void dbg.detach(target).catch(() => undefined)
  }, timeoutMs)
  try {
    await dbg.attach(target, '1.3')
    const response = (await dbg.sendCommand(target, 'Runtime.evaluate', {
      expression: wrapEvalExpression(expression),
      awaitPromise: true,
      returnByValue: true,
      userGesture: true,
    })) as {
      exceptionDetails?: { text?: string; exception?: { description?: string; value?: unknown } }
      result?: { type?: string; subtype?: string; description?: string; value?: unknown }
    }
    if (timedOut) {
      return { error: `eval_timed_out: expression exceeded ${timeoutMs}ms` }
    }
    if (response.exceptionDetails) {
      const details = response.exceptionDetails
      const desc = details.exception?.description || details.exception?.value || details.text || 'evaluation failed'
      return { error: typeof desc === 'string' ? desc : String(desc) }
    }
    const remote = response.result
    if (remote?.subtype === 'error') {
      return { error: remote.description || 'evaluation error' }
    }
    return { data: typeof remote?.value === 'string' ? remote.value : undefined }
  } catch (error) {
    if (timedOut) {
      return { error: `eval_timed_out: expression exceeded ${timeoutMs}ms` }
    }
    const detail = error instanceof Error ? error.message : String(error)
    return { error: detail }
  } finally {
    clearTimeout(watchdog)
    await dbg.detach(target).catch(() => undefined)
  }
}

/**
 * Generates a PDF of a tab's current page via CDP `Page.printToPDF` (the
 * debugger API). This is the cross-Chromium print-to-PDF path: there is no
 * public `chrome.tabs.printToPDF` API on real Chrome/Edge, but the DevTools
 * protocol command works on every Chromium browser (Chrome, Edge, Brave, ...)
 * in headful mode. The same watchdog/budget handling as evalInTabViaCdp applies
 * so a hung print does not hold the "debugging this tab" infobar forever.
 */
export async function printToPdfViaCdp(
  api: BrowserApi,
  tabId: number,
  options: { printBackground?: boolean },
  timeoutMs: number = DEFAULT_EVAL_TIMEOUT_MS,
): Promise<CdpPdfResult> {
  const dbg = api.debugger
  if (!dbg) {
    return {
      error:
        'pdf_unsupported: print-to-PDF needs the "debugger" permission and a browser that exposes the debugger API',
    }
  }
  const target = { tabId }
  let timedOut = false
  const watchdog = setTimeout(() => {
    timedOut = true
    // Detaching cancels the in-flight print and releases the tab.
    void dbg.detach(target).catch(() => undefined)
  }, timeoutMs)
  try {
    await dbg.attach(target, '1.3')
    const response = (await dbg.sendCommand(target, 'Page.printToPDF', {
      printBackground: options.printBackground ?? true,
      preferCSSPageSize: true,
      transferMode: 'ReturnAsBase64',
    })) as { data?: string }
    if (timedOut) {
      return { error: `pdf_timed_out: print exceeded ${timeoutMs}ms` }
    }
    if (typeof response.data !== 'string' || response.data.length === 0) {
      return { error: 'pdf_failed: Page.printToPDF returned no data' }
    }
    return { data: response.data }
  } catch (error) {
    if (timedOut) {
      return { error: `pdf_timed_out: print exceeded ${timeoutMs}ms` }
    }
    const detail = error instanceof Error ? error.message : String(error)
    return { error: detail }
  } finally {
    clearTimeout(watchdog)
    await dbg.detach(target).catch(() => undefined)
  }
}
