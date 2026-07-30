<template>
  <article
    class="group relative flex cursor-pointer flex-col rounded-xl border border-slate-200 bg-white p-4 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-indigo-300 hover:shadow-md dark:border-white/10 dark:bg-slate-900/60 dark:hover:border-indigo-500/40 dark:hover:bg-slate-900"
    :class="dense ? 'sm:flex-row sm:items-start sm:gap-4' : ''" role="button" tabindex="0"
    @click="$emit('open', prompt)" @keydown.enter.prevent="$emit('open', prompt)"
    @keydown.space.prevent="$emit('open', prompt)">
    <!-- hover actions -->
    <div class="absolute right-2.5 top-2.5 flex gap-1 opacity-0 transition group-hover:opacity-100">
      <button
        class="grid h-7 w-7 place-items-center rounded-md border border-slate-200 bg-white/90 text-slate-500 shadow-sm transition hover:border-indigo-300 hover:text-indigo-600 dark:border-white/10 dark:bg-slate-950/80 dark:text-slate-300 dark:hover:text-indigo-300"
        type="button" title="Use this prompt" @click.stop="$emit('use', prompt)">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14m-6-6 6 6-6 6" />
        </svg>
      </button>
      <button
        class="grid h-7 w-7 place-items-center rounded-md border border-slate-200 bg-white/90 text-slate-500 shadow-sm transition hover:border-indigo-300 hover:text-indigo-600 dark:border-white/10 dark:bg-slate-950/80 dark:text-slate-300 dark:hover:text-indigo-300"
        type="button" title="Copy content" @click.stop="$emit('copy', prompt)">
        <svg v-if="copied" class="h-3.5 w-3.5 text-emerald-500" fill="none" stroke="currentColor" stroke-width="2.4"
          viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="m5 13 4 4L19 7" />
        </svg>
        <svg v-else class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round"
            d="M8 16H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v2m-6 12h8a2 2 0 0 0 2-2v-8a2 2 0 0 0-2-2h-8a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2Z" />
        </svg>
      </button>
      <button
        class="grid h-7 w-7 place-items-center rounded-md border border-slate-200 bg-white/90 text-red-400 shadow-sm transition hover:border-red-300 hover:text-red-600 dark:border-white/10 dark:bg-slate-950/80 dark:hover:text-red-400"
        type="button" title="Delete prompt" @click.stop="$emit('delete', prompt)">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round"
            d="M19 7l-.867 12.142A2 2 0 0 1 16.138 21H7.862a2 2 0 0 1-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v3M4 7h16" />
        </svg>
      </button>
    </div>

    <div class="flex min-w-0 flex-1 flex-col" :class="dense ? 'sm:pr-0' : ''">
      <h4 class="truncate pr-20 text-[0.9rem] font-semibold text-slate-900 dark:text-slate-100">
        {{ prompt.title || 'Untitled prompt' }}
      </h4>
      <p v-if="prompt.description" class="mt-1 line-clamp-2 text-[0.76rem] text-slate-500 dark:text-slate-400">
        {{ prompt.description }}
      </p>
      <p v-if="!dense"
        class="mt-3 line-clamp-4 whitespace-pre-wrap rounded-lg bg-slate-50 p-2.5 font-mono text-[0.72rem] leading-5 text-slate-500 dark:bg-slate-950/60 dark:text-slate-400">
        {{ prompt.content }}</p>

      <div class="mt-auto flex flex-wrap items-center gap-1 pt-3">
        <span v-for="tag in (prompt.tags || []).slice(0, 4)" :key="tag"
          class="rounded-full bg-indigo-50 px-2 py-0.5 text-[0.62rem] font-medium text-indigo-600 dark:bg-indigo-500/15 dark:text-indigo-300">{{
            tag }}</span>
        <span v-if="(prompt.tags?.length || 0) > 4" class="text-[0.62rem] text-slate-400 dark:text-slate-500">
          +{{ (prompt.tags?.length || 0) - 4 }}
        </span>
      </div>

      <div
        class="mt-2 flex items-center justify-end gap-1 border-t border-slate-100 pt-2 text-[0.62rem] text-slate-400 dark:border-white/5 dark:text-slate-500">
        <svg class="h-3 w-3" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 2m6-2a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
        <span>{{ formatShortDate(prompt.updated_at || prompt.created_at) }}</span>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import type { PromptResponse } from '../../types'
import { formatShortDate } from './format'

defineProps<{
  prompt: PromptResponse
  copied: boolean
  /** Compact/list layout — hides the content preview and lays out horizontally. */
  dense?: boolean
}>()

defineEmits<{
  open: [prompt: PromptResponse]
  use: [prompt: PromptResponse]
  copy: [prompt: PromptResponse]
  delete: [prompt: PromptResponse]
}>()
</script>
