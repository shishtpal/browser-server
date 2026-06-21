<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { DEFAULT_SETTINGS, resetSettings, saveSettings } from '../lib/settings'
import { createApiClient } from '../composables/useApiClient'

const form = reactive({
  apiBase: DEFAULT_SETTINGS.apiBase,
  apiToken: DEFAULT_SETTINGS.apiToken,
  userId: DEFAULT_SETTINGS.userId,
  autoCapture: DEFAULT_SETTINGS.autoCapture,
})

const statusMessage = ref<string>('')
const statusKind = ref<'ok' | 'err' | 'idle'>('idle')
const isSaving = ref(false)
const isTesting = ref(false)
const connectionStatus = ref<'idle' | 'online' | 'offline'>('idle')

let statusTimer: number | undefined

const statusClass = computed(() => {
  switch (statusKind.value) {
    case 'ok':
      return 'mt-4 min-h-5 text-center text-sm text-emerald-400'
    case 'err':
      return 'mt-4 min-h-5 text-center text-sm text-rose-400'
    default:
      return 'mt-4 min-h-5 text-center text-sm text-slate-400'
  }
})

function showStatus(message: string, kind: 'ok' | 'err') {
  statusMessage.value = message
  statusKind.value = kind

  if (statusTimer !== undefined) {
    window.clearTimeout(statusTimer)
  }

  statusTimer = window.setTimeout(() => {
    statusMessage.value = ''
    statusKind.value = 'idle'
  }, 2500)
}

async function loadForm() {
  const stored = await chrome.storage.local.get('tracker_settings')
  const value = (stored as { tracker_settings?: Partial<typeof form> }).tracker_settings
  if (value) {
    Object.assign(form, value)
  }
}

async function handleSave() {
  const apiBase = form.apiBase.trim()
  const apiToken = form.apiToken.trim()
  const userId = form.userId.trim()
  const autoCapture = form.autoCapture

  if (!apiBase) {
    showStatus('Server URL is required.', 'err')
    return
  }

  try {
    new URL(apiBase)
  } catch {
    showStatus('Invalid server URL.', 'err')
    return
  }

  const numericUserId = Number.parseInt(userId, 10)
  if (!userId || Number.isNaN(numericUserId) || numericUserId < 1) {
    showStatus('User ID must be a positive number.', 'err')
    return
  }

  isSaving.value = true
  try {
    await saveSettings({ apiBase, apiToken, userId, autoCapture })
    showStatus('Settings saved.', 'ok')
  } finally {
    isSaving.value = false
  }
}

async function handleReset() {
  const defaults = await resetSettings()
  Object.assign(form, defaults)
  connectionStatus.value = 'idle'
  showStatus('Reset to defaults.', 'ok')
}

async function testConnection() {
  const apiBase = form.apiBase.trim()
  if (!apiBase) {
    showStatus('Enter a server URL first.', 'err')
    return
  }

  isTesting.value = true
  connectionStatus.value = 'idle'
  try {
    const client = createApiClient({ apiBase, apiToken: form.apiToken, userId: form.userId, autoCapture: form.autoCapture })
    const reachable = await client.ping()
    connectionStatus.value = reachable ? 'online' : 'offline'
    showStatus(reachable ? 'Server is reachable.' : 'Cannot reach server.', reachable ? 'ok' : 'err')
  } catch {
    connectionStatus.value = 'offline'
    showStatus('Connection failed.', 'err')
  } finally {
    isTesting.value = false
  }
}

onMounted(() => {
  void loadForm()
})
</script>

<template>
  <main class="mx-auto max-w-xl p-6">
    <section class="rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-xl">
      <header class="mb-6">
        <h1 class="text-2xl font-semibold text-rose-400">Extension Settings</h1>
        <p class="mt-2 text-sm text-slate-400">
          Configure how the browser extension connects to your local Browser Server instance.
        </p>
      </header>

      <div class="space-y-5">
        <label class="block">
          <span class="mb-2 block text-sm font-medium text-slate-300">Server URL</span>
          <input
            v-model="form.apiBase"
            type="text"
            class="w-full rounded-lg border border-slate-700 bg-slate-950 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-rose-400 focus:outline-none"
            placeholder="http://localhost:8080"
          />
          <span class="mt-2 block text-xs text-slate-500">
            Base URL of the running browser-server instance.
          </span>
        </label>

        <label class="block">
          <span class="mb-2 block text-sm font-medium text-slate-300">API Token</span>
          <input
            v-model="form.apiToken"
            type="password"
            autocomplete="off"
            class="w-full rounded-lg border border-slate-700 bg-slate-950 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-rose-400 focus:outline-none"
            placeholder="Paste the token from 'server token generate'"
          />
          <span class="mt-2 block text-xs text-slate-500">
            Required to authenticate with the server. Generate it by running
            <code class="rounded bg-slate-800 px-1 py-0.5 text-rose-300">server token generate</code>.
          </span>
        </label>

        <label class="block">
          <span class="mb-2 block text-sm font-medium text-slate-300">User ID</span>
          <input
            v-model="form.userId"
            type="number"
            min="1"
            class="w-full rounded-lg border border-slate-700 bg-slate-950 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-rose-400 focus:outline-none"
            placeholder="1"
          />
          <span class="mt-2 block text-xs text-slate-500">
            Numeric ID of the user to record history under.
          </span>
        </label>

        <label class="flex items-start gap-3 rounded-lg border border-slate-800 bg-slate-950/60 px-4 py-3">
          <input
            v-model="form.autoCapture"
            type="checkbox"
            class="mt-1 h-4 w-4 accent-rose-500"
          />
          <span>
            <span class="block text-sm font-medium text-slate-200">Auto-capture screenshots</span>
            <span class="mt-1 block text-xs text-slate-500">
              Capture the active tab when you open the Todos view.
            </span>
          </span>
        </label>
      </div>

      <div class="mt-6 flex items-center gap-3 rounded-lg border border-slate-800 bg-slate-950/60 px-4 py-3">
        <span
          class="inline-block h-2.5 w-2.5 rounded-full"
          :class="connectionStatus === 'idle'
            ? 'bg-slate-600'
            : connectionStatus === 'online'
              ? 'bg-emerald-400'
              : 'bg-rose-500'"
        />
        <span class="flex-1 text-sm text-slate-400">
          {{ connectionStatus === 'idle' ? 'Not tested' : connectionStatus === 'online' ? 'Server reachable' : 'Server unreachable' }}
        </span>
        <button
          type="button"
          :disabled="isTesting"
          class="rounded-lg border border-slate-700 px-3 py-1.5 text-xs font-medium text-slate-300 transition hover:border-slate-500 hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
          @click="testConnection"
        >
          {{ isTesting ? 'Testing…' : 'Test Connection' }}
        </button>
      </div>

      <div class="mt-6 flex gap-3">
        <button
          type="button"
          class="flex-1 rounded-lg border border-slate-700 px-4 py-3 text-sm font-medium text-slate-300 hover:border-slate-500 hover:text-white"
          @click="handleReset"
        >
          Reset Defaults
        </button>
        <button
          type="button"
          :disabled="isSaving"
          class="flex-1 rounded-lg bg-rose-500 px-4 py-3 text-sm font-medium text-white hover:bg-rose-400 disabled:cursor-not-allowed disabled:opacity-60"
          @click="handleSave"
        >
          {{ isSaving ? 'Saving…' : 'Save' }}
        </button>
      </div>

      <p :class="statusClass">{{ statusMessage }}</p>
    </section>

    <section class="mt-6 rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-xl">
      <h2 class="mb-4 text-lg font-semibold text-slate-200">Keyboard Shortcuts</h2>
      <p class="mb-4 text-xs text-slate-500">These shortcuts work inside the popup window.</p>
      <div class="space-y-2">
        <div class="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950/60 px-4 py-2.5">
          <span class="text-sm text-slate-300">Switch tabs</span>
          <div class="flex items-center gap-1.5">
            <kbd class="rounded border border-slate-700 bg-slate-800 px-2 py-0.5 text-xs font-medium text-slate-300">Ctrl</kbd>
            <span class="text-xs text-slate-500">+</span>
            <kbd class="rounded border border-slate-700 bg-slate-800 px-2 py-0.5 text-xs font-medium text-slate-300">1</kbd>
            <span class="text-xs text-slate-500">–</span>
            <kbd class="rounded border border-slate-700 bg-slate-800 px-2 py-0.5 text-xs font-medium text-slate-300">5</kbd>
          </div>
        </div>
        <div class="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950/60 px-4 py-2.5">
          <span class="text-sm text-slate-300">Clear search / filters</span>
          <div class="flex items-center gap-1.5">
            <kbd class="rounded border border-slate-700 bg-slate-800 px-2 py-0.5 text-xs font-medium text-slate-300">Ctrl</kbd>
            <span class="text-xs text-slate-500">+</span>
            <kbd class="rounded border border-slate-700 bg-slate-800 px-2 py-0.5 text-xs font-medium text-slate-300">Backspace</kbd>
          </div>
        </div>
      </div>
      <p class="mt-3 text-[11px] text-slate-500">
        Tab order: Usage (1), History (2), Bookmarks (3), Todos (4), Wallet (5).
      </p>
    </section>
  </main>
</template>
