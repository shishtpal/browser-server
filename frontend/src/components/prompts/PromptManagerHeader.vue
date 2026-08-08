<template>
  <header
    class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 bg-white/70 px-4 py-3 backdrop-blur-sm dark:border-white/10 dark:bg-slate-950/60"
  >
    <div class="flex min-w-0 items-center gap-3">
      <span
        class="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-gradient-to-br from-indigo-500 to-violet-600 text-white shadow-sm shadow-indigo-500/25"
      >
        <svg
          class="h-4.5 w-4.5"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h10M4 18h7" />
        </svg>
      </span>
      <div class="min-w-0">
        <div class="flex items-center gap-2">
          <h2
            class="truncate text-[0.95rem] font-bold tracking-tight text-slate-900 dark:text-white"
          >
            {{ view === 'editor' ? title || 'Untitled prompt' : 'Prompt Library' }}
          </h2>
          <span
            v-if="view === 'editor' && isDirty"
            class="rounded-full bg-amber-100 px-2 py-0.5 text-[0.6rem] font-semibold tracking-wide text-amber-700 uppercase dark:bg-amber-400/15 dark:text-amber-300"
            >Unsaved</span
          >
        </div>
        <p class="truncate text-[0.72rem] text-slate-500 dark:text-slate-400">
          {{ subtitle }}
          <template v-if="view === 'grid'">
            · {{ count }} prompt{{ count === 1 ? '' : 's' }}</template
          >
        </p>
      </div>
    </div>

    <div class="flex items-center gap-2">
      <div v-if="view === 'grid'" class="relative hidden sm:block">
        <svg
          class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="m21 21-4.35-4.35M17 11a6 6 0 11-12 0 6 6 0 0112 0z"
          />
        </svg>
        <input
          :value="search"
          type="search"
          placeholder="Search prompts…"
          class="w-56 rounded-lg border border-slate-200 bg-white py-1.5 pr-3 pl-8 text-[0.8rem] text-slate-700 transition outline-none placeholder:text-slate-400 focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/15 dark:border-white/10 dark:bg-slate-900 dark:text-slate-200 dark:placeholder:text-slate-500"
          @input="$emit('update:search', ($event.target as HTMLInputElement).value)"
        />
      </div>

      <button
        class="inline-flex items-center gap-1 rounded-lg bg-indigo-600 px-2.5 py-1.5 text-[0.78rem] font-semibold text-white shadow-sm transition hover:bg-indigo-700"
        type="button"
        @click="$emit('create')"
      >
        <svg
          class="h-3.5 w-3.5"
          fill="none"
          stroke="currentColor"
          stroke-width="2.2"
          viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 5v14m-7-7h14" />
        </svg>
        New
      </button>
      <button
        class="grid h-8 w-8 place-items-center rounded-lg border border-slate-200 text-slate-500 transition hover:bg-slate-50 hover:text-slate-700 dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-slate-200"
        type="button"
        aria-label="Close"
        @click="$emit('close')"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import type { PromptView } from '../../composables/usePromptManager';

defineProps<{
  view: PromptView;
  title: string;
  subtitle: string;
  count: number;
  isDirty: boolean;
  search: string;
}>();

defineEmits<{
  'update:search': [value: string];
  create: [];
  close: [];
}>();
</script>
