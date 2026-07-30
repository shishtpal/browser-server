<template>
  <div class="flex items-center justify-between gap-3 border-b border-slate-200 px-4 py-2.5 dark:border-white/10">
    <div class="flex min-w-0 items-center gap-2 text-[0.75rem] text-slate-500 dark:text-slate-400">
      <span class="truncate font-semibold text-slate-700 dark:text-slate-200">{{ activeTagLabel }}</span>
      <span v-if="search" class="truncate">· results for “{{ search }}”</span>
    </div>

    <div class="flex shrink-0 items-center gap-2">
      <!-- Mobile search (header search is hidden on small screens) -->
      <div class="relative sm:hidden">
        <svg class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-4.35-4.35M17 11a6 6 0 11-12 0 6 6 0 0112 0z" />
        </svg>
        <input :value="search" type="search" placeholder="Search…"
          class="w-32 rounded-lg border border-slate-200 bg-white py-1.5 pl-8 pr-2 text-[0.78rem] text-slate-700 outline-none focus:border-indigo-400 dark:border-white/10 dark:bg-slate-900 dark:text-slate-200"
          @input="$emit('update:search', ($event.target as HTMLInputElement).value)" />
      </div>

      <select :value="sortBy"
        class="h-8 rounded-lg border border-slate-200 bg-white px-2 text-[0.75rem] font-medium text-slate-600 outline-none transition focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/15 dark:border-white/10 dark:bg-slate-900 dark:text-slate-300"
        @change="$emit('update:sortBy', ($event.target as HTMLSelectElement).value)">
        <option value="updated">Recently updated</option>
        <option value="created">Recently created</option>
        <option value="title">Title A→Z</option>
      </select>

      <div class="flex overflow-hidden rounded-lg border border-slate-200 dark:border-white/10">
        <button type="button" class="grid h-8 w-8 place-items-center transition"
          :class="layout === 'grid' ? activeBtn : idleBtn" title="Grid view" @click="$emit('update:layout', 'grid')">
          <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round"
              d="M4 5h6v6H4zM14 5h6v6h-6zM4 15h6v4H4zM14 13h6v6h-6z" />
          </svg>
        </button>
        <button type="button" class="grid h-8 w-8 place-items-center transition"
          :class="layout === 'list' ? activeBtn : idleBtn" title="List view" @click="$emit('update:layout', 'list')">
          <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round"
              d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01" />
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PromptSort, PromptLayout } from '../../composables/usePromptManager'

defineProps<{
  activeTagLabel: string
  search: string
  sortBy: PromptSort
  layout: PromptLayout
}>()

defineEmits<{
  'update:search': [value: string]
  'update:sortBy': [value: string]
  'update:layout': [value: PromptLayout]
}>()

const activeBtn = 'bg-indigo-50 text-indigo-600 dark:bg-indigo-500/15 dark:text-indigo-300'
const idleBtn = 'text-slate-400 hover:bg-slate-50 hover:text-slate-600 dark:hover:bg-white/5 dark:hover:text-slate-200'
</script>
