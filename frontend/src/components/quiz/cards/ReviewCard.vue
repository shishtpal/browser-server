<template>
  <article
    class="overflow-hidden rounded-2xl border border-slate-200/80 bg-white shadow-lg shadow-slate-200/40 dark:border-slate-700/80 dark:bg-slate-800 dark:shadow-none"
  >
    <div class="p-4 sm:p-5">
      <!-- Meta -->
      <header class="flex flex-wrap items-center gap-1.5">
        <TypeBadge :type="question.type" />

        <span
          v-for="chip in metaChips"
          :key="chip"
          class="rounded-md bg-slate-100 px-2 py-1 text-[10px] font-semibold text-slate-500 dark:bg-slate-700/70 dark:text-slate-300"
        >
          {{ chip }}
        </span>

        <div class="ml-auto flex items-center gap-2">
          <button
            type="button"
            class="grid h-7 w-7 place-items-center rounded-lg text-slate-400 transition hover:bg-violet-50 hover:text-violet-600 dark:hover:bg-violet-900/30 dark:hover:text-violet-400"
            :title="copied ? 'Copied!' : 'Copy question as Markdown'"
            :aria-label="copied ? 'Copied' : 'Copy question as Markdown'"
            @click="copyMarkdown"
          >
            <Check
              v-if="copied"
              class="h-3.5 w-3.5 text-emerald-500"
              :stroke-width="2.5"
              aria-hidden="true"
            />
            <Copy v-else class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
          </button>

          <label
            class="flex items-center gap-1.5 text-[11px] font-semibold text-slate-500 dark:text-slate-400"
          >
            Difficulty
            <select
              :value="question.difficulty"
              :disabled="isSavingDifficulty"
              class="cursor-pointer rounded-md border border-slate-200 bg-slate-50 px-2 py-1.5 text-[11px] font-bold text-slate-700 transition outline-none focus:border-violet-400 focus:ring-2 focus:ring-violet-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-200 dark:focus:ring-violet-500/20"
              @change="$emit('difficulty-change', ($event.target as HTMLSelectElement).value)"
            >
              <option value="easy">Easy</option>
              <option value="medium">Medium</option>
              <option value="hard">Hard</option>
            </select>
          </label>
        </div>
      </header>

      <!-- Question -->
      <p
        class="mt-4 text-base leading-relaxed font-bold whitespace-pre-wrap text-slate-900 sm:text-lg dark:text-slate-50"
      >
        {{ question.question }}
      </p>

      <img
        v-if="imageSrc"
        :src="imageSrc"
        alt="Question image"
        class="mt-4 max-h-72 w-full rounded-xl border border-slate-200 bg-slate-50 object-contain dark:border-slate-700 dark:bg-slate-900"
      />

      <!-- Choice options (answering reveals immediately) -->
      <ul v-if="question.options?.length" class="mt-5 grid gap-2" aria-label="Answer options">
        <li v-for="option in question.options" :key="option.index">
          <button
            type="button"
            :disabled="revealed"
            :aria-pressed="selectedOptionIndex === option.index"
            class="group flex w-full items-center gap-3 rounded-xl border px-3 py-3 text-left text-sm font-medium transition-all duration-200"
            :class="optionClass(option)"
            @click="chooseOption(option.index)"
          >
            <span
              class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border text-xs font-black transition-colors"
              :class="optionBadgeClass(option)"
            >
              {{ optionLetter(option.index) }}
            </span>

            <span class="flex-1 leading-snug">{{ option.text }}</span>

            <span
              v-if="revealed && option.correct"
              class="flex shrink-0 items-center gap-1 text-xs font-bold text-emerald-700 dark:text-emerald-300"
            >
              <CircleCheck class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
              Correct
            </span>

            <span
              v-else-if="revealed && selectedOptionIndex === option.index"
              class="flex shrink-0 items-center gap-1 text-xs font-bold text-rose-700 dark:text-rose-300"
            >
              <CircleX class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
              Incorrect
            </span>
          </button>
        </li>
      </ul>

      <!-- Chronology -->
      <ol v-else-if="chronology.length" class="mt-5 space-y-2">
        <li
          v-for="(item, index) in chronology"
          :key="item.index"
          class="flex gap-3 rounded-xl border border-slate-100 bg-slate-50 px-3 py-2.5 text-sm text-slate-700 dark:border-slate-700/70 dark:bg-slate-900/50 dark:text-slate-300"
        >
          <strong
            class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-slate-200 text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200"
          >
            {{ revealed ? item.correct_order : index + 1 }}
          </strong>
          <span>{{ item.text }}</span>
        </li>
      </ol>

      <!-- Reveal for non-choice questions -->
      <Button
        v-if="!revealed && !question.options?.length"
        class="mt-5 w-full sm:w-auto"
        variant="gradient-violet"
        @click="$emit('reveal')"
      >
        <span class="inline-flex items-center gap-1.5">
          <Eye class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          Show answer
        </span>
      </Button>

      <!-- Answer details -->
      <div v-if="revealed" class="mt-5 space-y-3">
        <div
          v-if="selectedOptionIndex !== null && question.options?.length"
          class="flex items-center gap-2 rounded-xl border px-3 py-2.5 text-sm font-semibold"
          :class="
            selectedOptionCorrect
              ? 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-800/70 dark:bg-emerald-950/30 dark:text-emerald-200'
              : 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-800/70 dark:bg-rose-950/30 dark:text-rose-200'
          "
        >
          <component
            :is="selectedOptionCorrect ? CircleCheck : CircleX"
            class="h-4 w-4 shrink-0"
            :stroke-width="2.5"
            aria-hidden="true"
          />
          {{
            selectedOptionCorrect
              ? "Nice work — that's correct."
              : 'Not quite — review the correct answer below.'
          }}
        </div>

        <p
          v-if="question.expected_text"
          class="rounded-xl border border-emerald-100 bg-emerald-50 px-3 py-3 text-sm leading-relaxed text-emerald-800 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-200"
        >
          <strong class="mr-1">Expected answer:</strong>
          {{ question.expected_text }}
        </p>

        <!-- Ask AI: explain / cross-check (self-hides when AI is disabled) -->
        <CardAIAssistant :question="question" />

        <p
          class="rounded-xl border border-slate-100 bg-slate-50 px-3 py-3 text-sm leading-relaxed whitespace-pre-wrap text-slate-600 dark:border-slate-700/70 dark:bg-slate-900/50 dark:text-slate-300"
        >
          <strong class="text-slate-800 dark:text-slate-100">Explanation:</strong>
          <span v-if="question.explanation?.trim()">{{ question.explanation }}</span>
          <span v-else class="text-slate-400 italic dark:text-slate-500">
            No explanation provided.
          </span>
        </p>

        <!-- Recall rating -->
        <section
          v-if="!practiceMode"
          aria-labelledby="recall-rating"
          class="border-t border-slate-100 pt-4 dark:border-slate-700"
        >
          <div class="flex items-baseline justify-between gap-3">
            <h3
              id="recall-rating"
              class="text-sm font-extrabold text-slate-900 dark:text-slate-100"
            >
              How well did you remember?
            </h3>
            <span
              class="hidden items-center gap-1 text-[10px] font-medium text-slate-400 sm:inline-flex"
            >
              <Keyboard class="h-3.5 w-3.5" aria-hidden="true" />
              Use numpad 0–3
            </span>
          </div>

          <div class="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
            <Button
              v-for="rating in ratings"
              :key="rating.name"
              :disabled="isRating"
              :variant="rating.name === 'again' ? 'danger' : 'secondary'"
              class="relative capitalize"
              @click="submitRating(rating.name)"
            >
              {{ rating.label }}
              <kbd
                class="ml-1.5 rounded border border-current/20 bg-black/5 px-1 py-0.5 text-[9px] font-bold opacity-70 dark:bg-white/10"
              >
                {{ rating.shortcut }}
              </kbd>
            </Button>
          </div>
        </section>

        <Button
          v-if="practiceMode"
          class="w-full sm:w-auto"
          variant="gradient-violet"
          @click="$emit('next')"
        >
          <span class="inline-flex items-center gap-1.5">
            Next card
            <ArrowRight class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          </span>
        </Button>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useEventListener } from '@vueuse/core';
import { ArrowRight, Check, CircleCheck, CircleX, Copy, Eye, Keyboard } from '@lucide/vue';
import type { QuestionResponse, ReviewRating } from '../../../types';
import { API_BASE } from '../../../lib/api';
import { getToken } from '../../../lib/auth';
import { copyToClipboard } from '../../../utils/copyToClipboard';
import Button from '../../ui/Button.vue';
import TypeBadge from '../ui/TypeBadge.vue';
import CardAIAssistant from './CardAIAssistant.vue';
import {
  optionLetter,
  orderedChronology,
  questionImageSrc,
  questionToMarkdown,
} from '../quizFormat';

const props = defineProps<{
  question: QuestionResponse;
  revealed: boolean;
  isRating: boolean;
  isSavingDifficulty: boolean;
  practiceMode: boolean;
}>();

const emit = defineEmits<{
  reveal: [];
  rate: [rating: ReviewRating];
  next: [];
  'difficulty-change': [difficulty: string];
}>();

const selectedOptionIndex = ref<number | null>(null);

/* --------------------------- copy as Markdown --------------------------- */

const copied = ref(false);
let copyTimer: ReturnType<typeof setTimeout> | undefined;

async function copyMarkdown() {
  try {
    await copyToClipboard(questionToMarkdown(props.question));
  } catch {
    return; // clipboard unavailable
  }
  copied.value = true;
  clearTimeout(copyTimer);
  copyTimer = setTimeout(() => {
    copied.value = false;
  }, 2000);
}

const ratings: Array<{ name: ReviewRating; label: string; shortcut: string; code: string }> = [
  { name: 'again', label: 'Again', shortcut: '0', code: 'Numpad0' },
  { name: 'hard', label: 'Hard', shortcut: '1', code: 'Numpad1' },
  { name: 'good', label: 'Good', shortcut: '2', code: 'Numpad2' },
  { name: 'easy', label: 'Easy', shortcut: '3', code: 'Numpad3' },
];

const metaChips = computed(() =>
  [
    ...(props.question.tags ?? []),
    props.question.subject,
    props.question.topic,
    props.question.sub_topic,
  ].filter((v): v is string => Boolean(v)),
);

const chronology = computed(() =>
  props.revealed
    ? orderedChronology(props.question.chronology_items)
    : (props.question.chronology_items ?? []),
);

const imageSrc = computed(() => questionImageSrc(props.question.image_url, API_BASE, getToken()));

const selectedOptionCorrect = computed(() => {
  if (selectedOptionIndex.value === null) return false;
  return Boolean(
    props.question.options?.find((o) => o.index === selectedOptionIndex.value)?.correct,
  );
});

function chooseOption(index: number) {
  if (props.revealed) return;
  selectedOptionIndex.value = index;
  emit('reveal');
}

function submitRating(rating: ReviewRating) {
  if (props.isRating) return;
  emit('rate', rating);
}

function optionClass(option: { index: number; correct?: boolean }) {
  const selected = selectedOptionIndex.value === option.index;

  if (props.revealed && option.correct) {
    return 'cursor-default border-emerald-300 bg-emerald-50 text-emerald-900 dark:border-emerald-700 dark:bg-emerald-950/35 dark:text-emerald-100';
  }
  if (props.revealed && selected) {
    return 'cursor-default border-rose-300 bg-rose-50 text-rose-900 dark:border-rose-700 dark:bg-rose-950/35 dark:text-rose-100';
  }
  if (props.revealed) {
    return 'cursor-default border-slate-100 bg-slate-50/60 text-slate-400 dark:border-slate-700/60 dark:bg-slate-900/20 dark:text-slate-500';
  }
  return 'border-slate-200 bg-white text-slate-700 hover:-translate-y-px hover:border-violet-300 hover:bg-violet-50/50 hover:shadow-sm focus:outline-none focus:ring-2 focus:ring-violet-400 focus:ring-offset-2 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-violet-500 dark:hover:bg-violet-500/10 dark:focus:ring-offset-slate-800';
}

function optionBadgeClass(option: { index: number; correct?: boolean }) {
  const selected = selectedOptionIndex.value === option.index;

  if (props.revealed && option.correct) {
    return 'border-emerald-300 bg-emerald-100 text-emerald-700 dark:border-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300';
  }
  if (props.revealed && selected) {
    return 'border-rose-300 bg-rose-100 text-rose-700 dark:border-rose-700 dark:bg-rose-900/50 dark:text-rose-300';
  }
  return 'border-slate-200 bg-slate-50 text-slate-500 group-hover:border-violet-300 group-hover:bg-violet-100 group-hover:text-violet-700 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-300 dark:group-hover:border-violet-500 dark:group-hover:bg-violet-500/20 dark:group-hover:text-violet-300';
}

/* --------------------------- keyboard shortcuts --------------------------- */

function isTypingTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false;
  return target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName);
}

useEventListener(window, 'keydown', (event: KeyboardEvent) => {
  if (!props.revealed || props.isRating || isTypingTarget(event.target) || event.repeat) return;
  const rating = ratings.find((item) => item.code === event.code);
  if (!rating) return;
  event.preventDefault();
  submitRating(rating.name);
});

watch(
  () => props.question,
  () => {
    selectedOptionIndex.value = null;
  },
);

watch(
  () => props.revealed,
  (revealed) => {
    if (!revealed) selectedOptionIndex.value = null;
  },
);
</script>
