<template>
  <div
    class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
  >
    <h3
      class="mb-3 flex items-center gap-1.5 text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400"
    >
      <History class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
      Recent papers
    </h3>

    <EmptyState
      v-if="recentPapers.length === 0"
      title="No papers yet"
      description="Generate a paper from the Generate tab."
      icon="search"
      color="violet"
    />

    <div v-else class="space-y-2">
      <button
        v-for="paper in recentPapers"
        :key="paper.id"
        type="button"
        class="group flex w-full items-center gap-3 rounded-lg border border-gray-100 p-3 text-left transition hover:border-violet-300 hover:bg-violet-50/50 dark:border-slate-700 dark:hover:border-violet-700 dark:hover:bg-violet-900/10"
        @click="$emit('open', paper.id)"
      >
        <span
          class="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-violet-100 text-violet-600 dark:bg-violet-900/40 dark:text-violet-300"
        >
          <FileText class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block truncate text-xs font-bold text-slate-700 dark:text-slate-200">
            {{ paper.title }}
          </span>
          <span class="mt-0.5 block text-[11px] font-medium text-slate-400">
            {{ paper.question_count }} questions · {{ formatShortDate(paper.created_at) }}
          </span>
        </span>
        <ChevronRight
          class="h-4 w-4 shrink-0 text-slate-300 transition group-hover:translate-x-0.5 group-hover:text-violet-500 dark:text-slate-600"
          :stroke-width="2.5"
          aria-hidden="true"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ChevronRight, FileText, History } from '@lucide/vue';
import type { QuestionPaper } from '../../../types';
import EmptyState from '../../ui/EmptyState.vue';
import { formatShortDate } from '../quizFormat';

const props = withDefaults(defineProps<{ papers: QuestionPaper[]; limit?: number }>(), {
  limit: 5,
});

defineEmits<{ open: [id: number] }>();

const recentPapers = computed(() => props.papers.slice(0, props.limit));
</script>
