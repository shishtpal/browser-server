<template>
  <tr class="transition hover:bg-violet-50/60 dark:hover:bg-violet-900/20">
    <td class="px-3 py-3">
      <div class="flex items-center gap-3">
        <div class="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-violet-50 text-violet-600 dark:bg-violet-900/20 dark:text-violet-400">
          <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064" />
          </svg>
        </div>
        <span class="block truncate text-sm font-black text-slate-900 transition-colors dark:text-white" :title="entry.title">{{ entry.title }}</span>
      </div>
    </td>
    <td class="max-w-md px-3 py-3">
      <a :href="entry.url" target="_blank" rel="noopener" class="block truncate text-sm font-semibold text-blue-600 transition-colors hover:underline dark:text-blue-400" :title="entry.url">{{ entry.url }}</a>
    </td>
    <td class="px-3 py-3">
      <span class="whitespace-nowrap rounded-md bg-gray-100 px-2 py-1 text-[10px] font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400">{{ formatDate(entry.visited_at) }}</span>
    </td>
    <td class="px-3 py-3">
      <span v-if="entry.duration > 0" class="whitespace-nowrap rounded-md bg-violet-50 px-2 py-1 text-[10px] font-bold text-violet-700 transition-colors dark:bg-violet-900/20 dark:text-violet-400">{{ formatDuration(entry.duration) }}</span>
      <span v-else class="text-[10px] text-slate-400 dark:text-slate-500">—</span>
    </td>
    <td class="px-3 py-3 text-right">
      <button type="button" @click="emit('delete', entry.id)" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-red-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">Delete</button>
    </td>
  </tr>
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
