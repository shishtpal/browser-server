<template>
  <div
    class="mb-4 rounded-2xl border border-gray-200/80 bg-white/90 p-3 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90"
  >
    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div class="grid flex-1 grid-cols-[auto_1fr] gap-2">
        <select
          :value="searchColumn"
          aria-label="Search column"
          class="shrink-0 rounded-xl border border-gray-300 bg-white px-2.5 py-2 text-xs font-black text-slate-700 shadow-sm transition focus:border-emerald-400 focus:ring-4 focus:ring-emerald-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:border-emerald-400 dark:focus:ring-emerald-900/30"
          @change="
            $emit(
              'update:searchColumn',
              ($event.target as HTMLSelectElement).value as WalletSearchColumn,
            )
          "
        >
          <option v-for="col in WALLET_SEARCH_COLUMNS" :key="col.value" :value="col.value">
            {{ col.label }}
          </option>
        </select>

        <label class="relative">
          <span class="sr-only">Search wallet</span>
          <Search
            class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-400"
            aria-hidden="true"
          />
          <input
            :value="searchQuery"
            type="search"
            :placeholder="placeholder"
            class="w-full rounded-xl border border-gray-300 bg-white py-2 pr-9 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-400 focus:ring-4 focus:ring-emerald-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-emerald-900/30"
            @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
          />
          <button
            v-if="searchQuery"
            type="button"
            class="absolute top-1/2 right-1.5 grid h-7 w-7 -translate-y-1/2 place-items-center rounded-lg text-slate-400 transition hover:bg-gray-200 hover:text-slate-700 dark:hover:bg-slate-700 dark:hover:text-slate-200"
            aria-label="Clear wallet search"
            @click="$emit('update:searchQuery', '')"
          >
            <X class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          </button>
        </label>
      </div>

      <span
        class="shrink-0 text-center text-xs font-semibold text-slate-500 tabular-nums sm:text-left dark:text-slate-400"
        aria-live="polite"
      >
        {{ filteredCount }} of {{ totalCount }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Search, X } from '@lucide/vue';
import { WALLET_SEARCH_COLUMNS, type WalletSearchColumn } from './walletFormat';

defineProps<{
  searchQuery: string;
  searchColumn: WalletSearchColumn;
  placeholder: string;
  filteredCount: number;
  totalCount: number;
}>();

defineEmits<{
  'update:searchQuery': [value: string];
  'update:searchColumn': [value: WalletSearchColumn];
}>();
</script>
