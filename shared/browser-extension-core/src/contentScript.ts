export interface LoginProviderAccount {
  loginProvider: string
  username: string
}

type SendMessage = (message: unknown) => Promise<unknown>
const BANNER_TIMEOUT_MS = 5_000

function isProviderResponse(value: unknown): value is { accounts: LoginProviderAccount[] } {
  return (
    typeof value === 'object' &&
    value !== null &&
    'accounts' in value &&
    Array.isArray(value.accounts)
  )
}

function showProviderBanner(accounts: LoginProviderAccount[]): void {
  if (document.getElementById('browser-server-login-providers')) return

  const host = document.createElement('div')
  host.id = 'browser-server-login-providers'
  const shadow = host.attachShadow({ mode: 'closed' })
  const panel = document.createElement('aside')
  const title = document.createElement('strong')
  const close = document.createElement('button')
  const list = document.createElement('ul')
  const style = document.createElement('style')

  title.textContent = accounts.length === 1 ? 'Login available' : `${accounts.length} logins available`
  close.type = 'button'
  close.setAttribute('aria-label', 'Dismiss login provider message')
  close.textContent = '×'

  for (const account of accounts) {
    const item = document.createElement('li')
    const provider = document.createElement('b')
    const username = document.createElement('span')
    provider.textContent = account.loginProvider || 'Password'
    username.textContent = account.username || 'Saved account'
    item.append(provider, username)
    list.append(item)
  }

  style.textContent = `
    :host { all: initial; }
    aside { position: fixed; z-index: 2147483647; top: 18px; right: 18px; width: min(340px, calc(100vw - 36px)); box-sizing: border-box; padding: 16px; border: 1px solid #334155; border-radius: 12px; background: #0f172a; color: #f8fafc; box-shadow: 0 16px 40px rgba(0,0,0,.35); font: 13px/1.4 system-ui, sans-serif; }
    strong { display: block; padding-right: 28px; color: #f8fafc; font-size: 14px; }
    button { position: absolute; top: 8px; right: 10px; border: 0; background: transparent; color: #94a3b8; cursor: pointer; font-size: 24px; line-height: 1; }
    button:hover { color: #fff; }
    ul { display: grid; gap: 8px; margin: 12px 0 0; padding: 0; list-style: none; }
    li { display: flex; justify-content: space-between; gap: 12px; padding-top: 8px; border-top: 1px solid #1e293b; }
    b { color: #34d399; overflow-wrap: anywhere; }
    span { color: #cbd5e1; text-align: right; overflow-wrap: anywhere; }
  `

  panel.append(title, close, list)
  shadow.append(style, panel)
  document.documentElement.append(host)

  const timeoutId = window.setTimeout(() => host.remove(), BANNER_TIMEOUT_MS)
  close.addEventListener('click', () => {
    window.clearTimeout(timeoutId)
    host.remove()
  })
}

export async function initLoginProviderContentScript(sendMessage: SendMessage): Promise<void> {
  try {
    const response = await sendMessage({ type: 'getLoginProviders', hostname: location.hostname })
    if (isProviderResponse(response) && response.accounts.length > 0) {
      showProviderBanner(response.accounts)
    }
  } catch {
    // The extension may be reloading or the local server may be offline.
  }
}
