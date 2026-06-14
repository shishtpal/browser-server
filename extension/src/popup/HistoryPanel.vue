<script setup lang="ts">
import { computed, watch } from 'vue'
import { faviconUrl, timeAgo } from '@browser-server/shared-utils'
import {
  createApiClient,
  useExtensionSettings,
  useHistoryView,
  useUserId,
} from '../composables/composables'
import type { PanelStatus } from './types'

const emit = defineEmits<{ (event: 'status', status: PanelStatus): void }>()

const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

const { grouped, errorMessage, isLoading, load } = useHistoryView(client, userId)

defineExpose({ refresh: load })

const isReady = computed(() => Boolean(client.value) && userId.value > 0)
const showSkeleton = computed(() => isLoading.value && grouped.value.length === 0)

watch(
  [isReady, isLoading, errorMessage, grouped],
  () => {
    emit('status', {
      count: grouped.value.length,
      state: errorMessage.value ? 'error' : isLoading.value ? 'loading' : 'ready',
    })
  },
  { immediate: true, deep: true },
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
  <section class="px-2 py-2">
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
    <div v-else-if="grouped.length === 0" class="flex flex-col items-center gap-3 px-4 py-12 text-center">
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

    <!-- List -->
    <ul v-else class="space-y-0.5">
      <li
        v-for="entry in grouped"
        :key="entry.url"
        class="group flex items-center gap-3 rounded-lg px-2 py-2.5 transition hover:bg-slate-800/60"
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
        <div class="flex shrink-0 flex-col items-end gap-1">
          <span
            v-if="entry.count > 1"
            class="rounded-full bg-rose-500/10 px-2 py-0.5 text-[10px] font-semibold tabular-nums text-rose-300"
            :title="`${entry.count} visits`"
          >
            ×{{ entry.count }}
          </span>
          <span class="text-[10px] text-slate-500">{{ timeAgo(entry.lastVisited) }}</span>
        </div>
      </li>
    </ul>
  </section>
</template>
