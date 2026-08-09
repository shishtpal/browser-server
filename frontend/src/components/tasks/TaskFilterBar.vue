<template>
  <div
    class="mb-4 flex scrollbar-none items-center gap-1 overflow-x-auto rounded-xl bg-slate-100/80 p-1 dark:bg-slate-800/60"
    role="group"
    aria-label="Filter tasks by status"
  >
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-bold transition"
      :class="
        filter === option.value
          ? 'bg-white text-violet-700 shadow-sm dark:bg-slate-900 dark:text-violet-300'
          : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
      "
      :aria-pressed="filter === option.value"
      @click="$emit('change', option.value)"
    >
      <component
        v-if="option.icon"
        :is="option.icon"
        class="h-3.5 w-3.5"
        :stroke-width="2.25"
        aria-hidden="true"
      />
      {{ option.label }}
      <span
        v-if="option.count !== null"
        class="rounded-full px-1.5 py-px text-[9px] font-bold tabular-nums"
        :class="
          filter === option.value
            ? 'bg-violet-100 text-violet-700 dark:bg-violet-900/50 dark:text-violet-300'
            : 'bg-slate-200/80 text-slate-500 dark:bg-slate-700 dark:text-slate-400'
        "
      >
        {{ option.count }}
      </span>
    </button>

    <span
      v-if="live"
      class="mr-1 ml-auto flex shrink-0 items-center gap-1.5 text-[11px] font-medium text-violet-600 dark:text-violet-400"
      role="status"
    >
      <span class="relative flex h-1.5 w-1.5">
        <span
          class="absolute inline-flex h-full w-full animate-ping rounded-full bg-violet-400 opacity-75"
        ></span>
        <span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-violet-500"></span>
      </span>
      Live
    </span>
  </div>
</template>

<script setup lang="ts">
import type { AITaskStatus } from '@browser-server/shared-types';
import type { TaskFilter } from './composables/useAITasks';
import { computed } from 'vue';
import type { LucideIcon } from '@lucide/vue';
import { ListChecks } from '@lucide/vue';
import { TASK_STATUS_META, TASK_STATUS_ORDER } from './taskFormat';

const props = defineProps<{
  filter: TaskFilter;
  counts: Record<AITaskStatus, number>;
  /** Polling is active (anything queued/running). */
  live: boolean;
}>();

defineEmits<{ change: [filter: TaskFilter] }>();

const options = computed<
  { value: TaskFilter; label: string; icon: LucideIcon | null; count: number | null }[]
>(() => [
  { value: 'all', label: 'All', icon: ListChecks, count: null },
  ...TASK_STATUS_ORDER.map((value) => ({
    value,
    label: TASK_STATUS_META[value].label,
    icon: TASK_STATUS_META[value].icon,
    count: props.counts[value] ?? 0,
  })),
]);
</script>

<style scoped>
.scrollbar-none {
  scrollbar-width: none;
}
.scrollbar-none::-webkit-scrollbar {
  display: none;
}
</style>
