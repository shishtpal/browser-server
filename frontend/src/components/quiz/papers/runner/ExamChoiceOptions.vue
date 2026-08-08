<template>
  <div class="space-y-2.5">
    <p v-if="multiple" class="mb-2 text-xs font-semibold text-slate-500 dark:text-slate-400">
      Select all options that apply:
    </p>
    <button
      v-for="opt in question.options || []"
      :key="opt.index"
      type="button"
      class="flex w-full cursor-pointer items-center gap-3 rounded-xl border p-3.5 text-left text-sm font-medium transition"
      :class="
        isSelected(opt.index)
          ? 'border-violet-600 bg-violet-50/80 text-violet-900 shadow-sm dark:border-violet-500 dark:bg-violet-950/40 dark:text-violet-200'
          : 'border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-900/60 dark:text-slate-200 dark:hover:border-slate-700 dark:hover:bg-slate-800/80'
      "
      :aria-pressed="isSelected(opt.index)"
      @click="$emit('select', opt.index)"
    >
      <!-- Multi: checkbox -->
      <span
        v-if="multiple"
        class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md border transition"
        :class="
          isSelected(opt.index)
            ? 'border-violet-600 bg-violet-600 text-white'
            : 'border-slate-300 bg-white text-transparent dark:border-slate-600 dark:bg-slate-800'
        "
      >
        <Check class="h-3.5 w-3.5" :stroke-width="3" aria-hidden="true" />
      </span>
      <!-- Single: letter disc -->
      <span
        v-else
        class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-xs font-bold transition"
        :class="
          isSelected(opt.index)
            ? 'bg-violet-600 text-white'
            : 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300'
        "
      >
        {{ optionLetter(opt.index) }}
      </span>
      <span class="flex-1 leading-snug">{{ opt.text }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { Check } from '@lucide/vue';
import type { QuestionResponse } from '../../../../types';
import { optionLetter } from '../../quizFormat';

const props = withDefaults(
  defineProps<{
    question: QuestionResponse;
    /** single_choice: selected option index; multiple_choice: selected option indices */
    modelValue?: number;
    selectedIndexes?: number[];
    multiple?: boolean;
  }>(),
  {
    modelValue: undefined,
    selectedIndexes: () => [],
    multiple: false,
  },
);

defineEmits<{ select: [index: number] }>();

const isSelected = (index: number) =>
  props.multiple ? props.selectedIndexes.includes(index) : props.modelValue === index;
</script>
