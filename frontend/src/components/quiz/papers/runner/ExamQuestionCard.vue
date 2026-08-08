<template>
  <div
    class="mx-auto max-w-3xl rounded-2xl border border-slate-200/80 bg-white p-4 shadow-sm sm:p-7 dark:border-slate-800 dark:bg-slate-900"
  >
    <!-- Badges & flag -->
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <div class="flex flex-wrap items-center gap-1.5">
        <TypeBadge :type="question.type" />
        <DifficultyBadge v-if="question.difficulty" :difficulty="question.difficulty" />
        <span
          v-if="question.subject"
          class="rounded-md bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-600 dark:bg-slate-800 dark:text-slate-300"
        >
          {{ question.subject }}
        </span>
      </div>

      <button
        type="button"
        class="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-bold transition"
        :class="
          flagged
            ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/50 dark:text-amber-200'
            : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'
        "
        :aria-pressed="flagged"
        @click="$emit('toggle-flag')"
      >
        <Flag
          class="h-3.5 w-3.5"
          :stroke-width="2.25"
          :fill="flagged ? 'currentColor' : 'none'"
          aria-hidden="true"
        />
        {{ flagged ? 'Flagged for review' : 'Mark for review' }}
      </button>
    </div>

    <!-- Question text -->
    <h3
      class="text-base leading-relaxed font-bold whitespace-pre-line text-slate-900 sm:text-lg dark:text-slate-100"
    >
      <span class="mr-2 font-black text-violet-600 tabular-nums dark:text-violet-400"
        >{{ questionNumber }}.</span
      >
      {{ question.question }}
    </h3>

    <!-- Optional image -->
    <div
      v-if="imageSrc"
      class="my-4 max-w-md overflow-hidden rounded-xl border border-slate-200 dark:border-slate-700"
    >
      <img :src="imageSrc" alt="Question image" class="max-h-64 w-full object-contain" />
    </div>

    <!-- Answer input per type -->
    <div class="mt-6">
      <ExamChoiceOptions
        v-if="question.type === 'single_choice'"
        :question="question"
        :model-value="answer?.singleChoice"
        @select="$emit('single-choice', $event)"
      />

      <ExamChoiceOptions
        v-else-if="question.type === 'multiple_choice'"
        :question="question"
        :selected-indexes="answer?.multipleChoice || []"
        multiple
        @select="$emit('multiple-choice', $event)"
      />

      <ExamInputAnswer
        v-else-if="question.type === 'input'"
        :model-value="answer?.inputText || ''"
        @update:model-value="$emit('input-text', $event)"
      />

      <ExamChronologyOrder
        v-else-if="question.type === 'chronology'"
        :question="question"
        :order="chronologyOrder"
        @move="(from, to) => $emit('chronology-move', from, to)"
      />
    </div>

    <!-- Navigation -->
    <div
      class="mt-8 flex items-center justify-between gap-3 border-t border-slate-100 pt-4 dark:border-slate-800"
    >
      <Button
        variant="secondary"
        size="sm"
        :disabled="isFirst"
        class="flex-1 sm:flex-none"
        @click="$emit('prev')"
      >
        <span class="inline-flex items-center gap-1.5">
          <ChevronLeft class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          Previous
        </span>
      </Button>
      <Button
        variant="gradient-violet"
        size="sm"
        :disabled="isLast"
        class="flex-1 sm:flex-none"
        @click="$emit('next')"
      >
        <span class="inline-flex items-center gap-1.5">
          Next
          <ChevronRight class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
        </span>
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ChevronLeft, ChevronRight, Flag } from '@lucide/vue';
import type { QuestionResponse } from '../../../../types';
import type { UserAnswer } from '../../composables/usePaperAttempt';
import { API_BASE } from '../../../../lib/api';
import Button from '../../../ui/Button.vue';
import TypeBadge from '../../ui/TypeBadge.vue';
import DifficultyBadge from '../../ui/DifficultyBadge.vue';
import ExamChoiceOptions from './ExamChoiceOptions.vue';
import ExamChronologyOrder from './ExamChronologyOrder.vue';
import ExamInputAnswer from './ExamInputAnswer.vue';
import { questionImageSrc } from '../../quizFormat';

const props = defineProps<{
  question: QuestionResponse;
  questionNumber: number;
  flagged: boolean;
  answer?: UserAnswer;
  isFirst: boolean;
  isLast: boolean;
}>();

defineEmits<{
  'toggle-flag': [];
  'single-choice': [index: number];
  'multiple-choice': [index: number];
  'input-text': [text: string];
  'chronology-move': [fromIndex: number, toIndex: number];
  prev: [];
  next: [];
}>();

const imageSrc = computed(() => questionImageSrc(props.question.image_url, API_BASE));

const chronologyOrder = computed(
  () =>
    props.answer?.chronologyOrder ?? (props.question.chronology_items || []).map((i) => i.index),
);
</script>
