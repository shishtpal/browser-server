<template>
  <div class="space-y-2">
    <p class="mb-2 text-xs font-semibold text-slate-500 dark:text-slate-400">
      Arrange the items in the correct order (tap the arrows to move):
    </p>
    <div class="space-y-2">
      <div
        v-for="(itemIdx, seqIdx) in order"
        :key="itemIdx"
        class="flex items-center justify-between gap-3 rounded-xl border p-3 text-sm font-medium transition"
        :class="'border-slate-200 bg-slate-50/70 dark:border-slate-800 dark:bg-slate-900/80'"
      >
        <div class="flex min-w-0 items-center gap-3">
          <span
            class="grid h-7 w-7 shrink-0 place-items-center rounded-lg bg-violet-600 text-xs font-bold text-white tabular-nums"
          >
            {{ seqIdx + 1 }}
          </span>
          <span class="text-slate-800 dark:text-slate-200">
            {{ chronologyItemText(question, itemIdx) }}
          </span>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <button
            type="button"
            class="grid h-9 w-9 place-items-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-100 disabled:opacity-30 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
            :disabled="seqIdx === 0"
            aria-label="Move item up"
            @click="$emit('move', seqIdx, seqIdx - 1)"
          >
            <ArrowUp class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          </button>
          <button
            type="button"
            class="grid h-9 w-9 place-items-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-100 disabled:opacity-30 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
            :disabled="seqIdx === order.length - 1"
            aria-label="Move item down"
            @click="$emit('move', seqIdx, seqIdx + 1)"
          >
            <ArrowDown class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowDown, ArrowUp } from '@lucide/vue';
import type { QuestionResponse } from '../../../../types';
import { chronologyItemText } from '../../quizFormat';

defineProps<{ question: QuestionResponse; order: number[] }>();
defineEmits<{ move: [fromIndex: number, toIndex: number] }>();
</script>
