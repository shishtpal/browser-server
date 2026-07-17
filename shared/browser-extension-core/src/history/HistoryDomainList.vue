<script setup lang="ts">
import type { HistoryDomainSummary } from '@browser-server/shared-client'
import { faviconUrl, formatDuration } from '@browser-server/shared-utils'

defineProps<{
  domains: HistoryDomainSummary[]
  selectedDomain: string
  loading: boolean
  filter: string
}>()

const emit = defineEmits<{
  select: [domain: string]
  'update:filter': [value: string]
}>()
</script>

<template>
  <section class="flex min-w-0 flex-col overflow-hidden border-r border-slate-800">
    <div class="border-b border-slate-800 p-3">
      <h2 class="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">Domains</h2>
      <input
        :value="filter"
        type="search"
        placeholder="Filter domains…"
        class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm outline-none placeholder:text-slate-600 focus:border-rose-400"
        @input="emit('update:filter', ($event.target as HTMLInputElement).value)"
      />
    </div>
    <div class="min-h-0 flex-1 overflow-y-auto p-2">
      <p v-if="loading" class="p-4 text-center text-sm text-slate-500">Loading domains…</p>
      <p v-else-if="domains.length === 0" class="p-4 text-center text-sm text-slate-500">No domains found.</p>
      <button
        v-for="domain in domains"
        :key="domain.domain"
        type="button"
        class="mb-1 flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition"
        :class="selectedDomain === domain.domain ? 'bg-rose-500/15 text-rose-100' : 'hover:bg-slate-900'"
        @click="emit('select', domain.domain)"
      >
        <img
          :src="faviconUrl(`https://${domain.domain}`)"
          alt=""
          class="h-5 w-5 shrink-0 rounded"
          @error="($event.target as HTMLImageElement).style.visibility = 'hidden'"
        />
        <span class="min-w-0 flex-1">
          <span class="block truncate text-sm font-medium">{{ domain.domain }}</span>
          <span class="block text-[11px] text-slate-500">{{ domain.url_count }} links · {{ domain.visit_count }} visits</span>
        </span>
        <span v-if="domain.total_duration" class="text-[10px] tabular-nums text-slate-500">{{ formatDuration(domain.total_duration) }}</span>
      </button>
    </div>
  </section>
</template>
