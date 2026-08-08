<template>
  <div
    class="relative overflow-hidden rounded-2xl bg-gradient-to-br from-violet-600 via-indigo-600 to-slate-900 p-5 text-white shadow-xl sm:p-8"
  >
    <div
      class="relative z-10 flex flex-col items-center justify-between gap-5 sm:flex-row sm:gap-6"
    >
      <div class="min-w-0 text-center sm:text-left">
        <span
          class="inline-flex items-center gap-1.5 rounded-full bg-white/20 px-3 py-1 text-xs font-bold backdrop-blur-md"
        >
          <Award class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
          Exam Report Card
        </span>
        <h2 class="mt-2 text-xl leading-snug font-black break-words sm:text-3xl">{{ title }}</h2>
        <p class="mt-1 text-[11px] text-violet-200 sm:text-xs">
          {{ formatDateTime(record.completedAt) }} · Time taken:
          {{ formatTime(record.durationSeconds) }}
        </p>
      </div>

      <!-- Score ring -->
      <div
        class="flex h-24 w-24 shrink-0 flex-col items-center justify-center rounded-full border-4 border-white/30 bg-white/10 text-center backdrop-blur-md"
      >
        <span class="text-2xl font-black tabular-nums">{{ record.percentage }}%</span>
        <span class="text-[10px] font-bold tracking-wider text-violet-200 uppercase tabular-nums">
          {{ record.score }}/{{ record.maxScore }}
        </span>
      </div>
    </div>

    <!-- Metrics -->
    <div class="mt-6 grid grid-cols-3 gap-2 border-t border-white/15 pt-4 text-center sm:gap-3">
      <div
        v-for="metric in metrics"
        :key="metric.label"
        class="rounded-xl p-2.5 backdrop-blur-sm"
        :class="metric.bgClass"
      >
        <p
          class="flex items-center justify-center gap-1 text-[10px] font-bold tracking-wider uppercase"
          :class="metric.labelClass"
        >
          <component :is="metric.icon" class="h-3 w-3" :stroke-width="2.75" aria-hidden="true" />
          {{ metric.label }}
        </p>
        <p class="text-lg font-black tabular-nums" :class="metric.valueClass">{{ metric.value }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Award, CircleCheck, CircleMinus, CircleX, type LucideIcon } from '@lucide/vue';
import type { PaperAttemptRecord } from '../../composables/usePaperAttempt';
import { formatDateTime, formatTime } from '../../quizFormat';

const props = defineProps<{ record: PaperAttemptRecord; title: string }>();

const metrics = computed<
  Array<{
    label: string;
    value: number;
    icon: LucideIcon;
    bgClass: string;
    labelClass: string;
    valueClass: string;
  }>
>(() => [
  {
    label: 'Correct',
    value: props.record.correctCount,
    icon: CircleCheck,
    bgClass: 'bg-emerald-500/20',
    labelClass: 'text-emerald-200',
    valueClass: 'text-emerald-300',
  },
  {
    label: 'Incorrect',
    value: props.record.incorrectCount,
    icon: CircleX,
    bgClass: 'bg-rose-500/20',
    labelClass: 'text-rose-200',
    valueClass: 'text-rose-300',
  },
  {
    label: 'Skipped',
    value: props.record.unansweredCount,
    icon: CircleMinus,
    bgClass: 'bg-amber-500/20',
    labelClass: 'text-amber-200',
    valueClass: 'text-amber-300',
  },
]);
</script>
