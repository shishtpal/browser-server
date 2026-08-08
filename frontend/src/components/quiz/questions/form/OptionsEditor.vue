<template>
  <div
    class="space-y-4 rounded-xl border border-gray-200 bg-gray-50/50 p-3 sm:rounded-2xl sm:p-5 dark:border-slate-700 dark:bg-slate-800/30"
  >
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="min-w-0">
        <span class="block text-sm font-bold text-slate-700 dark:text-slate-300">Options</span>
        <span class="block text-xs text-slate-500 dark:text-slate-400">
          Tap the letter to mark {{ single ? 'exactly one' : 'one or more' }} correct
        </span>
      </div>
      <Button
        variant="secondary"
        size="sm"
        class="shrink-0"
        :disabled="options.length >= maxOptions"
        @click="addOption"
      >
        <span class="inline-flex items-center gap-1">
          <Plus class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Add option
        </span>
      </Button>
    </div>

    <TransitionGroup tag="div" name="row" class="space-y-2">
      <div v-for="(opt, i) in options" :key="i" class="flex items-center gap-2 sm:gap-3">
        <!-- Letter / correct toggle -->
        <button
          type="button"
          class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border-2 text-sm font-bold transition"
          :class="
            opt.correct
              ? 'border-emerald-500 bg-emerald-500 text-white shadow-sm shadow-emerald-500/30'
              : 'border-gray-300 bg-white text-slate-500 hover:border-emerald-400 hover:text-emerald-600 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-400 dark:hover:border-emerald-500'
          "
          :title="opt.correct ? 'Correct answer' : 'Mark as correct'"
          :aria-pressed="opt.correct"
          @click="toggleCorrect(i)"
        >
          <Check v-if="opt.correct" class="h-4 w-4" :stroke-width="3" aria-hidden="true" />
          <span v-else>{{ optionLetter(i) }}</span>
        </button>

        <div class="min-w-0 flex-1">
          <InputField v-model="opt.text" :placeholder="`Option ${optionLetter(i)}`" flex />
        </div>

        <button
          type="button"
          class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-slate-400 transition hover:bg-rose-50 hover:text-rose-500 disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-slate-400 dark:hover:bg-rose-900/20"
          :disabled="options.length <= minOptions"
          aria-label="Remove option"
          @click="removeOption(i)"
        >
          <Trash2 class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { Check, Plus, Trash2 } from '@lucide/vue';
import type { QuestionOption } from '../../../../types';
import Button from '../../../ui/Button.vue';
import InputField from '../../../ui/InputField.vue';
import { optionLetter } from '../../quizFormat';

const props = withDefaults(
  defineProps<{
    options: QuestionOption[];
    /** single_choice: exactly one correct; multiple_choice: any number. */
    single: boolean;
    minOptions?: number;
    maxOptions?: number;
  }>(),
  {
    minOptions: 2,
    maxOptions: 10,
  },
);

const addOption = () => {
  if (props.options.length >= props.maxOptions) return;
  props.options.push({ index: props.options.length, text: '', correct: false });
};

const removeOption = (index: number) => {
  props.options.splice(index, 1);
  props.options.forEach((o, idx) => (o.index = idx));
};

const toggleCorrect = (index: number) => {
  if (props.single) {
    props.options.forEach((o, idx) => (o.correct = idx === index));
  } else {
    props.options[index].correct = !props.options[index].correct;
  }
};
</script>

<style scoped>
.row-enter-active,
.row-leave-active {
  transition: all 0.2s ease;
}
.row-enter-from,
.row-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
