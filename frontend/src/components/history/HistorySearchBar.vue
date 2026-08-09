<template>
  <div
    class="mb-4 rounded-2xl border border-gray-200/80 bg-white/90 p-3 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90"
  >
    <div class="flex items-center gap-2">
      <label class="relative min-w-0 flex-1">
        <span class="sr-only">Filter history</span>
        <Search
          class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          :value="modelValue"
          type="search"
          placeholder="Filter by URL or title..."
          class="w-full rounded-xl border border-gray-300 bg-gray-50 py-2 pr-9 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:ring-4 focus:ring-violet-100 focus:outline-none dark:border-slate-600 dark:bg-slate-900/50 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30"
          @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        />
        <button
          v-if="modelValue"
          type="button"
          class="absolute top-1/2 right-1.5 grid h-7 w-7 -translate-y-1/2 place-items-center rounded-lg text-slate-400 transition hover:bg-gray-200 hover:text-slate-700 dark:hover:bg-slate-700 dark:hover:text-slate-200"
          aria-label="Clear history filter"
          @click="$emit('update:modelValue', '')"
        >
          <X class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
        </button>
      </label>

      <span
        class="shrink-0 text-xs font-semibold text-slate-500 tabular-nums dark:text-slate-400"
        aria-live="polite"
      >
        {{ filteredCount }} of {{ totalCount }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Search, X } from '@lucide/vue';

defineProps<{
  modelValue: string;
  filteredCount: number;
  totalCount: number;
}>();

defineEmits<{ 'update:modelValue': [value: string] }>();
</script>
