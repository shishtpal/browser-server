<template>
  <section
    class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-colors dark:border-slate-700 dark:bg-slate-800/90"
    aria-labelledby="usage-trend-heading"
  >
    <div class="mb-3 flex items-center justify-between gap-2">
      <h3
        id="usage-trend-heading"
        class="flex items-center gap-1.5 text-xs font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
      >
        <TrendingUp class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
        Trend · {{ groupBy }}
      </h3>
      <span class="text-[10px] font-semibold text-slate-400 dark:text-slate-500">
        Peak: {{ formatDuration(maxValue) }}
      </span>
    </div>

    <div
      class="flex items-end gap-1 sm:gap-1.5"
      :style="{ height: `${chartHeight + topLabelHeight + bottomLabelHeight}px` }"
      role="img"
      :aria-label="`Usage trend over ${points.length} ${groupBy} periods; peak ${formatDuration(maxValue)}`"
    >
      <div
        v-for="(point, index) in points"
        :key="point.period"
        class="group flex min-w-0 flex-1 flex-col items-center justify-end"
        :title="`${labels[index] ?? point.period}: ${formatDuration(point.total_seconds)}`"
      >
        <!-- Value on top (desktop only) -->
        <span
          class="mb-1 hidden text-[10px] font-medium text-slate-500 tabular-nums sm:block dark:text-slate-400"
          :class="{ 'opacity-0': point.total_seconds === 0 }"
        >
          {{ formatDuration(point.total_seconds) }}
        </span>

        <div
          class="w-full rounded-t-md transition-all duration-300 group-hover:opacity-100"
          :class="
            point.total_seconds > 0
              ? 'bg-gradient-to-t from-rose-600/70 to-rose-400/80 hover:from-rose-600 hover:to-rose-400'
              : 'bg-slate-100 dark:bg-slate-700'
          "
          :style="{ height: barHeight(point.total_seconds) }"
        />

        <!-- Period label -->
        <span
          class="mt-1 max-w-full truncate text-[9px] leading-tight text-slate-400 dark:text-slate-500"
          :class="{ 'opacity-40': point.total_seconds === 0 }"
        >
          {{ labels[index] ?? point.period }}
        </span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { TrendingUp } from '@lucide/vue';
import type { TimelinePoint } from '../../types';
import { formatDuration } from '../../lib/utils';
import type { GroupBy } from './analyticsFormat';

const props = withDefaults(
  defineProps<{
    points: TimelinePoint[];
    labels: string[];
    maxValue: number;
    groupBy: GroupBy;
  }>(),
  { groupBy: 'day' },
);

const chartHeight = 160;
const topLabelHeight = 16;
const bottomLabelHeight = 18;

const barHeight = (seconds: number): string => {
  if (props.maxValue <= 0 || seconds <= 0) return '3px';
  return `${Math.max(6, Math.round((seconds / props.maxValue) * chartHeight))}px`;
};
</script>
