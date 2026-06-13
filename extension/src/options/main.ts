import { DEFAULT_SETTINGS, getSettings, resetSettings, saveSettings } from '../lib/settings'
import '../styles/tailwind.css'

const app = document.querySelector<HTMLDivElement>('#app')
if (!app) {
  throw new Error('Options root element not found')
}

app.innerHTML = `
  <main class="mx-auto max-w-xl p-6">
    <section class="rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-xl">
      <header class="mb-6">
        <h1 class="text-2xl font-semibold text-rose-400">Extension Settings</h1>
        <p class="mt-2 text-sm text-slate-400">Configure how the browser extension connects to your local Browser Server instance.</p>
      </header>

      <div class="space-y-5">
        <label class="block">
          <span class="mb-2 block text-sm font-medium text-slate-300">Server URL</span>
          <input id="api-base" type="text" class="w-full rounded-lg border border-slate-700 bg-slate-950 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-rose-400 focus:outline-none" placeholder="http://localhost:8080" />
          <span class="mt-2 block text-xs text-slate-500">Base URL of the running browser-server instance.</span>
        </label>

        <label class="block">
          <span class="mb-2 block text-sm font-medium text-slate-300">User ID</span>
          <input id="user-id" type="number" min="1" class="w-full rounded-lg border border-slate-700 bg-slate-950 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-rose-400 focus:outline-none" placeholder="1" />
          <span class="mt-2 block text-xs text-slate-500">Numeric ID of the user to record history under.</span>
        </label>

        <label class="flex items-start gap-3 rounded-lg border border-slate-800 bg-slate-950/60 px-4 py-3">
          <input id="auto-capture" type="checkbox" class="mt-1 h-4 w-4 accent-rose-500" />
          <span>
            <span class="block text-sm font-medium text-slate-200">Auto-capture screenshots</span>
            <span class="mt-1 block text-xs text-slate-500">Capture the active tab when you open the Todos view.</span>
          </span>
        </label>
      </div>

      <div class="mt-6 flex gap-3">
        <button id="btn-reset" class="flex-1 rounded-lg border border-slate-700 px-4 py-3 text-sm font-medium text-slate-300 hover:border-slate-500 hover:text-white">Reset Defaults</button>
        <button id="btn-save" class="flex-1 rounded-lg bg-rose-500 px-4 py-3 text-sm font-medium text-white hover:bg-rose-400">Save</button>
      </div>

      <p id="status" class="mt-4 min-h-5 text-center text-sm text-slate-400"></p>
    </section>
  </main>
`

const apiBaseEl = document.querySelector<HTMLInputElement>('#api-base')!
const userIdEl = document.querySelector<HTMLInputElement>('#user-id')!
const autoCaptureEl = document.querySelector<HTMLInputElement>('#auto-capture')!
const statusEl = document.querySelector<HTMLParagraphElement>('#status')!

function showStatus(message: string, ok: boolean): void {
  statusEl.textContent = message
  statusEl.className = ok ? 'mt-4 min-h-5 text-center text-sm text-emerald-400' : 'mt-4 min-h-5 text-center text-sm text-rose-400'

  window.setTimeout(() => {
    statusEl.textContent = ''
    statusEl.className = 'mt-4 min-h-5 text-center text-sm text-slate-400'
  }, 2500)
}

async function loadForm(): Promise<void> {
  const settings = await getSettings()
  apiBaseEl.value = settings.apiBase
  userIdEl.value = settings.userId
  autoCaptureEl.checked = settings.autoCapture
}

async function handleSave(): Promise<void> {
  const apiBase = apiBaseEl.value.trim()
  const userId = userIdEl.value.trim()
  const autoCapture = autoCaptureEl.checked

  if (!apiBase) {
    showStatus('Server URL is required.', false)
    return
  }

  try {
    new URL(apiBase)
  } catch {
    showStatus('Invalid server URL.', false)
    return
  }

  const numericUserId = Number.parseInt(userId, 10)
  if (!userId || Number.isNaN(numericUserId) || numericUserId < 1) {
    showStatus('User ID must be a positive number.', false)
    return
  }

  await saveSettings({ apiBase, userId, autoCapture })
  showStatus('Settings saved.', true)
}

async function handleReset(): Promise<void> {
  const defaults = await resetSettings()
  apiBaseEl.value = defaults.apiBase
  userIdEl.value = defaults.userId
  autoCaptureEl.checked = defaults.autoCapture
  showStatus('Reset to defaults.', true)
}

document.querySelector<HTMLButtonElement>('#btn-save')!.addEventListener('click', () => void handleSave())
document.querySelector<HTMLButtonElement>('#btn-reset')!.addEventListener('click', () => void handleReset())

apiBaseEl.placeholder = DEFAULT_SETTINGS.apiBase
userIdEl.placeholder = DEFAULT_SETTINGS.userId
void loadForm()
