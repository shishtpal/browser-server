<template>
  <div v-if="hasAny" class="flex flex-wrap items-center gap-1">
    <TodoDueDateBadge v-if="todo.start_date" :due-date="todo.start_date" :status="todo.status" />

    <span
      v-if="todo.end_date && todo.end_date !== todo.start_date"
      class="inline-flex items-center gap-0.5 rounded-full bg-gray-100 px-1.5 py-0.5 text-[10px] font-black text-slate-500 dark:bg-slate-700 dark:text-slate-400"
      title="End date"
    >
      <ArrowRight class="h-2.5 w-2.5" :stroke-width="2.5" aria-hidden="true" />
      {{ endDateLabel }}
    </span>

    <span
      v-if="todo.rrule"
      class="inline-flex items-center gap-0.5 rounded-full bg-blue-50 px-1.5 py-0.5 text-[10px] font-black text-blue-600 dark:bg-blue-900/20 dark:text-blue-400"
      :title="`Recurring: ${rruleLabel}`"
    >
      <Repeat class="h-2.5 w-2.5" :stroke-width="2.5" aria-hidden="true" />
      {{ rruleLabel }}
    </span>

    <span
      v-if="todo.domain"
      class="inline-flex items-center gap-0.5 rounded-full bg-violet-50 px-1.5 py-0.5 text-[10px] font-black text-violet-600 dark:bg-violet-900/20 dark:text-violet-400"
    >
      <Globe class="h-2.5 w-2.5" :stroke-width="2.5" aria-hidden="true" />
      {{ todo.domain }}
    </span>

    <TodoTagBadges :tags="todo.tags || []" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ArrowRight, Globe, Repeat } from '@lucide/vue';
import type { Todo } from '../../types';
import TodoDueDateBadge from './TodoDueDateBadge.vue';
import TodoTagBadges from './TodoTagBadges.vue';
import { formatRrule } from './todoFormat';

const props = defineProps<{ todo: Todo }>();

const rruleLabel = computed(() => (props.todo.rrule ? formatRrule(props.todo.rrule) : ''));

const endDateLabel = computed(() =>
  props.todo.end_date ? new Date(props.todo.end_date).toLocaleDateString() : '',
);

const hasAny = computed(
  () =>
    Boolean(props.todo.start_date) ||
    Boolean(props.todo.end_date && props.todo.end_date !== props.todo.start_date) ||
    Boolean(props.todo.rrule) ||
    Boolean(props.todo.domain) ||
    (props.todo.tags || []).length > 0,
);
</script>
