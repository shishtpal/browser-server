<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { faviconUrl, formatDuration, timeAgo } from '@browser-server/shared-utils'
import {
  createApiClient,
  useExtensionSettings,
  useHistoryView,
  useUserId,
} from '../composables/composables'
import type { HistorySearchColumn } from '../composables/useHistoryView'
import type { PanelStatus } from './types'

const emit = defineEmits<{ (event: 'status', status: PanelStatus): void }>()

const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

const { filtered, paginatedEntries, errorMessage, isLoading, searchQuery, searchColumn, currentPage, totalPages, nextPage, prevPage, load } = useHistoryView(client, userId)

defineExpose({
  refresh: load,
  clearSearch: () => { searchQuery.value = ''; searchColumn.value = 'all' },
  hasActiveSearch: () => Boolean(searchQuery.value) || searchColumn.value !== 'all',
})

const copiedUrl = ref<string | null>(null)

function openUrl(url: string) {
  chrome.tabs.create({ url })
}

async function copyUrl(url: string) {
  try {
    await navigator.clipboard.writeText(url)
    copiedUrl.value = url
    setTimeout(() => {
      if (copiedUrl.value === url) copiedUrl.value = null
    }, 1500)
  } catch {
    // ignore
  }
}

const columnOptions: { value: HistorySearchColumn; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'title', label: 'Title' },
  { value: 'url', label: 'URL' },
]

const searchPlaceholder = computed(() => {
  if (searchColumn.value === 'title') return 'Search by title…'
  if (searchColumn.value === 'url') return 'Search by URL…'
  return 'Search history…'
})

const isReady = computed(() => Boolean(client.value) && userId.value > 0)
const showSkeleton = computed(() => isLoading.value && filtered.value.length === 0)

const totalCount = computed(() => filtered.value.length)

watch(
  [isReady, isLoading, errorMessage, totalCount],
  () => {
    emit('status', {
      count: totalCount.value,
      state: errorMessage.value ? 'error' : isLoading.value ? 'loading' : 'ready',
    })
  },
  { immediate: true },
)

// Auto-load as soon as settings (and thus the API client) become available.
watch(
  [client, userId],
  () => {
    if (isReady.value) void load()
  },
  { immediate: true },
)
</script>

<template>
  <section class="flex flex-col">
    <!-- Search bar -->
    <div class="border-b border-slate-800 px-3 py-2.5">
      <div class="flex items-center gap-2">
        <select
          v-model="searchColumn"
          class="shrink-0 rounded-lg border border-slate-700 bg-slate-900 px-2 py-1.5 text-xs font-semibold text-slate-300 outline-none transition focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
        >
          <option v-for="opt in columnOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <div class="relative flex-1">
          <svg class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="searchPlaceholder"
            class="w-full rounded-lg border border-slate-700 bg-slate-900 py-1.5 pl-8 pr-3 text-xs text-slate-100 outline-none transition placeholder:text-slate-500 focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
          />
        </div>
      </div>
    </div>

    <div class="px-2 py-2">
      <!-- Loading skeleton -->
      <div v-if="showSkeleton" class="space-y-1">
        <div v-for="n in 6" :key="n" class="flex items-center gap-3 rounded-lg px-2 py-2.5">
          <div class="h-4 w-4 shrink-0 animate-pulse rounded bg-slate-800" />
          <div class="flex-1 space-y-1.5">
            <div class="h-3 w-3/4 animate-pulse rounded bg-slate-800" />
            <div class="h-2.5 w-1/2 animate-pulse rounded bg-slate-800/70" />
          </div>
        </div>
      </div>

      <!-- Error -->
      <div v-else-if="errorMessage" class="flex flex-col items-center gap-3 px-4 py-12 text-center">
        <div class="flex h-12 w-12 items-center justify-center rounded-full bg-rose-500/10">
          <svg class="h-6 w-6 text-rose-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 9v4M12 17h.01" />
            <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          </svg>
        </div>
        <p class="text-sm font-medium text-slate-300">Can't reach the server</p>
        <p class="max-w-[280px] text-xs text-slate-500">{{ errorMessage }}</p>
        <button
          type="button"
          class="rounded-lg bg-rose-500 px-4 py-1.5 text-xs font-medium text-white transition hover:bg-rose-400"
          @click="load"
        >
          Try again
        </button>
      </div>

      <!-- Empty -->
      <div v-else-if="filtered.length === 0 && !searchQuery" class="flex flex-col items-center gap-3 px-4 py-12 text-center">
        <div class="flex h-12 w-12 items-center justify-center rounded-full bg-slate-800">
          <svg class="h-6 w-6 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 3v5h5" />
            <path d="M3.05 13A9 9 0 1 0 6 5.3L3 8" />
            <path d="M12 7v5l4 2" />
          </svg>
        </div>
        <p class="text-sm font-medium text-slate-300">No history yet</p>
        <p class="max-w-[260px] text-xs text-slate-500">Browse the web and your visited pages will show up here automatically.</p>
      </div>

      <!-- No search results -->
      <div v-else-if="filtered.length === 0" class="flex flex-col items-center gap-3 px-4 py-10 text-center">
        <div class="flex h-12 w-12 items-center justify-center rounded-full bg-slate-800">
          <svg class="h-6 w-6 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
        <p class="text-sm font-medium text-slate-300">No results</p>
        <p class="max-w-[260px] text-xs text-slate-500">No history entries match "{{ searchQuery }}".</p>
      </div>

      <!-- List -->
      <ul v-else class="space-y-0.5">
        <li
          v-for="entry in paginatedEntries"
          :key="entry.url"
          class="group flex cursor-pointer items-center gap-3 rounded-lg px-2 py-2.5 transition hover:bg-slate-800/60"
          :title="`Open ${entry.url}`"
          @click="openUrl(entry.url)"
        >
          <img
            :src="faviconUrl(entry.url)"
            alt=""
            class="h-4 w-4 shrink-0 rounded-sm"
            @error="($event.target as HTMLImageElement).style.visibility = 'hidden'"
          />
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-medium text-slate-100" :title="entry.title">{{ entry.title }}</p>
            <div class="flex items-center gap-1.5 text-[11px] text-slate-500">
              <span class="truncate" :title="entry.url">{{ entry.url }}</span>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-1.5">
            <button
              type="button"
              class="flex h-7 w-7 items-center justify-center rounded text-slate-500 opacity-0 transition hover:bg-slate-700 hover:text-slate-200 group-hover:opacity-100"
              :title="copiedUrl === entry.url ? 'Copied!' : 'Copy URL'"
              @click.stop="copyUrl(entry.url)"
            >
              <!-- Check icon when copied -->
              <svg v-if="copiedUrl === entry.url" class="h-3.5 w-3.5 text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M20 6 9 17l-5-5" />
              </svg>
              <!-- Copy icon -->
              <svg v-else class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
              </svg>
            </button>
            <div class="flex flex-col items-end gap-1">
              <span
                v-if="entry.count > 1"
                class="rounded-full bg-rose-500/10 px-2 py-0.5 text-[10px] font-semibold tabular-nums text-rose-300"
                :title="`${entry.count} visits`"
              >
                ×{{ entry.count }}
              </span>
              <span v-if="entry.totalDuration > 0" class="flex items-center gap-0.5 text-[10px] tabular-nums text-slate-400" :title="`${entry.count} visits, ${formatDuration(entry.totalDuration)} total`">
                <svg class="h-2.5 w-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 6v6l4 2" />
                </svg>
                {{ formatDuration(entry.totalDuration) }}
              </span>
              <span class="text-[10px] text-slate-500">{{ timeAgo(entry.lastVisited) }}</span>
            </div>
          </div>
        </li>
      </ul>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-between border-t border-slate-800 px-2 pt-2.5">
        <button
          type="button"
          :disabled="currentPage <= 1"
          class="flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-slate-400 transition hover:bg-slate-800 hover:text-slate-200 disabled:cursor-default disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-slate-400"
          @click="prevPage"
        >
          <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M15 18l-6-6 6-6" />
          </svg>
          Prev
        </button>
        <span class="text-xs tabular-nums text-slate-500">
          {{ currentPage }} / {{ totalPages }}
        </span>
        <button
          type="button"
          :disabled="currentPage >= totalPages"
          class="flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-slate-400 transition hover:bg-slate-800 hover:text-slate-200 disabled:cursor-default disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-slate-400"
          @click="nextPage"
        >
          Next
          <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 18l6-6-6-6" />
          </svg>
        </button>
      </div>
    </div>
  </section>
</template>
