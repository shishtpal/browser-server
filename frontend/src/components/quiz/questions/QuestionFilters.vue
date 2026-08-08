<template>
  <div
    class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
  >
    <div class="grid grid-cols-2 items-center gap-2 sm:grid-cols-3 lg:flex lg:flex-wrap">
      <!-- Search (always full width on the first row) -->
      <div class="relative col-span-2 sm:col-span-3 lg:w-64">
        <Search
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          :value="searchQuery"
          type="search"
          placeholder="Search questions…"
          class="w-full rounded-lg border border-gray-300 bg-white py-2 pr-3 pl-9 text-xs transition focus:border-violet-400 focus:ring-2 focus:ring-violet-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-violet-900/30"
          @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
          @keyup.enter="$emit('apply')"
        />
      </div>

      <select
        :value="filterType"
        class="w-full rounded-lg border border-gray-300 bg-white px-2 py-2 text-xs lg:w-auto dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        @change="
          $emit('update:filterType', ($event.target as HTMLSelectElement).value);
          $emit('apply');
        "
      >
        <option value="">All types</option>
        <option value="single_choice">Single choice</option>
        <option value="multiple_choice">Multiple choice</option>
        <option value="input">Free text</option>
        <option value="chronology">Chronology</option>
      </select>

      <select
        :value="filterDifficulty"
        class="w-full rounded-lg border border-gray-300 bg-white px-2 py-2 text-xs lg:w-auto dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        @change="
          $emit('update:filterDifficulty', ($event.target as HTMLSelectElement).value);
          $emit('apply');
        "
      >
        <option value="">All difficulties</option>
        <option value="easy">Easy</option>
        <option value="medium">Medium</option>
        <option value="hard">Hard</option>
      </select>

      <QuestionTagPicker
        v-model="tagsProxy"
        :available-tags="availableTags"
        class="col-span-2 sm:col-span-1 lg:w-auto"
        @apply="$emit('apply')"
      />

      <select
        v-if="subjects.length"
        :value="filterSubject"
        class="w-full rounded-lg border border-gray-300 bg-white px-2 py-2 text-xs lg:w-auto dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        @change="
          $emit('update:filterSubject', ($event.target as HTMLSelectElement).value);
          $emit('apply');
        "
      >
        <option value="">All subjects</option>
        <option v-for="subject in subjects" :key="subject" :value="subject">
          {{ subject }}
        </option>
      </select>

      <div class="col-span-2 flex gap-2 sm:col-span-3 lg:col-span-1 lg:ml-auto">
        <button
          v-if="hasActiveFilters"
          type="button"
          class="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-xs font-bold text-slate-500 transition hover:bg-slate-100 hover:text-slate-700 lg:flex-none dark:text-slate-400 dark:hover:bg-slate-700 dark:hover:text-slate-200"
          @click="
            $emit('clear');
            $emit('apply');
          "
        >
          <X class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Clear
        </button>
        <button
          type="button"
          class="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-slate-900 px-4 py-2 text-xs font-bold text-white transition hover:bg-slate-700 lg:flex-none dark:bg-white dark:text-slate-900 dark:hover:bg-slate-200"
          @click="$emit('apply')"
        >
          <Search class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Search
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Search, X } from '@lucide/vue';
import type { TagVocabulary } from '../../../types';
import QuestionTagPicker from './QuestionTagPicker.vue';

const props = defineProps<{
  vocabulary?: TagVocabulary | null;
  searchQuery: string;
  filterType: string;
  filterDifficulty: string;
  filterTags: string[];
  filterSubject: string;
  hasActiveFilters: boolean;
}>();

const emit = defineEmits<{
  'update:searchQuery': [value: string];
  'update:filterType': [value: string];
  'update:filterDifficulty': [value: string];
  'update:filterTags': [value: string[]];
  'update:filterSubject': [value: string];
  apply: [];
  clear: [];
}>();

const availableTags = computed(() => props.vocabulary?.tags ?? []);
const subjects = computed(() => props.vocabulary?.subjects ?? []);

const tagsProxy = computed({
  get: () => props.filterTags,
  set: (value: string[]) => emit('update:filterTags', value),
});
</script>
