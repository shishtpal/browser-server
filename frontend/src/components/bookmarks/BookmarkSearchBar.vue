<template>
  <div
    class="mb-4 rounded-2xl border border-gray-200/80 bg-white/90 p-3 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90"
  >
    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div class="grid flex-1 grid-cols-[auto_1fr] gap-2">
        <!-- Search column -->
        <select
          :value="searchColumn"
          aria-label="Search column"
          class="shrink-0 rounded-xl border border-gray-300 bg-white px-2.5 py-2 text-xs font-black text-slate-700 shadow-sm transition focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:border-cyan-400 dark:focus:ring-cyan-900/30"
          @change="
            $emit(
              'update:searchColumn',
              ($event.target as HTMLSelectElement).value as BookmarkSearchColumn,
            )
          "
        >
          <option v-for="col in SEARCH_COLUMNS" :key="col.value" :value="col.value">
            {{ col.label }}
          </option>
        </select>

        <!-- Query input -->
        <div class="relative">
          <Search
            class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-400"
            aria-hidden="true"
          />
          <input
            :value="searchQuery"
            type="search"
            :placeholder="placeholder"
            class="w-full rounded-xl border border-gray-300 bg-white py-2 pr-9 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
            @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
          />
          <button
            v-if="searchQuery"
            type="button"
            class="absolute top-1/2 right-1.5 grid h-7 w-7 -translate-y-1/2 place-items-center rounded-lg text-slate-400 transition hover:bg-gray-200 hover:text-slate-700 dark:hover:bg-slate-700 dark:hover:text-slate-200"
            aria-label="Clear search"
            @click="$emit('update:searchQuery', '')"
          >
            <X class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          </button>
        </div>
      </div>

      <div class="flex items-center justify-between gap-2 sm:justify-end">
        <!-- View toggle -->
        <div
          class="flex items-stretch overflow-hidden rounded-lg border border-gray-300 text-[11px] font-black shadow-sm dark:border-slate-600"
          role="group"
          aria-label="Bookmark view"
        >
          <button
            v-for="option in viewOptions"
            :key="option.value"
            type="button"
            class="inline-flex items-center gap-1.5 px-2.5 py-1.5 transition"
            :class="
              viewMode === option.value
                ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900'
                : 'bg-white text-slate-600 hover:bg-gray-100 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700'
            "
            :aria-pressed="viewMode === option.value"
            @click="$emit('update:viewMode', option.value)"
          >
            <component
              :is="option.icon"
              class="h-3.5 w-3.5"
              :stroke-width="2.5"
              aria-hidden="true"
            />
            {{ option.label }}
          </button>
        </div>

        <span
          class="shrink-0 text-xs font-semibold text-slate-500 tabular-nums dark:text-slate-400"
          aria-live="polite"
        >
          {{ displayCount }} of {{ totalCount }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { List, ListTree, Search, X, type LucideIcon } from '@lucide/vue';
import { SEARCH_COLUMNS, SEARCH_PLACEHOLDERS, type BookmarkSearchColumn } from './bookmarkFormat';
import type { BookmarkViewMode } from './composables/useBookmarkTree';

const props = defineProps<{
  searchQuery: string;
  searchColumn: BookmarkSearchColumn;
  viewMode: BookmarkViewMode;
  filteredCount: number;
  treeCount: number;
  totalCount: number;
}>();

defineEmits<{
  'update:searchQuery': [value: string];
  'update:searchColumn': [value: BookmarkSearchColumn];
  'update:viewMode': [value: BookmarkViewMode];
}>();

const viewOptions: { value: BookmarkViewMode; label: string; icon: LucideIcon }[] = [
  { value: 'flat', label: 'List', icon: List },
  { value: 'tree', label: 'Tree', icon: ListTree },
];

const displayCount = computed(() =>
  props.viewMode === 'tree' ? props.treeCount : props.filteredCount,
);

const placeholder = computed(() => SEARCH_PLACEHOLDERS[props.searchColumn]);
</script>
