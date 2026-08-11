<template>
  <div
    class="overflow-hidden rounded-2xl border border-emerald-200/80 bg-gradient-to-b from-emerald-50/70 to-emerald-100/40 p-5 text-center shadow-sm sm:p-6 dark:border-emerald-800/60 dark:from-emerald-950/30 dark:to-slate-900/60"
  >
    <div
      class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-900/50 dark:text-emerald-300"
    >
      <PartyPopper v-if="practiceMode" class="h-6 w-6" :stroke-width="2" aria-hidden="true" />
      <CircleCheck v-else class="h-6 w-6" :stroke-width="2" aria-hidden="true" />
    </div>

    <h2 class="mt-3 text-lg font-black text-emerald-900 sm:text-xl dark:text-emerald-100">
      {{ practiceMode ? 'Practice Complete!' : 'Session Complete!' }}
    </h2>

    <p v-if="practiceMode" class="mt-1 text-sm text-emerald-800 dark:text-emerald-300">
      You practiced the selected cards without altering their spaced repetition schedule.
    </p>
    <p v-else class="mt-1 text-sm text-emerald-800 dark:text-emerald-300">
      Great job! You reviewed <span class="font-bold">{{ reviewed }}</span> cards in this session.
    </p>
    <p
      v-if="skippedCount > 0"
      class="mt-1 text-sm font-medium text-violet-700 dark:text-violet-300"
    >
      You skipped <span class="font-bold">{{ skippedCount }}</span>
      {{ skippedCount === 1 ? 'question' : 'questions' }}.
    </p>

    <!-- Rating breakdown -->
    <div
      v-if="!practiceMode && reviewed > 0"
      class="mx-auto mt-5 grid max-w-lg grid-cols-2 gap-2 sm:grid-cols-4"
    >
      <div
        v-for="stat in ratingStats"
        :key="stat.label"
        class="rounded-xl border p-2.5"
        :class="stat.containerClass"
      >
        <span
          class="flex items-center justify-center gap-1 text-[11px] font-bold uppercase"
          :class="stat.labelClass"
        >
          <component :is="stat.icon" class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          {{ stat.label }}
        </span>
        <span class="text-lg font-black tabular-nums" :class="stat.valueClass">{{
          stat.count
        }}</span>
      </div>
    </div>

    <div class="mt-6 flex flex-col justify-center gap-3 sm:flex-row">
      <Button variant="gradient-violet" size="md" class="w-full sm:w-auto" @click="emit('again')">
        <span class="inline-flex items-center gap-1.5">
          <RotateCcw class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          {{ practiceMode ? 'Practice more' : 'Review more' }}
        </span>
      </Button>
      <Button variant="secondary" size="md" class="w-full sm:w-auto" @click="emit('change-tags')">
        <span class="inline-flex items-center gap-1.5">
          <Tags class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
          Change tags
        </span>
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  CircleCheck,
  CircleX,
  PartyPopper,
  RotateCcw,
  Tags,
  TrendingUp,
  Zap,
  type LucideIcon,
} from '@lucide/vue';
import type { ReviewRating } from '../../../types';
import Button from '../../ui/Button.vue';

const props = defineProps<{
  practiceMode: boolean;
  reviewed: number;
  ratingCounts: Record<ReviewRating, number>;
  skippedCount: number;
}>();

const emit = defineEmits<{
  again: [];
  'change-tags': [];
}>();

const ratingStats = computed<
  Array<{
    label: string;
    icon: LucideIcon;
    count: number;
    containerClass: string;
    labelClass: string;
    valueClass: string;
  }>
>(() => [
  {
    label: 'Again',
    icon: CircleX,
    count: props.ratingCounts.again,
    containerClass: 'border-rose-200 bg-rose-50/80 dark:border-rose-900/40 dark:bg-rose-950/30',
    labelClass: 'text-rose-600 dark:text-rose-400',
    valueClass: 'text-rose-700 dark:text-rose-300',
  },
  {
    label: 'Hard',
    icon: Zap,
    count: props.ratingCounts.hard,
    containerClass: 'border-amber-200 bg-amber-50/80 dark:border-amber-900/40 dark:bg-amber-950/30',
    labelClass: 'text-amber-600 dark:text-amber-400',
    valueClass: 'text-amber-700 dark:text-amber-300',
  },
  {
    label: 'Good',
    icon: TrendingUp,
    count: props.ratingCounts.good,
    containerClass: 'border-sky-200 bg-sky-50/80 dark:border-sky-900/40 dark:bg-sky-950/30',
    labelClass: 'text-sky-600 dark:text-sky-400',
    valueClass: 'text-sky-700 dark:text-sky-300',
  },
  {
    label: 'Easy',
    icon: CircleCheck,
    count: props.ratingCounts.easy,
    containerClass:
      'border-emerald-200 bg-emerald-100/60 dark:border-emerald-800/50 dark:bg-emerald-950/50',
    labelClass: 'text-emerald-700 dark:text-emerald-400',
    valueClass: 'text-emerald-800 dark:text-emerald-300',
  },
]);
</script>
