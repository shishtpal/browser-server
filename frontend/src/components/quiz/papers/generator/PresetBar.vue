<template>
  <div
    class="flex flex-col gap-3 rounded-2xl border border-slate-200/80 bg-white p-4 shadow-sm sm:flex-row sm:items-center sm:justify-between dark:border-slate-800 dark:bg-slate-900"
  >
    <div class="flex items-center gap-2.5">
      <span
        class="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-violet-100 text-violet-600 dark:bg-violet-900/40 dark:text-violet-300"
      >
        <Sparkles class="h-4.5 w-4.5 sm:h-5 sm:w-5" :stroke-width="2.25" aria-hidden="true" />
      </span>
      <div>
        <h3 class="text-xs font-bold tracking-wider text-slate-800 uppercase dark:text-slate-200">
          Quick presets
        </h3>
        <p class="text-[11px] text-slate-500 dark:text-slate-400">
          Tap a preset to fill in the form instantly
        </p>
      </div>
    </div>

    <div class="grid grid-cols-3 gap-2 sm:flex sm:flex-wrap">
      <button
        v-for="preset in presets"
        :key="preset.name"
        type="button"
        class="inline-flex items-center justify-center gap-1.5 rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-[11px] font-bold text-slate-700 transition hover:border-violet-300 hover:bg-violet-50 hover:text-violet-800 sm:text-xs dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-violet-700 dark:hover:bg-violet-950/40 dark:hover:text-violet-300"
        @click="$emit('apply', preset)"
      >
        <component
          :is="preset.icon"
          class="hidden h-3.5 w-3.5 sm:inline"
          :stroke-width="2.25"
          aria-hidden="true"
        />
        {{ preset.name }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Target, Zap, NotebookPen, type LucideIcon } from '@lucide/vue';
import type { QuestionDifficulty } from '../../../../types';

export interface PaperPreset {
  name: string;
  title: string;
  icon: LucideIcon;
  sections: Array<{ count: number; difficulty?: QuestionDifficulty }>;
}

const presets: PaperPreset[] = [
  {
    name: 'Quick 10',
    title: 'Quick 10 Question Quiz',
    icon: Zap,
    sections: [{ count: 10 }],
  },
  {
    name: '30 Mock',
    title: 'Comprehensive Mock Exam',
    icon: NotebookPen,
    sections: [
      { count: 15, difficulty: 'medium' },
      { count: 15, difficulty: 'hard' },
    ],
  },
  {
    name: '50 Full',
    title: 'Full Length Examination',
    icon: Target,
    sections: [{ count: 50 }],
  },
];

defineEmits<{ apply: [preset: PaperPreset] }>();
</script>
