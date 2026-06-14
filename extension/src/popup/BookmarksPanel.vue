<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { faviconUrl } from '@browser-server/shared-utils'
import {
  createApiClient,
  useBookmarksView,
  useExtensionSettings,
  useUserId,
} from '../composables/composables'
import type { BookmarkSearchColumn } from '../composables/useBookmarksView'
import { getActiveTabInfo } from '../lib/browser'
import type { PanelStatus } from './types'

const emit = defineEmits<{ (event: 'status', status: PanelStatus): void }>()

const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

const { filtered, items, errorMessage, isLoading, searchQuery, searchColumn, load, addBookmark, deleteBookmark } =
  useBookmarksView(client, userId)

defineExpose({ refresh: load })

const copiedUrl = ref<string | null>(null)
const newTitle = ref('')
const newUrl = ref('')
const isSaving = ref(false)
const savedFlash = ref(false)

const columnOptions: { value: BookmarkSearchColumn; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'title', label: 'Title' },
  { value: 'url', label: 'URL' },
]

const searchPlaceholder = computed(() => {
  if (searchColumn.value === 'title') return 'Search by title…'
  if (searchColumn.value === 'url') return 'Search by URL…'
  return 'Search bookmarks…'
})

const canSave = computed(() => newTitle.value.trim() && newUrl.value.trim())

onMounted(async () => {
  const tab = await getActiveTabInfo()
  if (tab) {
    newTitle.value = tab.title
    newUrl.value = tab.url
  }
})

async function save() {
  if (!canSave.value || !userId.value) return
  isSaving.value = true
  try {
    await addBookmark({
      user_id: userId.value,
      title: newTitle.value.trim(),
      url: newUrl.value.trim(),
    })
    savedFlash.value = true
    setTimeout(() => {
      savedFlash.value = false
    }, 1500)
  } finally {
    isSaving.value = false
  }
}

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

async function remove(id: number) {
  await deleteBookmark(id)
}

const isReady = computed(() => Boolean(client.value) && userId.value > 0)
const showSkeleton = computed(() => isLoading.value && items.value.length === 0)

watch(
  [isReady, isLoading, errorMessage, filtered],
  () => {
    emit('status', {
      count: filtered.value.length,
      state: errorMessage.value ? 'error' : isLoading.value ? 'loading' : 'ready',
    })
  },
  { immediate: true, deep: true },
)

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
    <!-- Add bookmark form -->
    <div class="border-b border-slate-800 px-3 py-2.5">
      <form class="flex flex-col gap-2" @submit.prevent="save">
        <input
          v-model="newTitle"
          type="text"
          placeholder="Title"
          class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs text-slate-100 outline-none transition placeholder:text-slate-500 focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
        />
        <div class="flex gap-2">
          <input
            v-model="newUrl"
            type="text"
            placeholder="URL"
            class="min-w-0 flex-1 rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs text-slate-100 outline-none transition placeholder:text-slate-500 focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
          />
          <button
            type="submit"
            :disabled="!canSave || isSaving"
            class="flex shrink-0 items-center gap-1 rounded-lg bg-rose-500 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-rose-400 disabled:cursor-not-allowed disabled:opacity-40"
          >
            <svg v-if="savedFlash" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M20 6 9 17l-5-5" />
            </svg>
            <svg v-else class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 5v14M5 12h14" />
            </svg>
            {{ savedFlash ? 'Saved' : 'Save' }}
          </button>
        </div>
      </form>
    </div>

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
      <!-- Skeleton -->
      <div v-if="showSkeleton" class="space-y-1">
        <div v-for="n in 5" :key="n" class="flex items-center gap-3 rounded-lg px-2 py-2.5">
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
            <path d="m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v16z" />
          </svg>
        </div>
        <p class="text-sm font-medium text-slate-300">No bookmarks yet</p>
        <p class="max-w-[260px] text-xs text-slate-500">Use the form above to save the current page.</p>
      </div>

      <!-- No search results -->
      <div v-else-if="filtered.length === 0" class="flex flex-col items-center gap-3 px-4 py-10 text-center">
        <div class="flex h-12 w-12 items-center justify-center rounded-full bg-slate-800">
          <svg class="h-6 w-6 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
        <p class="text-sm font-medium text-slate-300">No results</p>
        <p class="max-w-[260px] text-xs text-slate-500">No bookmarks match "{{ searchQuery }}".</p>
      </div>

      <!-- List -->
      <ul v-else class="space-y-0.5">
        <li
          v-for="bookmark in filtered"
          :key="bookmark.id"
          class="group flex cursor-pointer items-center gap-3 rounded-lg px-2 py-2.5 transition hover:bg-slate-800/60"
          :title="`Open ${bookmark.url}`"
          @click="openUrl(bookmark.url)"
        >
          <img
            :src="faviconUrl(bookmark.url)"
            alt=""
            class="h-4 w-4 shrink-0 rounded-sm"
            @error="($event.target as HTMLImageElement).style.visibility = 'hidden'"
          />
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-medium text-slate-100" :title="bookmark.title">{{ bookmark.title }}</p>
            <p class="truncate text-[11px] text-slate-500" :title="bookmark.url">{{ bookmark.url }}</p>
          </div>
          <div class="flex shrink-0 items-center gap-1">
            <!-- Copy button -->
            <button
              type="button"
              class="flex h-7 w-7 items-center justify-center rounded text-slate-500 opacity-0 transition hover:bg-slate-700 hover:text-slate-200 group-hover:opacity-100"
              :title="copiedUrl === bookmark.url ? 'Copied!' : 'Copy URL'"
              @click.stop="copyUrl(bookmark.url)"
            >
              <svg v-if="copiedUrl === bookmark.url" class="h-3.5 w-3.5 text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M20 6 9 17l-5-5" />
              </svg>
              <svg v-else class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
              </svg>
            </button>
            <!-- Delete button -->
            <button
              type="button"
              class="flex h-7 w-7 items-center justify-center rounded text-slate-500 opacity-0 transition hover:bg-rose-500 hover:text-white group-hover:opacity-100"
              title="Delete"
              @click.stop="remove(bookmark.id)"
            >
              <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
              </svg>
            </button>
          </div>
        </li>
      </ul>
    </div>
  </section>
</template>
