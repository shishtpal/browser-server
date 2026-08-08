<template>
  <div
    class="rounded-2xl border border-slate-200/80 bg-white p-4 shadow-sm sm:p-5 dark:border-slate-800 dark:bg-slate-900"
  >
    <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
      <h3 class="text-base font-bold text-slate-900 dark:text-slate-100">
        Detailed Question Review
      </h3>

      <!-- Segmented filter -->
      <div
        class="flex w-full scrollbar-none items-center gap-1 overflow-x-auto rounded-xl bg-slate-100 p-1 sm:w-auto dark:bg-slate-800"
        role="group"
        aria-label="Filter reviewed questions"
      >
        <button
          v-for="filter in filters"
          :key="filter.key"
          type="button"
          class="inline-flex flex-1 items-center justify-center gap-1 rounded-lg px-3 py-1.5 text-xs font-bold whitespace-nowrap transition sm:flex-none"
          :class="
            activeFilter === filter.key
              ? 'bg-white text-violet-700 shadow-sm dark:bg-slate-700 dark:text-violet-300'
              : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200'
          "
          :aria-pressed="activeFilter === filter.key"
          @click="activeFilter = filter.key"
        >
          <component :is="filter.icon" class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          {{ filter.label }}
        </button>
      </div>
    </div>

    <div v-if="filteredQuestions.length" class="space-y-3 sm:space-y-4">
      <ExamReviewItem
        v-for="q in filteredQuestions"
        :key="q.id"
        :question="q"
        :question-number="originalIndex(q.id) + 1"
        :result="resultFor(q.id)"
        :answer="answers[q.id]"
      />
    </div>
    <p v-else class="py-8 text-center text-sm text-slate-500 dark:text-slate-400">
      No questions match this filter.
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { CircleCheck, CircleX, Flag, ListChecks, type LucideIcon } from '@lucide/vue';
import type { QuestionPaper, QuestionResponse } from '../../../../types';
import type {
  PaperAttemptRecord,
  QuestionAttemptResult,
  UserAnswer,
} from '../../composables/usePaperAttempt';
import ExamReviewItem from './ExamReviewItem.vue';

const props = defineProps<{
  paper: QuestionPaper;
  record: PaperAttemptRecord;
  answers: Record<number, UserAnswer>;
  isFlagged: (qId: number) => boolean;
}>();

const filters: ReadonlyArray<{ key: string; label: string; icon: LucideIcon }> = [
  { key: 'all', label: 'All', icon: ListChecks },
  { key: 'incorrect', label: 'Incorrect', icon: CircleX },
  { key: 'correct', label: 'Correct', icon: CircleCheck },
  { key: 'flagged', label: 'Flagged', icon: Flag },
] as const;

type FilterKey = (typeof filters)[number]['key'];
const activeFilter = ref<FilterKey>('all');

const resultFor = (qId: number): QuestionAttemptResult | undefined =>
  props.record.results.find((r) => r.questionId === qId);

const originalIndex = (qId: number) => props.paper.questions?.findIndex((q) => q.id === qId) ?? 0;

const filteredQuestions = computed<QuestionResponse[]>(() => {
  const list = props.paper.questions || [];
  if (activeFilter.value === 'all') return list;
  return list.filter((q) => {
    const res = resultFor(q.id);
    if (activeFilter.value === 'correct') return Boolean(res?.isCorrect);
    if (activeFilter.value === 'incorrect') return !res?.isCorrect;
    if (activeFilter.value === 'flagged') return props.isFlagged(q.id);
    return true;
  });
});
</script>

<style scoped>
.scrollbar-none {
  scrollbar-width: none;
}
.scrollbar-none::-webkit-scrollbar {
  display: none;
}
</style>
