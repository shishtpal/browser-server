<template>
  <div
    class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-colors dark:border-slate-700 dark:bg-slate-800/60"
  >
    <div class="flex flex-wrap items-start gap-2">
      <span
        class="rounded-md px-2 py-0.5 text-[10px] font-black uppercase tracking-wide"
        :class="typeBadge"
        >{{ typeLabel }}</span
      >
      <span
        class="rounded-md px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide"
        :class="difficultyBadge"
        >{{ question.difficulty }}</span
      >
      <span
        v-for="tag in question.tags ?? []"
        :key="tag"
        class="rounded-md bg-slate-100 px-2 py-0.5 text-[10px] font-semibold text-slate-600 dark:bg-slate-700 dark:text-slate-300"
        >{{ tag }}</span
      >
      <span
        v-if="question.subject"
        class="rounded-md bg-slate-100 px-2 py-0.5 text-[10px] font-semibold text-slate-600 dark:bg-slate-700 dark:text-slate-300"
        >{{ question.subject }}</span
      >
      <span
        v-if="question.topic"
        class="rounded-md bg-slate-100 px-2 py-0.5 text-[10px] font-semibold text-slate-600 dark:bg-slate-700 dark:text-slate-300"
        >{{ question.topic }}</span
      >
      <span
        v-if="question.sub_topic"
        class="rounded-md bg-slate-100 px-2 py-0.5 text-[10px] font-semibold text-slate-600 dark:bg-slate-700 dark:text-slate-300"
        >{{ question.sub_topic }}</span
      >
      <div class="ml-auto flex gap-1">
        <button
          type="button"
          class="rounded-md px-2 py-0.5 text-[11px] font-semibold text-indigo-600 transition hover:bg-indigo-50 dark:text-indigo-400 dark:hover:bg-indigo-900/30"
          @click="$emit('edit', question)"
        >
          Edit
        </button>
        <button
          type="button"
          class="rounded-md px-2 py-0.5 text-[11px] font-semibold text-rose-600 transition hover:bg-rose-50 dark:text-rose-400 dark:hover:bg-rose-900/30"
          @click="$emit('delete', question.id)"
        >
          Delete
        </button>
      </div>
    </div>

    <p class="mt-2 whitespace-pre-wrap text-sm font-medium text-slate-800 dark:text-slate-100">
      {{ question.question }}
    </p>

    <img
      v-if="question.image_url"
      :src="imageSrc"
      alt="Question image"
      class="mt-2 max-h-48 rounded-lg border border-gray-200 object-contain dark:border-slate-700"
    />

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
        <span class="font-bold">{{ String.fromCharCode(65 + opt.index) }}.</span> {{ opt.text }}
        <span v-if="opt.correct" class="ml-auto text-[10px] font-black">✓</span>
      </li>
    </ul>

    <ol v-else-if="question.chronology_items?.length" class="mt-2 space-y-1">
      <li
        v-for="item in orderedChronology"
        :key="item.index"
        class="flex items-center gap-2 rounded-md px-2 py-1 text-xs text-slate-600 dark:text-slate-400"
      >
        <span
          class="grid h-5 w-5 shrink-0 place-items-center rounded bg-indigo-100 text-[10px] font-black text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300"
          >{{ item.correct_order }}</span
        >
        {{ item.text }}
      </li>
    </ol>

    <p v-else-if="question.expected_text" class="mt-2 text-xs text-slate-600 dark:text-slate-400">
      <span class="font-bold text-emerald-600 dark:text-emerald-400">Answer:</span>
      {{ question.expected_text }}
    </p>

    <div v-if="question.explanation" class="mt-2">
      <button
        type="button"
        class="text-[11px] font-semibold text-slate-500 underline-offset-2 hover:underline dark:text-slate-400"
        @click="showExplanation = !showExplanation"
      >
        {{ showExplanation ? "Hide explanation" : "Show explanation" }}
      </button>
      <pre
        v-if="showExplanation"
        class="mt-1 whitespace-pre-wrap rounded-lg bg-slate-50 p-2 text-xs text-slate-600 dark:bg-slate-900/60 dark:text-slate-300"
        >{{ question.explanation }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { API_BASE } from "../../lib/api";
import type { QuestionResponse } from "../../types";

const props = defineProps<{ question: QuestionResponse }>();
defineEmits<{ edit: [question: QuestionResponse]; delete: [id: number] }>();

const showExplanation = ref(false);

const typeLabel = computed(() => props.question.type.replace("_", " "));
const typeBadge = computed(() => {
  const map: Record<string, string> = {
    single_choice: "bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300",
    multiple_choice: "bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300",
    input: "bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300",
    chronology: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300",
  };
  return map[props.question.type] ?? map.single_choice;
});
const difficultyBadge = computed(() => {
  const map: Record<string, string> = {
    easy: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300",
    medium: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300",
    hard: "bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300",
  };
  return map[props.question.difficulty] ?? map.medium;
});

const imageSrc = computed(() =>
  props.question.image_url?.startsWith("http")
    ? props.question.image_url
    : `${API_BASE}${props.question.image_url}`,
);

const orderedChronology = computed(() =>
  [...(props.question.chronology_items ?? [])].sort((a, b) => a.correct_order - b.correct_order),
);
</script>
