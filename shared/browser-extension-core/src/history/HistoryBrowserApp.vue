<script setup lang="ts">
import type { GroupedHistoryEntry, HistoryDomainSummary } from '@browser-server/shared-client'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { faviconUrl, formatDuration, timeAgo } from '@browser-server/shared-utils'
import { getBrowserApi } from '../browserApi'
import { createApiClient, useUserId } from '../composables/useApiClient'
import { useExtensionSettings } from '../composables/useExtensionSettings'

const PAGE_SIZE = 100
const MIN_PANEL_WIDTH = 220
const DIVIDER_WIDTH = 6

const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

const domains = ref<HistoryDomainSummary[]>([])
const entries = ref<GroupedHistoryEntry[]>([])
const selectedDomain = ref('')
const selectedEntry = ref<GroupedHistoryEntry | null>(null)
const domainFilter = ref('')
const linkFilter = ref('')
const debouncedLinkFilter = ref('')
const domainLoading = ref(false)
const linksLoading = ref(false)
const errorMessage = ref('')
const totalEntries = ref(0)
const currentPage = ref(1)
const browser = ref<HTMLElement | null>(null)
const firstWidth = ref(280)
const secondWidth = ref(430)
const resizing = ref(false)
let linkFilterTimer: ReturnType<typeof setTimeout> | undefined
let domainRequest = 0
let linksRequest = 0
let resizeObserver: ResizeObserver | undefined
let stopResize: (() => void) | undefined

const filteredDomains = computed(() => {
  const query = domainFilter.value.trim().toLowerCase()
  return query ? domains.value.filter((item) => item.domain.includes(query)) : domains.value
})
const totalPages = computed(() => Math.max(1, Math.ceil(totalEntries.value / PAGE_SIZE)))
const gridStyle = computed(() => ({
  gridTemplateColumns: `${firstWidth.value}px ${DIVIDER_WIDTH}px ${secondWidth.value}px ${DIVIDER_WIDTH}px minmax(${MIN_PANEL_WIDTH}px, 1fr)`,
}))
const isReady = computed(() => Boolean(client.value) && userId.value > 0)

async function loadDomains() {
  if (!client.value || !userId.value) return
  const request = ++domainRequest
  domainLoading.value = true
  errorMessage.value = ''
  try {
    const result = await client.value.getHistoryDomains(userId.value)
    if (request !== domainRequest) return
    domains.value = result
    if (!selectedDomain.value || !result.some((item) => item.domain === selectedDomain.value)) {
      selectDomain(result[0]?.domain ?? '')
    }
  } catch (error) {
    if (request !== domainRequest) return
    domains.value = []
    errorMessage.value = error instanceof Error ? error.message : 'Could not load history.'
  } finally {
    if (request === domainRequest) domainLoading.value = false
  }
}

async function refresh() {
  await loadDomains()
  await loadEntries()
}

async function loadEntries() {
  const request = ++linksRequest
  if (!client.value || !userId.value || !selectedDomain.value) {
    entries.value = []
    totalEntries.value = 0
    return
  }
  linksLoading.value = true
  errorMessage.value = ''
  try {
    const result = await client.value.getGroupedHistory({
      user_id: userId.value,
      domain: selectedDomain.value,
      q: debouncedLinkFilter.value.trim() || undefined,
      column: 'all',
      limit: PAGE_SIZE,
      offset: (currentPage.value - 1) * PAGE_SIZE,
    })
    if (request !== linksRequest) return
    entries.value = result.entries
    totalEntries.value = result.total
  } catch (error) {
    if (request !== linksRequest) return
    entries.value = []
    totalEntries.value = 0
    errorMessage.value = error instanceof Error ? error.message : 'Could not load links.'
  } finally {
    if (request === linksRequest) linksLoading.value = false
  }
}

function selectDomain(domain: string) {
  if (selectedDomain.value === domain) return
  selectedDomain.value = domain
  selectedEntry.value = null
  linkFilter.value = ''
  debouncedLinkFilter.value = ''
  currentPage.value = 1
  void loadEntries()
}

function selectEntry(entry: GroupedHistoryEntry) {
  selectedEntry.value = entry
}

function openInTab(url: string) {
  void getBrowserApi().tabs.create({ url, active: true })
}

function changePage(delta: number) {
  const next = currentPage.value + delta
  if (next < 1 || next > totalPages.value) return
  currentPage.value = next
  void loadEntries()
}

function setInitialWidths() {
  const available = browser.value?.clientWidth ?? window.innerWidth
  firstWidth.value = Math.max(MIN_PANEL_WIDTH, Math.round(available * 0.22))
  secondWidth.value = Math.max(MIN_PANEL_WIDTH, Math.round(available * 0.32))
  clampWidths()
}

function clampWidths() {
  const available = browser.value?.clientWidth ?? window.innerWidth
  const maxCombined = available - MIN_PANEL_WIDTH - DIVIDER_WIDTH * 2
  firstWidth.value = Math.min(Math.max(MIN_PANEL_WIDTH, firstWidth.value), maxCombined - MIN_PANEL_WIDTH)
  secondWidth.value = Math.min(Math.max(MIN_PANEL_WIDTH, secondWidth.value), maxCombined - firstWidth.value)
}

function setupBrowserLayout() {
  resizeObserver?.disconnect()
  if (!browser.value) return
  setInitialWidths()
  resizeObserver = new ResizeObserver(clampWidths)
  resizeObserver.observe(browser.value)
}

function startResize(panel: 'first' | 'second', event: PointerEvent) {
  event.preventDefault()
  stopResize?.()
  const startX = event.clientX
  const startFirst = firstWidth.value
  const startSecond = secondWidth.value
  resizing.value = true

  function move(moveEvent: PointerEvent) {
    const delta = moveEvent.clientX - startX
    if (panel === 'first') {
      firstWidth.value = startFirst + delta
    } else {
      secondWidth.value = startSecond + delta
    }
    clampWidths()
  }

  stopResize = () => {
    resizing.value = false
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', stopResize!)
    window.removeEventListener('pointercancel', stopResize!)
    window.removeEventListener('blur', stopResize!)
    stopResize = undefined
  }

  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', stopResize)
  window.addEventListener('pointercancel', stopResize)
  window.addEventListener('blur', stopResize)
}

watch(linkFilter, (value) => {
  if (linkFilterTimer) clearTimeout(linkFilterTimer)
  linkFilterTimer = setTimeout(() => {
    debouncedLinkFilter.value = value
    currentPage.value = 1
    void loadEntries()
  }, 200)
})

watch([client, userId], () => {
  domainRequest++
  linksRequest++
  domains.value = []
  entries.value = []
  selectedDomain.value = ''
  selectedEntry.value = null
  totalEntries.value = 0
  domainLoading.value = false
  linksLoading.value = false
  if (isReady.value) {
    void loadDomains()
    void nextTick(setupBrowserLayout)
  }
}, { immediate: true })

onMounted(() => {
  void nextTick(setupBrowserLayout)
})

onBeforeUnmount(() => {
  if (linkFilterTimer) clearTimeout(linkFilterTimer)
  stopResize?.()
  resizeObserver?.disconnect()
})
</script>

<template>
  <main class="flex h-screen min-w-[760px] flex-col overflow-hidden bg-slate-950 text-slate-100">
    <header class="flex h-14 shrink-0 items-center justify-between border-b border-slate-800 px-5">
      <div>
        <h1 class="text-base font-semibold">History Browser</h1>
        <p class="text-xs text-slate-500">Browse by domain and preview visited pages</p>
      </div>
      <button type="button" class="rounded-lg border border-slate-700 px-3 py-1.5 text-xs font-medium text-slate-300 transition hover:border-slate-500 hover:text-white" @click="refresh">
        Refresh
      </button>
    </header>

    <div v-if="!isReady" class="flex flex-1 items-center justify-center text-sm text-slate-400">
      Configure a server URL, API token, and user ID in extension settings first.
    </div>
    <div v-else ref="browser" class="grid min-h-0 flex-1 overflow-hidden" :style="gridStyle">
      <section class="flex min-w-0 flex-col overflow-hidden border-r border-slate-800">
        <div class="border-b border-slate-800 p-3">
          <h2 class="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">Domains</h2>
          <input v-model="domainFilter" type="search" placeholder="Filter domains…" class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm outline-none placeholder:text-slate-600 focus:border-rose-400" />
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto p-2">
          <p v-if="domainLoading" class="p-4 text-center text-sm text-slate-500">Loading domains…</p>
          <p v-else-if="filteredDomains.length === 0" class="p-4 text-center text-sm text-slate-500">No domains found.</p>
          <button v-for="domain in filteredDomains" :key="domain.domain" type="button" class="mb-1 flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition" :class="selectedDomain === domain.domain ? 'bg-rose-500/15 text-rose-100' : 'hover:bg-slate-900'" @click="selectDomain(domain.domain)">
            <img :src="faviconUrl(`https://${domain.domain}`)" alt="" class="h-5 w-5 shrink-0 rounded" @error="($event.target as HTMLImageElement).style.visibility = 'hidden'" />
            <span class="min-w-0 flex-1">
              <span class="block truncate text-sm font-medium">{{ domain.domain }}</span>
              <span class="block text-[11px] text-slate-500">{{ domain.url_count }} links · {{ domain.visit_count }} visits</span>
            </span>
            <span v-if="domain.total_duration" class="text-[10px] tabular-nums text-slate-500">{{ formatDuration(domain.total_duration) }}</span>
          </button>
        </div>
      </section>

      <div role="separator" aria-label="Resize domain panel" class="group cursor-col-resize bg-slate-900 hover:bg-rose-500/60" @pointerdown="startResize('first', $event)">
        <div class="mx-auto h-full w-px bg-slate-700 group-hover:bg-rose-300" />
      </div>

      <section class="flex min-w-0 flex-col overflow-hidden border-r border-slate-800">
        <div class="border-b border-slate-800 p-3">
          <h2 class="mb-2 truncate text-xs font-semibold uppercase tracking-wider text-slate-400">{{ selectedDomain || 'Links' }}</h2>
          <input v-model="linkFilter" :disabled="!selectedDomain" type="search" placeholder="Filter titles and URLs…" class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm outline-none placeholder:text-slate-600 focus:border-rose-400 disabled:opacity-50" />
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto p-2">
          <p v-if="linksLoading" class="p-4 text-center text-sm text-slate-500">Loading links…</p>
          <p v-else-if="!selectedDomain" class="p-4 text-center text-sm text-slate-500">Select a domain to see its history.</p>
          <p v-else-if="entries.length === 0" class="p-4 text-center text-sm text-slate-500">No links found.</p>
          <button v-for="entry in entries" :key="entry.url" type="button" class="mb-1 flex w-full items-start gap-3 rounded-lg px-3 py-2.5 text-left transition" :class="selectedEntry?.url === entry.url ? 'bg-sky-500/15' : 'hover:bg-slate-900'" @click="selectEntry(entry)">
            <img :src="faviconUrl(entry.url)" alt="" class="mt-0.5 h-4 w-4 shrink-0 rounded-sm" @error="($event.target as HTMLImageElement).style.visibility = 'hidden'" />
            <span class="min-w-0 flex-1">
              <span class="block truncate text-sm font-medium text-slate-200">{{ entry.title || entry.url }}</span>
              <span class="block truncate text-[11px] text-slate-500">{{ entry.url }}</span>
              <span class="mt-1 block text-[10px] text-slate-600">{{ entry.count }} visits · {{ timeAgo(entry.last_visited) }}</span>
            </span>
          </button>
        </div>
        <footer v-if="totalPages > 1" class="flex shrink-0 items-center justify-between border-t border-slate-800 px-3 py-2 text-xs text-slate-500">
          <button type="button" :disabled="currentPage === 1" class="rounded px-2 py-1 hover:bg-slate-800 disabled:opacity-30" @click="changePage(-1)">Previous</button>
          <span>{{ currentPage }} / {{ totalPages }}</span>
          <button type="button" :disabled="currentPage === totalPages" class="rounded px-2 py-1 hover:bg-slate-800 disabled:opacity-30" @click="changePage(1)">Next</button>
        </footer>
      </section>

      <div role="separator" aria-label="Resize link panel" class="group cursor-col-resize bg-slate-900 hover:bg-rose-500/60" @pointerdown="startResize('second', $event)">
        <div class="mx-auto h-full w-px bg-slate-700 group-hover:bg-rose-300" />
      </div>

      <section class="flex min-w-0 flex-col overflow-hidden">
        <template v-if="selectedEntry">
          <div class="flex h-14 shrink-0 items-center gap-3 border-b border-slate-800 px-4">
            <img :src="faviconUrl(selectedEntry.url)" alt="" class="h-4 w-4 rounded-sm" />
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium">{{ selectedEntry.title || selectedEntry.url }}</p>
              <p class="truncate text-[11px] text-slate-500">{{ selectedEntry.url }}</p>
            </div>
            <button type="button" class="shrink-0 rounded-lg bg-rose-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-rose-400" @click="openInTab(selectedEntry.url)">Open in tab</button>
          </div>
          <iframe :key="selectedEntry.url" :src="selectedEntry.url" :title="`Preview of ${selectedEntry.title || selectedEntry.url}`" sandbox="allow-forms allow-modals allow-popups allow-same-origin allow-scripts" class="min-h-0 flex-1 bg-white" :class="{ 'pointer-events-none': resizing }" />
        </template>
        <div v-else class="flex flex-1 flex-col items-center justify-center gap-3 px-8 text-center">
          <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-slate-900 text-slate-600">
            <svg class="h-7 w-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 3h7v7M10 14 21 3M21 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5" /></svg>
          </div>
          <p class="text-sm font-medium text-slate-300">Select a link for quick view</p>
          <p class="max-w-sm text-xs leading-5 text-slate-600">Some websites prevent embedded previews. Use “Open in tab” when a page does not load here.</p>
        </div>
      </section>
    </div>

    <div v-if="errorMessage" class="absolute bottom-4 left-1/2 -translate-x-1/2 rounded-lg border border-rose-500/30 bg-rose-950 px-4 py-2 text-xs text-rose-200 shadow-xl">
      {{ errorMessage }}
    </div>
  </main>
</template>
