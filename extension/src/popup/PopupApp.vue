<script setup lang="ts">
import type { PanelKey, PanelStatus } from './types'
import { computed, onMounted, reactive, ref, useTemplateRef, watch } from 'vue'
import { faviconUrl } from '@browser-server/shared-utils'
import { createApiClient, useExtensionSettings } from '../composables/composables'
import { getActiveTabDomain } from '../lib/browser'
import BookmarksPanel from './BookmarksPanel.vue'
import HistoryPanel from './HistoryPanel.vue'
import TodosPanel from './TodosPanel.vue'
import WalletPanel from './WalletPanel.vue'
import AnalyticsPanel from './AnalyticsPanel.vue'

const props = defineProps<{ initialPanel: PanelKey }>()

const activePanel = ref<PanelKey>(props.initialPanel)
const activeDomain = ref<string | null>(null)
const serverReachable = ref<boolean | null>(null)
const isChecking = ref(false)

const { settings } = useExtensionSettings()

const historyPanel = useTemplateRef<InstanceType<typeof HistoryPanel>>('historyPanel')
const todosPanel = useTemplateRef<InstanceType<typeof TodosPanel>>('todosPanel')
const walletPanel = useTemplateRef<InstanceType<typeof WalletPanel>>('walletPanel')
const bookmarksPanel = useTemplateRef<InstanceType<typeof BookmarksPanel>>('bookmarksPanel')
const analyticsPanel = useTemplateRef<InstanceType<typeof AnalyticsPanel>>('analyticsPanel')

const status = reactive<Record<PanelKey, PanelStatus>>({
  history: { count: 0, state: 'loading' },
  todos: { count: 0, state: 'loading' },
  wallet: { count: 0, state: 'loading' },
  bookmarks: { count: 0, state: 'loading' },
  analytics: { count: 0, state: 'loading' },
})

const tabs: { key: PanelKey; label: string }[] = [
  { key: 'analytics', label: 'Usage' },
  { key: 'history', label: 'History' },
  { key: 'bookmarks', label: 'Bookmarks' },
  { key: 'todos', label: 'Todos' },
  { key: 'wallet', label: 'Wallet' },
]

const activeStatus = computed(() => status[activePanel.value])

const connection = computed<{ color: string; label: string }>(() => {
  if (serverReachable.value === false) return { color: 'bg-rose-500', label: 'Server offline' }
  const state = activeStatus.value.state
  if (state === 'error') return { color: 'bg-rose-500', label: 'Offline' }
  if (state === 'loading') return { color: 'bg-amber-400 animate-pulse', label: 'Syncing' }
  return { color: 'bg-emerald-400', label: 'Connected' }
})

const isRefreshing = computed(() => activeStatus.value.state === 'loading' || isChecking.value)

function onStatus(key: PanelKey, next: PanelStatus) {
  status[key] = next
}

async function checkConnection() {
  if (!settings.value) return
  isChecking.value = true
  const client = createApiClient(settings.value)
  serverReachable.value = await client.ping()
  isChecking.value = false
}

function refreshActive() {
  if (serverReachable.value === false) {
    void checkConnection()
    return
  }
  if (activePanel.value === 'history') historyPanel.value?.refresh()
  else if (activePanel.value === 'todos') todosPanel.value?.refresh()
  else if (activePanel.value === 'wallet') walletPanel.value?.refresh()
  else if (activePanel.value === 'bookmarks') bookmarksPanel.value?.refresh()
  else if (activePanel.value === 'analytics') analyticsPanel.value?.refresh()
}

function openSettings() {
  chrome.runtime.openOptionsPage()
}

watch(settings, () => {
  if (settings.value) void checkConnection()
}, { immediate: true })

onMounted(async () => {
  activeDomain.value = await getActiveTabDomain()
})
</script>

<template>
  <main class="flex h-[560px] w-[480px] flex-col overflow-hidden bg-slate-950 text-slate-100">
    <!-- Header -->
    <header class="shrink-0 border-b border-slate-800 bg-slate-900/80 px-4 pt-3 pb-2 backdrop-blur">
      <div class="flex items-center justify-between gap-2">
        <div class="flex min-w-0 items-center gap-2.5">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-rose-500 to-rose-700 shadow-lg shadow-rose-900/40">
            <svg class="h-4.5 w-4.5 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="9" />
              <path d="M3 12h18M12 3a14 14 0 0 1 0 18 14 14 0 0 1 0-18" />
            </svg>
          </div>
          <div class="min-w-0">
            <h1 class="text-sm font-semibold leading-tight text-white">Browser Server</h1>
            <div class="flex items-center gap-1.5 text-[11px] text-slate-400">
              <span class="inline-block h-1.5 w-1.5 rounded-full" :class="connection.color" />
              <span>{{ connection.label }}</span>
            </div>
          </div>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <button
            type="button"
            title="Refresh"
            class="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-800 hover:text-white"
            @click="refreshActive"
          >
            <svg
              class="h-4 w-4"
              :class="{ 'animate-spin': isRefreshing }"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M21 12a9 9 0 1 1-2.64-6.36" />
              <path d="M21 3v6h-6" />
            </svg>
          </button>
          <button
            type="button"
            title="Settings"
            class="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-800 hover:text-white"
            @click="openSettings"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Active site context -->
      <div class="mt-2.5 flex items-center gap-2 rounded-lg bg-slate-800/60 px-2.5 py-1.5">
        <img
          v-if="activeDomain"
          :src="faviconUrl(`https://${activeDomain}`)"
          alt=""
          class="h-3.5 w-3.5 shrink-0 rounded-sm"
          @error="($event.target as HTMLImageElement).style.visibility = 'hidden'"
        />
        <span v-else class="inline-block h-3.5 w-3.5 shrink-0 rounded-sm bg-slate-700" />
        <span class="truncate text-xs text-slate-300" :title="activeDomain ?? ''">
          {{ activeDomain ?? 'No active site' }}
        </span>
      </div>
    </header>

    <!-- Server offline banner -->
    <div
      v-if="serverReachable === false"
      class="shrink-0 border-b border-rose-800/50 bg-rose-500/10 px-4 py-3"
    >
      <div class="flex items-start gap-2.5">
        <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-rose-500/15">
          <svg class="h-4 w-4 text-rose-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
            <path d="M12 9v4" />
            <path d="M12 17h.01" />
          </svg>
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-semibold text-rose-300">Browser server is not running</p>
          <p class="mt-0.5 text-[11px] text-rose-400/80">
            Start <span class="font-mono">{{ settings?.apiBase ?? 'localhost:8080' }}</span> to sync your data.
          </p>
          <button
            type="button"
            class="mt-2 rounded-md bg-rose-500 px-3 py-1 text-[11px] font-medium text-white transition hover:bg-rose-400 disabled:opacity-50"
            :disabled="isChecking"
            @click="checkConnection"
          >
            {{ isChecking ? 'Checking…' : 'Retry connection' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Tabs -->
    <nav class="grid shrink-0 grid-cols-5 gap-1 border-b border-slate-800 bg-slate-900/40 px-2 py-2">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        class="flex items-center justify-center gap-1.5 rounded-lg px-2 py-1.5 text-xs font-medium transition"
        :class="activePanel === tab.key
          ? 'bg-rose-500/15 text-rose-300 ring-1 ring-inset ring-rose-500/30'
          : 'text-slate-400 hover:bg-slate-800/70 hover:text-slate-200'"
        @click="activePanel = tab.key"
      >
        <!-- History icon -->
        <svg v-if="tab.key === 'history'" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 3v5h5" />
          <path d="M3.05 13A9 9 0 1 0 6 5.3L3 8" />
          <path d="M12 7v5l4 2" />
        </svg>
        <!-- Bookmarks icon -->
        <svg v-else-if="tab.key === 'bookmarks'" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v16z" />
        </svg>
        <!-- Todos icon -->
        <svg v-else-if="tab.key === 'todos'" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M9 11l3 3L22 4" />
          <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
        </svg>
        <!-- Wallet icon -->
        <svg v-else-if="tab.key === 'wallet'" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M19 7V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" />
          <path d="M21 7H7a2 2 0 0 0 0 4h14v-4z" />
          <circle cx="16" cy="9" r="0.5" fill="currentColor" />
        </svg>
        <!-- Analytics icon -->
        <svg v-else-if="tab.key === 'analytics'" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 20V10" />
          <path d="M18 20V4" />
          <path d="M6 20v-4" />
        </svg>
        <span>{{ tab.label }}</span>
        <span
          v-if="status[tab.key].count > 0"
          class="rounded-full px-1.5 text-[10px] font-semibold tabular-nums"
          :class="activePanel === tab.key ? 'bg-rose-500/30 text-rose-100' : 'bg-slate-700 text-slate-300'"
        >
          {{ status[tab.key].count }}
        </span>
      </button>
    </nav>

    <!-- Content -->
    <div class="min-h-0 flex-1 overflow-y-auto">
      <HistoryPanel
        v-show="activePanel === 'history'"
        ref="historyPanel"
        @status="onStatus('history', $event)"
      />
      <BookmarksPanel
        v-show="activePanel === 'bookmarks'"
        ref="bookmarksPanel"
        @status="onStatus('bookmarks', $event)"
      />
      <TodosPanel
        v-show="activePanel === 'todos'"
        ref="todosPanel"
        @status="onStatus('todos', $event)"
      />
      <WalletPanel
        v-show="activePanel === 'wallet'"
        ref="walletPanel"
        @status="onStatus('wallet', $event)"
      />
      <AnalyticsPanel
        v-show="activePanel === 'analytics'"
        ref="analyticsPanel"
        @status="onStatus('analytics', $event)"
      />
    </div>
  </main>
</template>
