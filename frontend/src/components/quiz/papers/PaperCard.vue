<template>
  <article
    class="group relative flex flex-col justify-between rounded-2xl border border-slate-200/80 bg-white p-4 shadow-sm transition hover:border-violet-300 hover:shadow-md sm:p-5 dark:border-slate-800 dark:bg-slate-900 dark:hover:border-violet-700/60"
  >
    <div>
      <!-- Title & delete -->
      <div class="flex items-start justify-between gap-3">
        <h4
          class="min-w-0 text-base leading-snug font-bold break-words text-slate-900 transition group-hover:text-violet-600 dark:text-slate-100 dark:group-hover:text-violet-400"
        >
          {{ paper.title }}
        </h4>
        <button
          type="button"
          class="grid h-8 w-8 shrink-0 place-items-center rounded-lg text-slate-400 transition hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/40 dark:hover:text-rose-400"
          title="Delete paper"
          aria-label="Delete paper"
          @click="$emit('delete', paper)"
        >
          <Trash2 class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
      </div>

      <!-- Meta badges -->
      <div class="mt-3 flex flex-wrap items-center gap-1.5 text-[11px]">
        <span
          class="inline-flex items-center gap-1 rounded-md bg-violet-50 px-2 py-0.5 font-bold text-violet-700 dark:bg-violet-900/40 dark:text-violet-300"
        >
          <ListChecks class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          {{ paper.question_count }} Questions
        </span>
        <span
          class="inline-flex items-center gap-1 rounded-md bg-slate-100 px-2 py-0.5 font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300"
        >
          <Layers class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          {{ paper.sections.length }} Section{{ paper.sections.length === 1 ? '' : 's' }}
        </span>
        <span class="inline-flex items-center gap-1 text-slate-400">
          <Calendar class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          {{ formatShortDate(paper.created_at) }}
        </span>
      </div>

      <!-- Best attempt -->
      <div v-if="bestAttempt" class="mt-3">
        <div
          class="inline-flex items-center gap-2 rounded-xl border border-emerald-200/60 bg-emerald-50/70 px-2.5 py-1 text-xs font-semibold text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-300"
        >
          <Trophy class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
          <span>Best attempt: {{ bestAttempt.percentage }}%</span>
          <span class="text-[10px] text-emerald-600 tabular-nums dark:text-emerald-400">
            ({{ bestAttempt.score }}/{{ bestAttempt.maxScore }})
          </span>
        </div>
      </div>
    </div>

    <!-- Actions -->
    <div class="mt-5 flex items-center gap-2 border-t border-slate-100 pt-4 dark:border-slate-800">
      <button
        type="button"
        class="inline-flex flex-1 items-center justify-center gap-1.5 rounded-xl bg-gradient-to-r from-violet-600 to-indigo-600 px-3.5 py-2.5 text-xs font-bold text-white shadow-sm transition hover:from-violet-700 hover:to-indigo-700"
        @click="$emit('attempt', paper)"
      >
        <Play class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        Attempt Online
      </button>

      <button
        type="button"
        class="inline-flex items-center justify-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-xs font-bold text-slate-700 transition hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
        title="View answer key & questions"
        @click="$emit('open', paper.id)"
      >
        <Eye class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
        View
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Calendar, Eye, Layers, ListChecks, Play, Trash2, Trophy } from '@lucide/vue';
import type { QuestionPaper } from '../../../types';
import { getBestPaperAttempt } from '../composables/usePaperAttempt';
import { formatShortDate } from '../quizFormat';

const props = defineProps<{ paper: QuestionPaper }>();

defineEmits<{
  open: [id: number];
  attempt: [paper: QuestionPaper];
  delete: [paper: QuestionPaper];
}>();

const bestAttempt = computed(() => getBestPaperAttempt(props.paper.id));
</script>
