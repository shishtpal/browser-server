<template>
  <button
    type="button"
    :disabled="status === 'archived'"
    :aria-label="ariaLabel"
    :title="ariaLabel"
    :aria-pressed="status === 'completed'"
    class="grid shrink-0 place-items-center rounded-full border-2 transition disabled:cursor-default"
    :class="[sizeClass, toggleClass]"
    @click="$emit('toggle')"
  >
    <Check
      v-if="status === 'completed'"
      :class="iconClass"
      :stroke-width="3.5"
      aria-hidden="true"
    />
    <LoaderCircle
      v-else-if="status === 'in_progress'"
      :class="[iconClass, 'animate-spin [animation-duration:1.8s]']"
      :stroke-width="3"
      aria-hidden="true"
    />
  </button>
</template>

<script setup lang="ts">
import type { TodoStatus } from '../../types';
import { computed } from 'vue';
import { Check, LoaderCircle } from '@lucide/vue';
import { statusAriaLabel } from './todoFormat';

const props = withDefaults(
  defineProps<{
    status: TodoStatus;
    size?: 'sm' | 'md';
  }>(),
  { size: 'md' },
);

defineEmits<{ toggle: [] }>();

const ariaLabel = computed(() => statusAriaLabel(props.status));

const sizeClass = computed(() => (props.size === 'sm' ? 'h-4 w-4' : 'h-5 w-5'));
const iconClass = computed(() => (props.size === 'sm' ? 'h-2.5 w-2.5' : 'h-3 w-3'));

const toggleClass = computed(() => {
  if (props.status === 'completed') return 'border-emerald-500 bg-emerald-500 text-white';
  if (props.status === 'in_progress') return 'border-blue-500 bg-blue-500 text-white';
  return 'border-gray-300 text-transparent hover:border-indigo-400 dark:border-slate-600 dark:hover:border-indigo-400';
});
</script>
