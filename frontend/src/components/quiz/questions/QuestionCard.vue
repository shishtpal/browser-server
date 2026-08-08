<template>
  <article
    class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-colors hover:border-violet-200 dark:border-slate-700 dark:bg-slate-800/60 dark:hover:border-violet-800/60"
  >
    <!-- Meta row: badges + actions -->
    <div class="flex flex-wrap items-start gap-1.5">
      <TypeBadge :type="question.type" />
      <DifficultyBadge :difficulty="question.difficulty" />
      <span
        v-for="chip in metaChips"
        :key="chip"
        class="rounded-md bg-slate-100 px-2 py-0.5 text-[10px] font-semibold text-slate-600 dark:bg-slate-700 dark:text-slate-300"
      >
        {{ chip }}
      </span>

      <div class="ml-auto flex shrink-0 gap-0.5">
        <button
          type="button"
          class="grid h-7 w-7 place-items-center rounded-lg text-slate-400 transition hover:bg-violet-50 hover:text-violet-600 dark:hover:bg-violet-900/30 dark:hover:text-violet-400"
          title="Edit question"
          aria-label="Edit question"
          @click="$emit('edit', question)"
        >
          <Pencil class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
        </button>
        <button
          type="button"
          class="grid h-7 w-7 place-items-center rounded-lg text-slate-400 transition hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-900/30 dark:hover:text-rose-400"
          title="Delete question"
          aria-label="Delete question"
          @click="$emit('delete', question.id)"
        >
          <Trash2 class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
        </button>
      </div>
    </div>

    <p class="mt-2 text-sm font-medium whitespace-pre-wrap text-slate-800 dark:text-slate-100">
      {{ question.question }}
    </p>

    <img
      v-if="imageSrc"
      :src="imageSrc"
      alt="Question image"
      class="mt-2 max-h-48 w-full rounded-lg border border-gray-200 object-contain sm:w-auto dark:border-slate-700"
    />

    <!-- Choice options -->
    <ul v-if="question.options?.length" class="mt-2 space-y-1">
      <li
        v-for="opt in question.options"
        :key="opt.index"
        class="flex items-center gap-2 rounded-md px-2 py-1 text-xs"
        :class="
          opt.correct
            ? 'bg-emerald-50 font-semibold text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
            : 'text-slate-600 dark:text-slate-400'
        "
      >
        <span class="font-bold">{{ optionLetter(opt.index) }}.</span>
        <span class="min-w-0 flex-1">{{ opt.text }}</span>
        <Check
          v-if="opt.correct"
          class="h-3.5 w-3.5 shrink-0 text-emerald-600 dark:text-emerald-400"
          :stroke-width="3"
          aria-label="Correct answer"
        />
      </li>
    </ul>

    <!-- Chronology answer -->
    <ol v-else-if="orderedItems.length" class="mt-2 space-y-1">
      <li
        v-for="item in orderedItems"
        :key="item.index"
        class="flex items-center gap-2 rounded-md px-2 py-1 text-xs text-slate-600 dark:text-slate-400"
      >
        <span
          class="grid h-5 w-5 shrink-0 place-items-center rounded bg-violet-100 text-[10px] font-black text-violet-700 dark:bg-violet-900/30 dark:text-violet-300"
        >
          {{ item.correct_order }}
        </span>
        {{ item.text }}
      </li>
    </ol>

    <!-- Free text answer -->
    <p v-else-if="question.expected_text" class="mt-2 text-xs text-slate-600 dark:text-slate-400">
      <span class="font-bold text-emerald-600 dark:text-emerald-400">Answer:</span>
      {{ question.expected_text }}
    </p>

    <!-- Collapsible explanation -->
    <div v-if="question.explanation" class="mt-2">
      <button
        type="button"
        class="inline-flex items-center gap-1 text-[11px] font-semibold text-slate-500 underline-offset-2 hover:text-violet-600 hover:underline dark:text-slate-400 dark:hover:text-violet-400"
        :aria-expanded="showExplanation"
        @click="showExplanation = !showExplanation"
      >
        <ChevronDown
          class="h-3.5 w-3.5 transition-transform"
          :class="{ 'rotate-180': showExplanation }"
          aria-hidden="true"
        />
        {{ showExplanation ? 'Hide explanation' : 'Show explanation' }}
      </button>
      <pre
        v-if="showExplanation"
        class="mt-1 rounded-lg bg-slate-50 p-2 text-xs whitespace-pre-wrap text-slate-600 dark:bg-slate-900/60 dark:text-slate-300"
        >{{ question.explanation }}</pre>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { Check, ChevronDown, Pencil, Trash2 } from '@lucide/vue';
import type { QuestionResponse } from '../../../types';
import { API_BASE } from '../../../lib/api';
import TypeBadge from '../ui/TypeBadge.vue';
import DifficultyBadge from '../ui/DifficultyBadge.vue';
import { optionLetter, orderedChronology, questionImageSrc } from '../quizFormat';

const props = defineProps<{ question: QuestionResponse }>();
defineEmits<{ edit: [question: QuestionResponse]; delete: [id: number] }>();

const showExplanation = ref(false);

const metaChips = computed(() =>
  [
    ...(props.question.tags ?? []),
    props.question.subject,
    props.question.topic,
    props.question.sub_topic,
  ].filter((v): v is string => Boolean(v)),
);

const imageSrc = computed(() => questionImageSrc(props.question.image_url, API_BASE));

const orderedItems = computed(() => orderedChronology(props.question.chronology_items));
</script>
