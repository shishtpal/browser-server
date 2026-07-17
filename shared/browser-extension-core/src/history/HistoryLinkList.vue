<script setup lang="ts">
import type { GroupedHistoryEntry } from '@browser-server/shared-client'
import { faviconUrl, timeAgo } from '@browser-server/shared-utils'

defineProps<{
  entries: GroupedHistoryEntry[]
  selectedEntry: GroupedHistoryEntry | null
  selectedDomain: string
  loading: boolean
  filter: string
  currentPage: number
  totalPages: number
}>()

const emit = defineEmits<{
  select: [entry: GroupedHistoryEntry]
  'update:filter': [value: string]
  changePage: [delta: number]
}>()
</script>

<template>
  <section class="flex min-w-0 flex-col overflow-hidden border-r border-slate-800">
    <div class="border-b border-slate-800 p-3">
      <h2 class="mb-2 truncate text-xs font-semibold uppercase tracking-wider text-slate-400">{{ selectedDomain || 'Links' }}</h2>
      <input
        :value="filter"
        :disabled="!selectedDomain"
        type="search"
        placeholder="Filter titles and URLs…"
        class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm outline-none placeholder:text-slate-600 focus:border-rose-400 disabled:opacity-50"
        @input="emit('update:filter', ($event.target as HTMLInputElement).value)"
      />
    </div>
    <div class="min-h-0 flex-1 overflow-y-auto p-2">
      <p v-if="loading" class="p-4 text-center text-sm text-slate-500">Loading links…</p>
      <p v-else-if="!selectedDomain" class="p-4 text-center text-sm text-slate-500">Select a domain to see its history.</p>
      <p v-else-if="entries.length === 0" class="p-4 text-center text-sm text-slate-500">No links found.</p>
      <button
        v-for="entry in entries"
        :key="entry.url"
        type="button"
        class="mb-1 flex w-full items-start gap-3 rounded-lg px-3 py-2.5 text-left transition"
        :class="selectedEntry?.url === entry.url ? 'bg-sky-500/15' : 'hover:bg-slate-900'"
        @click="emit('select', entry)"
      >
        <img
          :src="faviconUrl(entry.url)"
          alt=""
          class="mt-0.5 h-4 w-4 shrink-0 rounded-sm"
          @error="($event.target as HTMLImageElement).style.visibility = 'hidden'"
        />
        <span class="min-w-0 flex-1">
          <span class="block truncate text-sm font-medium text-slate-200">{{ entry.title || entry.url }}</span>
          <span class="block truncate text-[11px] text-slate-500">{{ entry.url }}</span>
          <span class="mt-1 block text-[10px] text-slate-600">{{ entry.count }} visits · {{ timeAgo(entry.last_visited) }}</span>
        </span>
      </button>
    </div>
    <footer v-if="totalPages > 1" class="flex shrink-0 items-center justify-between border-t border-slate-800 px-3 py-2 text-xs text-slate-500">
      <button type="button" :disabled="currentPage === 1" class="rounded px-2 py-1 hover:bg-slate-800 disabled:opacity-30" @click="emit('changePage', -1)">Previous</button>
      <span>{{ currentPage }} / {{ totalPages }}</span>
      <button type="button" :disabled="currentPage === totalPages" class="rounded px-2 py-1 hover:bg-slate-800 disabled:opacity-30" @click="emit('changePage', 1)">Next</button>
    </footer>
  </section>
</template>
