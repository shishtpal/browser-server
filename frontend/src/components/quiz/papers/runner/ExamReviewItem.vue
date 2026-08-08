<template>
  <article class="rounded-xl border p-3.5 transition sm:p-4" :class="borderClass">
    <div class="mb-2 flex items-center justify-between gap-2">
      <span class="text-xs font-bold text-slate-500 tabular-nums dark:text-slate-400">
        Question {{ questionNumber }}
      </span>
      <span
        class="inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[10px] font-black tracking-wider uppercase"
        :class="badgeClass"
      >
        <component
          :is="result?.isCorrect ? CircleCheck : CircleX"
          class="h-3 w-3"
          :stroke-width="3"
          aria-hidden="true"
        />
        {{ result?.isCorrect ? 'Correct (+1)' : 'Incorrect (0)' }}
      </span>
    </div>

    <p class="text-sm font-semibold whitespace-pre-wrap text-slate-900 dark:text-slate-100">
      {{ question.question }}
    </p>

    <!-- Answer comparison -->
    <div class="mt-3 space-y-2 text-xs">
      <div
        class="rounded-lg p-2.5 font-medium break-words"
        :class="
          result?.isCorrect
            ? 'bg-emerald-50 text-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200'
            : 'bg-rose-50 text-rose-900 dark:bg-rose-950/40 dark:text-rose-200'
        "
      >
        <span class="font-bold">Your answer:</span>
        {{ userAnswerText }}
      </div>

      <div
        v-if="!result?.isCorrect"
        class="rounded-lg bg-emerald-50/70 p-2.5 font-medium break-words text-emerald-900 dark:bg-emerald-950/30 dark:text-emerald-300"
      >
        <span class="font-bold">Correct answer:</span>
        {{ result?.expectedAnswerText }}
      </div>
    </div>

    <!-- Explanation -->
    <div
      v-if="question.explanation"
      class="mt-3 rounded-lg bg-slate-50 p-3 text-xs text-slate-600 dark:bg-slate-800/60 dark:text-slate-300"
    >
      <span class="mb-1 flex items-center gap-1 font-bold text-violet-600 dark:text-violet-400">
        <Lightbulb class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
        Explanation
      </span>
      <p class="whitespace-pre-wrap">{{ question.explanation }}</p>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { CircleCheck, CircleX, Lightbulb } from '@lucide/vue';
import type { QuestionResponse } from '../../../../types';
import type { QuestionAttemptResult, UserAnswer } from '../../composables/usePaperAttempt';
import { formatUserAnswerText } from '../../quizFormat';

const props = defineProps<{
  question: QuestionResponse;
  questionNumber: number;
  result?: QuestionAttemptResult;
  answer?: UserAnswer;
}>();

const userAnswerText = computed(() => formatUserAnswerText(props.question, props.answer));

const borderClass = computed(() => {
  if (!props.result) return 'border-slate-200 dark:border-slate-800';
  return props.result.isCorrect
    ? 'border-emerald-200 bg-emerald-50/20 dark:border-emerald-900/40 dark:bg-emerald-950/10'
    : 'border-rose-200 bg-rose-50/20 dark:border-rose-900/40 dark:bg-rose-950/10';
});

const badgeClass = computed(() =>
  props.result?.isCorrect
    ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-300'
    : 'bg-rose-100 text-rose-800 dark:bg-rose-900/50 dark:text-rose-300',
);
</script>
