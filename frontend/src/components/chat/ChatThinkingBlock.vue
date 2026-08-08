<template>
  <div
    v-if="reasoning"
    class="mb-2 overflow-hidden rounded-lg border border-slate-200/70 bg-slate-50/60 dark:border-white/10 dark:bg-white/5"
  >
    <button
      type="button"
      class="flex w-full items-center gap-1.5 px-2.5 py-1.5 text-left text-[0.78em] font-medium text-slate-500 transition hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <svg
        class="h-3.5 w-3.5 shrink-0"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M9.75 3.5a4.75 4.75 0 0 0-4.75 4.75c0 1.07.354 2.057.95 2.85-.6.34-.95.98-.95 1.68 0 .78.44 1.45 1.09 1.78A2.25 2.25 0 0 0 8 18.5h.25c.42 1.15 1.52 2 2.85 2h.05a3 3 0 0 0 2.85-2V5.75a2.25 2.25 0 0 0-2.25-2.25h-2Z"
        />
      </svg>
      <span
        class="inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-indigo-400"
        v-if="streaming"
      ></span>
      {{ streaming ? 'Thinking…' : 'Thought' }}
      <svg
        class="ml-auto h-3.5 w-3.5 text-slate-400 transition-transform"
        :class="expanded ? 'rotate-180' : ''"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m19 9-7 7-7-7" />
      </svg>
    </button>
    <div
      v-show="expanded"
      class="max-h-64 overflow-auto border-t border-slate-200/70 px-2.5 py-2 font-sans text-[0.8em] leading-[1.6] break-words whitespace-pre-wrap text-slate-500 italic dark:border-white/10 dark:text-slate-400"
    >
      {{ reasoning }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { useLocalStorage } from '@vueuse/core';

defineProps<{
  reasoning?: string;
  streaming: boolean;
}>();

// Persist the user's expanded/collapsed preference across messages and reloads.
const expanded = useLocalStorage('bs.ai.thinkingExpanded', true);
</script>
