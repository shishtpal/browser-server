<script setup lang="ts">
import { computed, onMounted, reactive, ref, useTemplateRef } from 'vue'
import { faviconUrl } from '@browser-server/shared-utils'
import { getActiveTabDomain } from '../lib/browser'
import HistoryPanel from './HistoryPanel.vue'
import TodosPanel from './TodosPanel.vue'
import WalletPanel from './WalletPanel.vue'
import type { PanelKey, PanelStatus } from './types'

const props = defineProps<{ initialPanel: PanelKey }>()

const activePanel = ref<PanelKey>(props.initialPanel)
const activeDomain = ref<string | null>(null)

const historyPanel = useTemplateRef<InstanceType<typeof HistoryPanel>>('historyPanel')
const todosPanel = useTemplateRef<InstanceType<typeof TodosPanel>>('todosPanel')
const walletPanel = useTemplateRef<InstanceType<typeof WalletPanel>>('walletPanel')

const status = reactive<Record<PanelKey, PanelStatus>>({
  history: { count: 0, state: 'loading' },
  todos: { count: 0, state: 'loading' },
  wallet: { count: 0, state: 'loading' },
})

const tabs: { key: PanelKey; label: string }[] = [
  { key: 'history', label: 'History' },
  { key: 'todos', label: 'Todos' },
  { key: 'wallet', label: 'Wallet' },
]

const activeStatus = computed(() => status[activePanel.value])

const connection = computed<{ color: string; label: string }>(() => {
  const state = activeStatus.value.state
  if (state === 'error') return { color: 'bg-rose-500', label: 'Offline' }
  if (state === 'loading') return { color: 'bg-amber-400 animate-pulse', label: 'Syncing' }
  return { color: 'bg-emerald-400', label: 'Connected' }
})

const isRefreshing = computed(() => activeStatus.value.state === 'loading')

function onStatus(key: PanelKey, next: PanelStatus) {
  status[key] = next
}

function refreshActive() {
  if (activePanel.value === 'history') historyPanel.value?.refresh()
  else if (activePanel.value === 'todos') todosPanel.value?.refresh()
  else walletPanel.value?.refresh()
}

function openSettings() {
  chrome.runtime.openOptionsPage()
}

onMounted(async () => {
  activeDomain.value = await getActiveTabDomain()
})
</script>

<template>
  <main class="flex h-[560px] w-[400px] flex-col overflow-hidden bg-slate-950 text-slate-100">
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

    <!-- Tabs -->
    <nav class="grid shrink-0 grid-cols-3 gap-1 border-b border-slate-800 bg-slate-900/40 px-2 py-2">
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
        <!-- Todos icon -->
        <svg v-else-if="tab.key === 'todos'" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M9 11l3 3L22 4" />
          <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
        </svg>
        <!-- Wallet icon -->
        <svg v-else class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M19 7V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" />
          <path d="M21 7H7a2 2 0 0 0 0 4h14v-4z" />
          <circle cx="16" cy="9" r="0.5" fill="currentColor" />
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
    </div>
  </main>
</template>
