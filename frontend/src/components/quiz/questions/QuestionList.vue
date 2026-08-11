<template>
  <div class="space-y-4">
    <!-- Section header -->
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 class="flex items-center gap-2 text-lg font-black text-slate-800 dark:text-slate-100">
        Question Bank
        <span
          class="rounded-full bg-violet-100 px-2 py-0.5 text-xs font-bold text-violet-700 tabular-nums dark:bg-violet-900/40 dark:text-violet-300"
        >
          {{ questions.length }}
        </span>
      </h2>
      <div class="flex w-full items-center gap-2 sm:w-auto">
        <Button
          variant="ghost"
          size="sm"
          class="flex-1 sm:flex-none"
          :disabled="isRefreshing"
          @click="handleRefresh"
        >
          <span class="inline-flex items-center gap-1.5">
            <RefreshCw
              class="h-4 w-4"
              :class="{ 'animate-spin': isRefreshing }"
              :stroke-width="2.5"
              aria-hidden="true"
            />
            Refresh
          </span>
        </Button>
        <Button
          variant="gradient-violet"
          size="sm"
          class="flex-1 sm:flex-none"
          @click="$emit('add')"
        >
          <span class="inline-flex items-center gap-1.5">
            <Plus class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
            Add question
          </span>
        </Button>
      </div>
    </div>

    <QuestionFilters
      v-model:search-query="searchQueryProxy"
      v-model:filter-type="filterTypeProxy"
      v-model:filter-difficulty="filterDifficultyProxy"
      v-model:filter-tags="filterTagsProxy"
      v-model:filter-subject="filterSubjectProxy"
      :vocabulary="vocabulary"
      :has-active-filters="hasActiveFilters"
      @apply="$emit('applyFilters')"
      @clear="$emit('clearFilters')"
    />

    <EmptyState
      v-if="questions.length === 0"
      :title="hasActiveFilters ? 'No questions match your filters' : 'No questions yet'"
      :description="
        hasActiveFilters
          ? 'Try clearing or adjusting the filters above.'
          : 'Add your first question using the button above.'
      "
      icon="search"
      color="violet"
    />

    <template v-else>
      <div class="space-y-3">
        <QuestionCard
          v-for="question in paginatedQuestions"
          :key="question.id"
          :question="question"
          @edit="$emit('edit', $event)"
          @delete="$emit('delete', $event)"
        />
      </div>

      <QuestionPagination
        v-model:page="currentPage"
        :total="questions.length"
        :per-page="itemsPerPage"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { Plus, RefreshCw } from '@lucide/vue';
import type { QuestionResponse, TagVocabulary } from '../../../types';
import Button from '../../ui/Button.vue';
import EmptyState from '../../ui/EmptyState.vue';
import QuestionCard from './QuestionCard.vue';
import QuestionFilters from './QuestionFilters.vue';
import QuestionPagination from './QuestionPagination.vue';

const props = defineProps<{
  questions: QuestionResponse[];
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
  applyFilters: [];
  clearFilters: [];
  add: [];
  edit: [question: QuestionResponse];
  delete: [id: number];
  refresh: [];
}>();

/** Refresh with brief spinner feedback ---------------------------------- */

const isRefreshing = ref(false);

async function handleRefresh() {
  if (isRefreshing.value) return;
  isRefreshing.value = true;
  emit('refresh');
  // Keep the spinner visible for at least 600 ms so the animation is readable.
  setTimeout(() => {
    isRefreshing.value = false;
  }, 600);
}

/** v-model proxies ----------------------------------------------------- */

const searchQueryProxy = computed({
  get: () => props.searchQuery,
  set: (v: string) => emit('update:searchQuery', v),
});
const filterTypeProxy = computed({
  get: () => props.filterType,
  set: (v: string) => emit('update:filterType', v),
});
const filterDifficultyProxy = computed({
  get: () => props.filterDifficulty,
  set: (v: string) => emit('update:filterDifficulty', v),
});
const filterTagsProxy = computed({
  get: () => props.filterTags,
  set: (v: string[]) => emit('update:filterTags', v),
});
const filterSubjectProxy = computed({
  get: () => props.filterSubject,
  set: (v: string) => emit('update:filterSubject', v),
});

/** Pagination ----------------------------------------------------------- */

const itemsPerPage = 20;
const currentPage = ref(1);

watch(
  () => [
    props.searchQuery,
    props.filterType,
    props.filterDifficulty,
    props.filterTags,
    props.filterSubject,
  ],
  () => {
    currentPage.value = 1;
  },
  { deep: true },
);

const totalPages = computed(() => Math.max(1, Math.ceil(props.questions.length / itemsPerPage)));

// Clamp the page when deleting from the bank shrinks the result set.
watch(totalPages, (pages) => {
  if (currentPage.value > pages) currentPage.value = pages;
});

const paginatedQuestions = computed(() => {
  const start = (currentPage.value - 1) * itemsPerPage;
  return props.questions.slice(start, start + itemsPerPage);
});
</script>
