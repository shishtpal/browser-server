import { getBrowserApi } from '../browserApi'

const UNSAFE_PREVIEW_RULE_ID = 99999

/**
 * Whether the browser supports the declarativeNetRequest API needed
 * for unsafe preview (stripping anti-framing headers).
 */
export function hasDnrSupport(): boolean {
  return Boolean(getBrowserApi().declarativeNetRequest)
}

/**
 * Install a temporary session rule that strips X-Frame-Options and CSP
 * response headers for sub_frame requests to the given hostname.
 *
 * Only effective when the declarativeNetRequest API is available.
 */
export async function installUnsafePreviewRule(hostname: string): Promise<void> {
  const dnr = getBrowserApi().declarativeNetRequest
  if (!dnr) return

  await dnr.updateSessionRules({
    removeRuleIds: [UNSAFE_PREVIEW_RULE_ID],
    addRules: [
      {
        id: UNSAFE_PREVIEW_RULE_ID,
        priority: 1,
        condition: {
          requestDomains: [hostname],
          resourceTypes: ['sub_frame'],
        },
        action: {
          type: 'modifyHeaders',
          responseHeaders: [
            { header: 'x-frame-options', operation: 'remove' },
            { header: 'content-security-policy', operation: 'remove' },
            { header: 'content-security-policy-report-only', operation: 'remove' },
          ],
        },
      },
    ],
  })
}

/**
 * Remove the unsafe preview session rule.
 */
export async function removeUnsafePreviewRule(): Promise<void> {
  const dnr = getBrowserApi().declarativeNetRequest
  if (!dnr) return

  await dnr.updateSessionRules({
    removeRuleIds: [UNSAFE_PREVIEW_RULE_ID],
  })
}
