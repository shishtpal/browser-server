<template>
  <span
    v-if="label"
    class="inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] font-black"
    :class="badgeClass"
  >
    <Clock
      v-if="label === 'Overdue' || label === 'Today'"
      class="h-2.5 w-2.5"
      :stroke-width="2.5"
      aria-hidden="true"
    />
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Clock } from '@lucide/vue';
import type { TodoStatus } from '../../types';
import { dueDateBadgeClass, dueDateLabel } from './todoFormat';

const props = withDefaults(
  defineProps<{
    dueDate: string | null;
    status?: TodoStatus;
  }>(),
  { status: 'pending' },
);

const partialTodo = computed(() => ({
  start_date: props.dueDate,
  status: props.status,
}));

const badgeClass = computed(() => dueDateBadgeClass(partialTodo.value));
const label = computed(() => dueDateLabel(partialTodo.value));
</script>
