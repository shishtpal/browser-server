<template>
  <div class="mb-4">
    <div class="flex flex-col gap-3">
      <!-- Row 1: Title, Stats, and right-aligned Controls -->
      <div class="flex flex-wrap items-center gap-3">
        <div>
          <p
            class="mb-0.5 inline-flex rounded-full px-2 py-0.5 text-[10px] font-bold tracking-[0.2em] uppercase transition-colors"
            :class="badgeClass"
          >
            {{ badge }}
          </p>
          <h1
            class="text-xl font-black tracking-tight text-slate-900 transition-colors sm:text-2xl dark:text-white"
          >
            {{ title }}
          </h1>
        </div>
        <div data-slot="stats" v-if="$slots.stats" class="flex flex-wrap items-center gap-1.5">
          <slot name="stats"></slot>
        </div>
        <div
          data-slot="controls"
          v-if="$slots.controls"
          class="ml-auto flex flex-wrap items-center gap-1.5"
        >
          <slot name="controls"></slot>
        </div>
      </div>

      <!-- Row 2: Actions -->
      <div
        data-slot="actions"
        v-if="$slots.actions"
        class="flex w-full flex-wrap items-center gap-1.5"
      >
        <slot name="actions"></slot>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(
  defineProps<{
    badge: string;
    title: string;
    color?: 'indigo' | 'cyan' | 'violet' | 'emerald' | 'amber' | 'rose';
  }>(),
  {
    color: 'indigo',
  },
);

const badgeClass = computed(() => {
  const colors = {
    indigo: 'bg-indigo-50 text-indigo-600 dark:bg-indigo-900/20 dark:text-indigo-400',
    cyan: 'bg-cyan-50 text-cyan-700 dark:bg-cyan-900/20 dark:text-cyan-400',
    violet: 'bg-violet-50 text-violet-700 dark:bg-violet-900/20 dark:text-violet-400',
    emerald: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400',
    amber: 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400',
    rose: 'bg-rose-50 text-rose-700 dark:bg-rose-900/20 dark:text-rose-400',
  };
  return colors[props.color];
});
</script>
