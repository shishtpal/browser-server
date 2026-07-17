<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useHistoryBrowser } from '../composables/useHistoryBrowser'
import HistoryDomainList from './HistoryDomainList.vue'
import HistoryLinkList from './HistoryLinkList.vue'
import HistoryPreviewPanel from './HistoryPreviewPanel.vue'

const MIN_PANEL_WIDTH = 220
const DIVIDER_WIDTH = 6

const previewPanel = ref<InstanceType<typeof HistoryPreviewPanel> | null>(null)
const browser = ref<HTMLElement | null>(null)
const firstWidth = ref(280)
const secondWidth = ref(430)
const resizing = ref(false)
let resizeObserver: ResizeObserver | undefined
let stopResize: (() => void) | undefined

const {
  settings,
  domains,
  entries,
  selectedDomain,
  selectedEntry,
  domainFilter,
  linkFilter,
  domainLoading,
  linksLoading,
  errorMessage,
  currentPage,
  filteredDomains,
  totalPages,
  isReady,
  refresh,
  selectDomain,
  selectEntry,
  changePage,
  cleanup,
} = useHistoryBrowser(
  (entry) => previewPanel.value?.onEntryChanged(entry),
  () => void nextTick(setupBrowserLayout),
)

const unsafePreviewEnabled = computed(() => Boolean(settings.value?.unsafePreview))

const gridStyle = computed(() => ({
  gridTemplateColumns: `${firstWidth.value}px ${DIVIDER_WIDTH}px ${secondWidth.value}px ${DIVIDER_WIDTH}px minmax(${MIN_PANEL_WIDTH}px, 1fr)`,
}))

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

onMounted(() => {
  void nextTick(setupBrowserLayout)
})

onBeforeUnmount(() => {
  cleanup()
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
      <HistoryDomainList
        :domains="filteredDomains"
        :selected-domain="selectedDomain"
        :loading="domainLoading"
        :filter="domainFilter"
        @select="selectDomain"
        @update:filter="domainFilter = $event"
      />

      <div role="separator" aria-label="Resize domain panel" class="group cursor-col-resize bg-slate-900 hover:bg-rose-500/60" @pointerdown="startResize('first', $event)">
        <div class="mx-auto h-full w-px bg-slate-700 group-hover:bg-rose-300" />
      </div>

      <HistoryLinkList
        :entries="entries"
        :selected-entry="selectedEntry"
        :selected-domain="selectedDomain"
        :loading="linksLoading"
        :filter="linkFilter"
        :current-page="currentPage"
        :total-pages="totalPages"
        @select="selectEntry"
        @update:filter="linkFilter = $event"
        @change-page="changePage"
      />

      <div role="separator" aria-label="Resize link panel" class="group cursor-col-resize bg-slate-900 hover:bg-rose-500/60" @pointerdown="startResize('second', $event)">
        <div class="mx-auto h-full w-px bg-slate-700 group-hover:bg-rose-300" />
      </div>

      <HistoryPreviewPanel
        ref="previewPanel"
        :entry="selectedEntry"
        :unsafe-preview-enabled="unsafePreviewEnabled"
        :resizing="resizing"
      />
    </div>

    <div v-if="errorMessage" class="absolute bottom-4 left-1/2 -translate-x-1/2 rounded-lg border border-rose-500/30 bg-rose-950 px-4 py-2 text-xs text-rose-200 shadow-xl">
      {{ errorMessage }}
    </div>
  </main>
</template>
