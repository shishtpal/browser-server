<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { clearToken, getToken, setToken } from '../lib/auth'

const open = ref(false)
const draft = ref('')
const hasToken = ref(false)
const saved = ref(false)

function refresh() {
  const token = getToken()
  hasToken.value = Boolean(token)
  draft.value = token ?? ''
}

function toggle() {
  open.value = !open.value
  if (open.value) refresh()
}

function save() {
  setToken(draft.value)
  refresh()
  saved.value = true
  window.setTimeout(() => (saved.value = false), 1500)
}

function clear() {
  clearToken()
  refresh()
}

function onDocClick(e: MouseEvent) {
  const root = document.getElementById('api-token-settings')
  if (root && !root.contains(e.target as Node)) {
    open.value = false
  }
}

onMounted(() => {
  refresh()
  document.addEventListener('click', onDocClick)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
})
</script>

<template>
  <div id="api-token-settings" class="relative">
    <button
      type="button"
      @click="toggle"
      :title="hasToken ? 'API token set' : 'API token not set — click to add'"
      class="grid h-8 w-8 place-items-center rounded-lg border transition"
      :class="hasToken
        ? 'border-emerald-200 bg-emerald-50 text-emerald-600 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-400'
        : 'border-rose-200 bg-rose-50 text-rose-600 dark:border-rose-500/20 dark:bg-rose-500/10 dark:text-rose-400'"
      aria-label="API token settings"
    >
      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
      </svg>
    </button>

    <div
      v-if="open"
      class="absolute right-0 z-50 mt-2 w-72 rounded-xl border border-gray-200 bg-white p-4 shadow-xl dark:border-white/10 dark:bg-slate-900"
    >
      <p class="mb-2 text-xs font-semibold text-slate-900 dark:text-white">API Token</p>
      <p class="mb-3 text-[11px] leading-relaxed text-slate-500 dark:text-slate-400">
        Required to access the API. Generate it by running
        <code class="rounded bg-slate-100 px-1 py-0.5 text-indigo-600 dark:bg-white/10 dark:text-indigo-300">server token generate</code>.
      </p>
      <input
        v-model="draft"
        type="password"
        autocomplete="off"
        placeholder="Paste API token"
        class="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-slate-900 placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none dark:border-white/10 dark:bg-slate-950 dark:text-white"
      />
      <div class="mt-3 flex items-center gap-2">
        <button
          type="button"
          @click="save"
          class="flex-1 rounded-lg bg-indigo-500 px-3 py-2 text-xs font-semibold text-white transition hover:bg-indigo-400"
        >
          {{ saved ? 'Saved ✓' : 'Save' }}
        </button>
        <button
          type="button"
          @click="clear"
          class="rounded-lg border border-gray-200 px-3 py-2 text-xs font-medium text-slate-600 transition hover:bg-gray-100 dark:border-white/10 dark:text-slate-300 dark:hover:bg-white/10"
        >
          Clear
        </button>
      </div>
    </div>
  </div>
</template>
