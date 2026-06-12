<template>
  <div
    v-if="node.type === 'folder'"
    class="flex cursor-pointer items-center gap-2 border-b border-gray-100 px-3 py-2 text-xs font-black text-slate-600 transition hover:bg-slate-50 dark:border-slate-700/50 dark:text-slate-300 dark:hover:bg-slate-800/50"
    :style="{ paddingLeft: 12 + node.depth * 20 + 'px' }"
    @click="emit('toggleFolder', node.key)"
  >
    <svg class="h-3 w-3 shrink-0 text-slate-400 transition" :class="node.expanded ? 'rotate-90' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
    </svg>
    <svg class="h-4 w-4 shrink-0 text-amber-500" fill="currentColor" viewBox="0 0 24 24">
      <path d="M2 6a2 2 0 012-2h5l2 2h9a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
    </svg>
    <span class="font-black">{{ node.name }}</span>
    <span class="text-[10px] font-semibold text-slate-400 dark:text-slate-500">({{ node.count }})</span>
  </div>
  <div
    v-else
    class="group flex items-center gap-2 border-b border-gray-100 px-3 py-2 transition hover:bg-cyan-50/60 dark:border-slate-700/50 dark:hover:bg-cyan-900/20"
    :style="{ paddingLeft: 12 + node.depth * 20 + 'px' }"
  >
    <div class="grid h-7 w-7 shrink-0 place-items-center rounded-md bg-gradient-to-br from-slate-900 to-slate-800 text-[10px] font-black text-white dark:from-slate-950 dark:to-slate-900">{{ getInitial(node.bookmark!.title) }}</div>
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <span class="truncate text-sm font-black text-slate-900 dark:text-white" :title="node.bookmark!.title">{{ node.bookmark!.title }}</span>
        <span class="shrink-0 text-[10px] text-cyan-600 dark:text-cyan-400">{{ formatHost(node.bookmark!.url) }}</span>
      </div>
      <div class="flex items-center gap-2">
        <a :href="node.bookmark!.url" target="_blank" rel="noopener" class="truncate text-xs font-semibold text-blue-600 hover:underline dark:text-blue-400">{{ node.bookmark!.url }}</a>
        <button
          v-for="tag in node.bookmark!.tags"
          :key="tag"
          type="button"
          @click.stop="emit('filterTag', tag)"
          class="shrink-0 rounded-md bg-gray-100 px-1.5 py-0.5 text-[9px] font-black text-slate-600 transition hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400"
        >#{{ tag }}</button>
      </div>
    </div>
    <div class="flex shrink-0 gap-1 opacity-0 transition group-hover:opacity-100">
      <button type="button" @click="emit('edit', node.bookmark!)" class="rounded-lg px-2 py-1 text-[10px] font-black text-slate-500 transition hover:bg-cyan-50 hover:text-cyan-700 dark:hover:bg-cyan-900/10 dark:hover:text-cyan-400">Edit</button>
      <button type="button" @click="emit('delete', node.bookmark!.id)" class="rounded-lg px-2 py-1 text-[10px] font-black text-red-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">Delete</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FlatTreeEntry } from '../../composables/useBookmarkTree'
import { getInitial, formatHost } from '../../composables/useBookmarks'
import type { BookmarkResponse } from '../../types'

interface Props {
  node: FlatTreeEntry
}

defineProps<Props>()

const emit = defineEmits<{
  toggleFolder: [key: string]
  edit: [bookmark: BookmarkResponse]
  delete: [id: number]
  filterTag: [tag: string]
}>()
</script>
