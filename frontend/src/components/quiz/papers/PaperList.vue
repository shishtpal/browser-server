<template>
  <div class="space-y-4 sm:space-y-6">
    <!-- Header + search -->
    <div
      class="flex flex-col gap-3 rounded-2xl border border-slate-200/80 bg-white p-4 shadow-sm sm:flex-row sm:flex-wrap sm:items-center sm:justify-between dark:border-slate-800 dark:bg-slate-900"
    >
      <div class="flex items-center gap-3">
        <span
          class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-violet-100 text-violet-700 dark:bg-violet-900/40 dark:text-violet-300"
        >
          <FileText class="h-5 w-5" :stroke-width="2.25" aria-hidden="true" />
        </span>
        <div>
          <h3 class="text-sm font-bold text-slate-900 dark:text-slate-100">
            Generated Question Papers
          </h3>
          <p class="text-xs text-slate-500 dark:text-slate-400">
            {{ papers.length }} paper{{ papers.length === 1 ? '' : 's' }} available for study &
            online examination
          </p>
        </div>
      </div>

      <div class="relative w-full sm:max-w-xs">
        <Search
          class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="searchQuery"
          type="search"
          placeholder="Search papers by title..."
          class="w-full rounded-xl border border-slate-300 bg-slate-50/80 py-2 pr-8 pl-9 text-xs text-slate-900 shadow-sm transition focus:border-violet-500 focus:bg-white focus:ring-2 focus:ring-violet-500/20 focus:outline-none dark:border-slate-700 dark:bg-slate-800/80 dark:text-slate-100 dark:focus:bg-slate-900"
        />
        <button
          v-if="searchQuery"
          type="button"
          class="absolute inset-y-0 right-0 flex items-center pr-2.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
          aria-label="Clear search"
          @click="searchQuery = ''"
        >
          <X class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        </button>
      </div>
    </div>

    <!-- Empty state -->
    <EmptyState
      v-if="filteredPapers.length === 0"
      :title="searchQuery ? 'No matching papers found' : 'No papers generated yet'"
      :description="
        searchQuery
          ? 'Try adjusting your search query or clear the filter.'
          : 'Generate a customized question paper from the Generate tab.'
      "
      icon="search"
      color="violet"
    />

    <!-- Grid -->
    <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <PaperCard
        v-for="paper in filteredPapers"
        :key="paper.id"
        :paper="paper"
        @open="$emit('open', $event)"
        @attempt="$emit('attempt', $event)"
        @delete="deletingPaper = $event"
      />
    </div>

    <PaperDeleteDialog
      :paper="deletingPaper"
      @cancel="deletingPaper = null"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { FileText, Search, X } from '@lucide/vue';
import type { QuestionPaper } from '../../../types';
import EmptyState from '../../ui/EmptyState.vue';
import PaperCard from './PaperCard.vue';
import PaperDeleteDialog from './PaperDeleteDialog.vue';

const props = defineProps<{ papers: QuestionPaper[] }>();

const emit = defineEmits<{
  open: [id: number];
  attempt: [paper: QuestionPaper];
  delete: [id: number];
}>();

const searchQuery = ref('');
const deletingPaper = ref<QuestionPaper | null>(null);

const filteredPapers = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  if (!q) return props.papers;
  return props.papers.filter((p) => p.title.toLowerCase().includes(q));
});

const confirmDelete = () => {
  if (deletingPaper.value) {
    emit('delete', deletingPaper.value.id);
    deletingPaper.value = null;
  }
};
</script>
