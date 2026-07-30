<template>
  <aside
    class="flex min-w-[200px] shrink-0 flex-col overflow-hidden border-r border-slate-200 bg-slate-50 dark:border-white/10 dark:bg-slate-900/40"
    :style="{ width: `${width}px` }">
    <div class="px-4 pb-2 pt-3">
      <h3 class="text-[0.68rem] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">Tags</h3>
    </div>

    <nav class="chat-scroll flex-1 space-y-0.5 overflow-auto px-2 pb-4">
      <!-- All prompts -->
      <button type="button"
        class="flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-[0.82rem] font-medium transition"
        :class="activeTag === null ? active : idle" @click="$emit('select', null)">
        <svg class="h-4 w-4 shrink-0 opacity-70" fill="none" stroke="currentColor" stroke-width="1.8"
          viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3 7h18M3 12h18M3 17h18" />
        </svg>
        <span class="flex-1 truncate">All prompts</span>
        <span class="text-[0.7rem] tabular-nums text-slate-400 dark:text-slate-500">{{ totalCount }}</span>
      </button>

      <!-- Untagged -->
      <button v-if="untaggedCount > 0" type="button"
        class="flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-[0.82rem] font-medium transition"
        :class="activeTag === UNTAGGED_FILTER ? active : idle" @click="$emit('select', UNTAGGED_FILTER)">
        <svg class="h-4 w-4 shrink-0 opacity-70" fill="none" stroke="currentColor" stroke-width="1.8"
          viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round"
            d="M9 12h6m-3-3v6m-7 4h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v14a2 2 0 002 2z" />
        </svg>
        <span class="flex-1 truncate">Untagged</span>
        <span class="text-[0.7rem] tabular-nums text-slate-400 dark:text-slate-500">{{ untaggedCount }}</span>
      </button>

      <div class="my-2 border-t border-slate-200 dark:border-white/5"></div>

      <p v-if="tags.length === 0"
        class="rounded-lg border border-dashed border-slate-200 px-3 py-4 text-center text-[0.72rem] text-slate-400 dark:border-white/10 dark:text-slate-500">
        No tags yet
      </p>

      <button v-for="tag in tags" :key="tag.name" type="button"
        class="flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-[0.82rem] font-medium transition"
        :class="activeTag === tag.name ? active : idle" @click="$emit('select', tag.name)">
        <span class="grid h-4 w-4 shrink-0 place-items-center text-[0.9rem] font-bold opacity-60">#</span>
        <span class="flex-1 truncate">{{ tag.name }}</span>
        <span class="text-[0.7rem] tabular-nums text-slate-400 dark:text-slate-500">{{ tag.count }}</span>
      </button>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { UNTAGGED_FILTER, type TagFilter, type TagItem } from '../../composables/usePrompts'

defineProps<{
  tags: TagItem[]
  activeTag: TagFilter
  totalCount: number
  untaggedCount: number
  width: number
}>()

defineEmits<{
  select: [tag: TagFilter]
}>()

const active = 'bg-indigo-50 text-indigo-700 ring-1 ring-inset ring-indigo-500/15 dark:bg-indigo-500/15 dark:text-indigo-300 dark:ring-indigo-400/10'
const idle = 'text-slate-600 hover:bg-white hover:text-slate-900 dark:text-slate-300 dark:hover:bg-white/5 dark:hover:text-white'
</script>
