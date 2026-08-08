<template>
  <div
    class="mb-4 rounded-2xl border border-gray-200/80 bg-white/90 p-3 shadow-sm dark:border-slate-700/80 dark:bg-slate-800/90"
  >
    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <label class="relative min-w-0 flex-1">
        <span class="sr-only">Search todos</span>
        <Search
          class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          :value="modelValue"
          type="search"
          placeholder="Search titles, descriptions, or tags..."
          class="w-full rounded-xl border border-gray-300 bg-gray-50 py-2.5 pr-10 pl-10 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-900/50 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
          @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        />
        <button
          v-if="modelValue"
          type="button"
          class="absolute top-1/2 right-2 grid h-7 w-7 -translate-y-1/2 place-items-center rounded-lg text-slate-400 transition hover:bg-gray-200 hover:text-slate-700 dark:hover:bg-slate-700 dark:hover:text-slate-200"
          aria-label="Clear todo search"
          @click="$emit('update:modelValue', '')"
        >
          <X class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
        </button>
      </label>
      <p
        class="shrink-0 text-center text-xs font-bold text-slate-500 sm:text-left dark:text-slate-400"
        aria-live="polite"
      >
        {{ resultSummary }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Search, X } from '@lucide/vue';

const props = defineProps<{
  modelValue: string;
  resultCount: number;
}>();

defineEmits<{ 'update:modelValue': [value: string] }>();

const resultSummary = computed(() => {
  const count = props.resultCount;
  const noun = props.modelValue.trim() ? 'result' : 'todo';
  return `${count} ${noun}${count === 1 ? '' : 's'}`;
});
</script>
