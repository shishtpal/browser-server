<template>
  <div class="flex items-center gap-1.5">
    <select
      :value="sortField"
      aria-label="Sort field"
      class="w-full rounded-lg border border-gray-300 bg-gray-50 px-2 py-1.5 text-[11px] font-black text-slate-700 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none sm:w-auto dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
      @change="
        $emit('update:sortField', ($event.target as HTMLSelectElement).value as TodoSortField)
      "
    >
      <option v-for="opt in SORT_OPTIONS" :key="opt.value" :value="opt.value">
        {{ opt.label }}
      </option>
    </select>
    <button
      type="button"
      :aria-label="`Sort direction: ${sortDir === 'asc' ? 'ascending' : 'descending'}`"
      :title="`Sort ${sortDir === 'asc' ? 'descending' : 'ascending'}`"
      class="grid h-[30px] w-[30px] place-items-center rounded-lg bg-gray-100 text-slate-500 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600"
      @click="$emit('toggle-dir')"
    >
      <ArrowUp
        v-if="sortDir === 'asc'"
        class="h-3.5 w-3.5"
        :stroke-width="2.5"
        aria-hidden="true"
      />
      <ArrowDown v-else class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
    </button>
  </div>
</template>

<script setup lang="ts">
import type { TodoSortField } from '../../types';
import { ArrowDown, ArrowUp } from '@lucide/vue';
import { SORT_OPTIONS } from './todoFormat';

defineProps<{
  sortField: TodoSortField;
  sortDir: 'asc' | 'desc';
}>();

defineEmits<{ 'update:sortField': [value: TodoSortField]; 'toggle-dir': [] }>();
</script>
