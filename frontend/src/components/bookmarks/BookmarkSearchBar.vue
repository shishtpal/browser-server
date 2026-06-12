<template>
  <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div class="relative flex-1">
      <svg class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input
        :value="searchQuery"
        @input="onSearchInput"
        type="text"
        placeholder="Search bookmarks by title, URL, tags, folder..."
        class="w-full rounded-xl border border-gray-300 bg-white py-2 pl-10 pr-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
      />
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <div class="flex overflow-hidden rounded-lg border border-gray-300 text-xs font-black shadow-sm dark:border-slate-600">
        <button
          type="button"
          @click="emit('update:viewMode', 'flat')"
          :class="[viewMode === 'flat' ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900' : 'bg-white text-slate-600 hover:bg-gray-100 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700', 'px-2.5 py-1.5 transition']"
        >List</button>
        <button
          type="button"
          @click="emit('update:viewMode', 'tree')"
          :class="[viewMode === 'tree' ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900' : 'bg-white text-slate-600 hover:bg-gray-100 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700', 'px-2.5 py-1.5 transition']"
        >Tree</button>
      </div>
      <span class="text-xs font-semibold text-slate-500 dark:text-slate-400">{{ displayCount }} of {{ totalCount }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  searchQuery: string
  viewMode: 'flat' | 'tree'
  filteredCount: number
  treeCount: number
  totalCount: number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:searchQuery': [value: string]
  'update:viewMode': [value: 'flat' | 'tree']
}>()

const displayCount = computed(() =>
  props.viewMode === 'tree' ? props.treeCount : props.filteredCount
)

const onSearchInput = (e: Event) => {
  emit('update:searchQuery', (e.target as HTMLInputElement).value)
}
</script>
