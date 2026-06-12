<template>
  <article class="relative rounded-xl border border-gray-200/80 bg-white p-3 shadow-sm transition hover:-translate-y-0.5 hover:border-violet-200 hover:shadow-md dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-violet-500/30">
    <div class="absolute left-3 top-4 h-3 w-3 rounded-full border-3 border-white bg-violet-500 shadow-sm dark:border-slate-800"></div>
    <div class="pl-7">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <span class="block truncate text-sm font-black text-slate-900 transition-colors dark:text-white">{{ entry.title }}</span>
          <span class="mt-0.5 block truncate text-xs font-semibold text-blue-600 transition-colors hover:underline dark:text-blue-400">{{ entry.url }}</span>
          <div class="mt-2 flex flex-wrap gap-2">
            <span class="rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400">{{ formatDate(entry.visited_at) }}</span>
            <span v-if="entry.duration > 0" class="rounded-md bg-violet-50 px-2 py-0.5 text-[10px] font-bold text-violet-700 transition-colors dark:bg-violet-900/20 dark:text-violet-400">{{ formatDuration(entry.duration) }}</span>
          </div>
        </div>
        <button type="button" @click="emit('delete', entry.id)" class="shrink-0 rounded-lg bg-red-50 px-3 py-1.5 text-xs font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { formatDate, formatDuration } from '../../lib/utils'
import type { History } from '../../types'

interface Props {
  entry: History
}

defineProps<Props>()

const emit = defineEmits<{
  delete: [id: number]
}>()
</script>
