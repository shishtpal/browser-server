<template>
  <div
    class="sticky top-0 z-10 border-b border-slate-200 bg-white/95 px-3 py-2.5 shadow-sm backdrop-blur sm:px-4 dark:border-slate-800 dark:bg-slate-900/95"
  >
    <div class="flex flex-wrap items-center justify-between gap-2 sm:gap-3">
      <div class="flex min-w-0 items-center gap-2 sm:gap-3">
        <span
          class="inline-flex shrink-0 items-center gap-1.5 rounded-full bg-violet-50 px-2.5 py-1 text-xs font-bold text-violet-700 tabular-nums sm:px-3 dark:bg-violet-900/40 dark:text-violet-300"
          role="timer"
          aria-label="Elapsed time"
        >
          <Timer class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          {{ formatTime(elapsedTime) }}
        </span>
        <span class="truncate text-xs font-medium text-slate-500 tabular-nums dark:text-slate-400">
          Q {{ currentIndex + 1 }} <span class="text-slate-300 dark:text-slate-600">/</span>
          {{ total }}
        </span>
      </div>

      <div class="flex shrink-0 items-center gap-1.5 sm:gap-2">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-slate-100 px-2.5 py-1.5 text-xs font-semibold text-slate-700 transition hover:bg-slate-200 md:hidden dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
          :aria-expanded="paletteOpen"
          @click="$emit('toggle-palette')"
        >
          <LayoutGrid class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          {{ paletteOpen ? 'Hide grid' : 'Grid' }}
        </button>

        <Button variant="gradient-emerald" size="sm" @click="$emit('submit')">
          <span class="inline-flex items-center gap-1.5">
            <CircleCheck class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
            Finish exam
          </span>
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CircleCheck, LayoutGrid, Timer } from '@lucide/vue';
import Button from '../../../ui/Button.vue';
import { formatTime } from '../../quizFormat';

defineProps<{
  elapsedTime: number;
  currentIndex: number;
  total: number;
  paletteOpen: boolean;
}>();

defineEmits<{ 'toggle-palette': []; submit: [] }>();
</script>
