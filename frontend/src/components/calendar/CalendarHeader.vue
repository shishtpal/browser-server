<template>
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <!-- Navigation -->
    <div class="flex items-center gap-2">
      <button
        type="button"
        class="grid h-9 w-9 place-items-center rounded-lg border border-gray-200 text-slate-600 transition hover:border-violet-300 hover:bg-gray-100 hover:text-violet-600 sm:h-8 sm:w-8 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700"
        aria-label="Previous period"
        @click="$emit('navigate', -1)"
      >
        <ChevronLeft class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
      </button>
      <button
        type="button"
        class="inline-flex h-9 items-center gap-1.5 rounded-lg border border-gray-200 px-3 text-xs font-black text-slate-700 transition hover:border-violet-300 hover:bg-gray-100 hover:text-violet-600 sm:h-8 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700"
        @click="$emit('today')"
      >
        <CalendarCheck class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
        Today
      </button>
      <button
        type="button"
        class="grid h-9 w-9 place-items-center rounded-lg border border-gray-200 text-slate-600 transition hover:border-violet-300 hover:bg-gray-100 hover:text-violet-600 sm:h-8 sm:w-8 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700"
        aria-label="Next period"
        @click="$emit('navigate', 1)"
      >
        <ChevronRight class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
      </button>
      <h2 class="ml-1 min-w-0 truncate text-sm font-black text-slate-900 dark:text-white">
        {{ periodLabel }}
      </h2>
    </div>

    <!-- View switcher -->
    <div
      class="flex scrollbar-none items-center gap-1 overflow-x-auto rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-slate-600 dark:bg-slate-800"
      role="group"
      aria-label="Calendar view"
    >
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        class="flex shrink-0 items-center gap-1.5 rounded-md px-3 py-2 text-xs font-bold transition sm:py-1.5"
        :class="
          option.value === currentView
            ? 'bg-white text-violet-700 shadow-sm dark:bg-slate-700 dark:text-violet-300'
            : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
        "
        :aria-pressed="option.value === currentView"
        @click="$emit('changeView', option.value)"
      >
        <component :is="option.icon" class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
        {{ option.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CalendarView } from './types';
import {
  Calendar1,
  ChevronLeft,
  CalendarCheck,
  ChevronRight,
  CalendarDays,
  CalendarRange,
  LayoutGrid,
  type LucideIcon,
} from '@lucide/vue';

defineProps<{
  periodLabel: string;
  currentView: CalendarView;
}>();

defineEmits<{
  (e: 'navigate', dir: number): void;
  (e: 'today'): void;
  (e: 'changeView', view: CalendarView): void;
}>();

const options: { value: CalendarView; label: string; icon: LucideIcon }[] = [
  { value: 'day', label: 'Day', icon: Calendar1 },
  { value: 'week', label: 'Week', icon: CalendarRange },
  { value: 'month', label: 'Month', icon: CalendarDays },
  { value: 'year', label: 'Year', icon: LayoutGrid },
];
</script>

<style scoped>
.scrollbar-none {
  scrollbar-width: none;
}

.scrollbar-none::-webkit-scrollbar {
  display: none;
}
</style>
