<template>
  <div
    class="absolute right-1 -bottom-3 hidden items-center gap-0.5 rounded-lg border border-slate-200 bg-white p-0.5 shadow-sm group-focus-within:flex group-hover:flex dark:border-white/10 dark:bg-slate-900"
    :class="positionClass"
  >
    <button
      v-for="action in actions"
      :key="action.name"
      class="rounded-md p-1.5 text-slate-400 transition dark:hover:bg-white/10"
      :class="[action.className, active.includes(action.name) ? action.activeClassName : '']"
      :title="action.title"
      :aria-label="action.title"
      :aria-pressed="active.includes(action.name) ? true : undefined"
      :aria-busy="busy.includes(action.name) ? true : undefined"
      :disabled="busy.includes(action.name)"
      type="button"
      @click="$emit('action', action.name)"
    >
      <LoaderCircle
        v-if="busy.includes(action.name)"
        class="h-3.5 w-3.5 animate-spin"
        :stroke-width="2.25"
        aria-hidden="true"
      />
      <component
        :is="action.icon"
        v-else
        class="h-3.5 w-3.5"
        :stroke-width="2.25"
        aria-hidden="true"
      />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  Copy,
  GitBranch,
  LoaderCircle,
  Sigma,
  Trash2,
  Volume2,
  type LucideIcon,
} from '@lucide/vue';

export type BubbleActionName = 'copy' | 'speak' | 'branch' | 'math' | 'delete';

const props = withDefaults(
  defineProps<{
    /** Which actions to show, in order. */
    include?: BubbleActionName[];
    /** Actions currently toggled on (shown pressed). */
    active?: BubbleActionName[];
    /** Actions currently in-flight (shown as a spinner). */
    busy?: BubbleActionName[];
  }>(),
  { include: () => ['copy', 'speak', 'branch', 'delete'], active: () => [], busy: () => [] },
);

defineEmits<{ action: [name: BubbleActionName] }>();

const ALL: Record<
  BubbleActionName,
  {
    name: BubbleActionName;
    title: string;
    icon: LucideIcon;
    className: string;
    activeClassName?: string;
  }
> = {
  copy: {
    name: 'copy',
    title: 'Copy',
    icon: Copy,
    className:
      'hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-slate-200',
  },
  speak: {
    name: 'speak',
    title: 'Read aloud (text-to-speech)',
    icon: Volume2,
    className:
      'hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400',
    activeClassName: 'text-indigo-600 dark:text-indigo-400',
  },
  branch: {
    name: 'branch',
    title: 'Branch a new conversation from here',
    icon: GitBranch,
    className:
      'hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400',
  },
  math: {
    name: 'math',
    title: 'Toggle math rendering (MathJax)',
    icon: Sigma,
    className:
      'hover:bg-violet-50 hover:text-violet-600 dark:hover:bg-violet-500/10 dark:hover:text-violet-400',
    activeClassName: 'text-violet-600 dark:text-violet-400',
  },
  delete: {
    name: 'delete',
    title: 'Delete message',
    icon: Trash2,
    className:
      'hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-500/10 dark:hover:text-red-400',
  },
};

const actions = computed(() => props.include.map((name) => ALL[name]));

// The bar lives at the bottom-right of either bubble variant.
const positionClass = 'right-1 -bottom-3';
</script>
