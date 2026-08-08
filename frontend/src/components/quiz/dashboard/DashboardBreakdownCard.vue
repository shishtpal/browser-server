<template>
  <div
    class="flex flex-col rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
  >
    <div class="mb-2 flex items-center justify-between">
      <h3
        class="flex items-center gap-1.5 text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400"
      >
        <component :is="icon" class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        {{ title }}
      </h3>
      <span v-if="hint" class="text-[10px] font-semibold text-slate-400">{{ hint }}</span>
    </div>

    <div
      v-if="items.length"
      class="max-h-44 scrollbar-thin scrollbar-thumb-slate-300 space-y-1 overflow-y-auto pr-1 dark:scrollbar-thumb-slate-600"
    >
      <div
        v-for="item in items"
        :key="item.key"
        class="flex items-center justify-between gap-2 py-1 text-xs"
      >
        <span
          class="flex min-w-0 items-center gap-1.5 font-semibold text-slate-600 dark:text-slate-300"
        >
          <span
            v-if="item.dotClass"
            class="h-2 w-2 shrink-0 rounded-full"
            :class="item.dotClass"
            aria-hidden="true"
          ></span>
          <component
            :is="item.icon"
            v-else-if="item.icon"
            class="h-3.5 w-3.5 shrink-0 text-slate-400"
            :stroke-width="2.25"
            aria-hidden="true"
          />
          <span class="truncate" :title="item.label">{{ item.label }}</span>
        </span>
        <span
          class="shrink-0 rounded px-2 py-0.5 font-black"
          :class="
            item.accent
              ? 'bg-violet-50 text-violet-700 dark:bg-violet-900/40 dark:text-violet-300'
              : 'bg-slate-100 text-slate-900 dark:bg-slate-700 dark:text-white'
          "
        >
          {{ item.count }}
        </span>
      </div>
    </div>
    <p v-else class="flex flex-1 items-center text-xs text-slate-400">
      {{ emptyText }}
    </p>
  </div>
</template>

<script setup lang="ts">
import type { LucideIcon } from '@lucide/vue';

export interface BreakdownItem {
  key: string;
  label: string;
  count: number;
  /** Colored dot (difficulty) or leading icon — not both. */
  dotClass?: string;
  icon?: LucideIcon;
  /** Uses the violet accent style for the count pill. */
  accent?: boolean;
}

withDefaults(
  defineProps<{
    title: string;
    icon: LucideIcon;
    items: BreakdownItem[];
    hint?: string;
    emptyText?: string;
  }>(),
  {
    hint: undefined,
    emptyText: 'Nothing here yet.',
  },
);
</script>

<style scoped>
.scrollbar-thin {
  scrollbar-width: thin;
}
</style>
