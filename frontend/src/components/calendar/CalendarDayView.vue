<template>
  <div class="flex flex-col gap-3">
    <!-- Day summary header -->
    <div
      class="flex items-center gap-3 rounded-xl border border-gray-200/80 bg-white p-3 dark:border-slate-700/80 dark:bg-slate-800/90"
    >
      <div
        class="grid h-12 w-12 shrink-0 place-items-center rounded-xl text-xl font-black tabular-nums"
        :class="
          day?.isToday
            ? 'bg-violet-600 text-white dark:bg-violet-500'
            : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
        "
      >
        {{ dayNumber }}
      </div>
      <div class="min-w-0">
        <div class="truncate text-sm font-black text-slate-800 dark:text-slate-200">
          {{ dayName }}
        </div>
        <div class="text-xs text-slate-500 dark:text-slate-400">
          {{ fullDate }} · {{ todoCount }} todo{{ todoCount !== 1 ? 's' : '' }}
        </div>
      </div>
    </div>

    <!-- All-day task list -->
    <div
      class="flex-1 divide-y divide-gray-100 rounded-xl border border-gray-200/80 bg-white dark:divide-slate-700/60 dark:border-slate-700/80 dark:bg-slate-800/90"
    >
      <p
        v-if="todoCount === 0"
        class="flex items-center justify-center gap-2 px-4 py-10 text-xs font-semibold text-slate-400 dark:text-slate-500"
      >
        <Inbox class="h-4 w-4" :stroke-width="2" aria-hidden="true" />
        Nothing scheduled for this day.
      </p>

      <button
        v-for="todo in dayTodos"
        :key="todo.id"
        type="button"
        class="flex w-full items-start gap-3 px-3 py-2.5 text-left transition hover:bg-violet-50/60 dark:hover:bg-violet-950/20"
        :class="{ 'opacity-60': todo.status === 'completed' }"
        @click.stop="$emit('todoClick', todo)"
      >
        <span
          class="mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full"
          :class="todoDotClass(todo)"
          aria-hidden="true"
        ></span>
        <span class="min-w-0 flex-1">
          <span
            class="block text-sm font-bold"
            :class="
              todo.status === 'completed'
                ? 'text-slate-400 line-through dark:text-slate-500'
                : 'text-slate-800 dark:text-slate-100'
            "
          >
            {{ todo.title }}
          </span>
          <span
            v-if="todo.description"
            class="mt-0.5 block truncate text-[11px] text-slate-500 dark:text-slate-400"
            v-html="linkifyDescription(todo.description)"
          ></span>
          <span class="mt-1 flex flex-wrap items-center gap-1">
            <TodoPriorityBadge :priority="todo.priority" />
            <span
              v-if="todo.rrule"
              class="inline-flex items-center gap-0.5 rounded-full bg-blue-50 px-1.5 py-0.5 text-[9px] font-black text-blue-600 dark:bg-blue-900/20 dark:text-blue-400"
            >
              <Repeat class="h-2.5 w-2.5" :stroke-width="2.5" aria-hidden="true" />
              Recurring
            </span>
          </span>
        </span>
        <ChevronRight
          class="mt-1 h-4 w-4 shrink-0 text-slate-300 dark:text-slate-600"
          :stroke-width="2.5"
          aria-hidden="true"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ChevronRight, Inbox, Repeat } from '@lucide/vue';
import type { CalendarDay } from './types';
import type { Todo } from '../../types';
import { linkifyDescription } from '../../lib/descriptionLinks';
import TodoPriorityBadge from '../todos/TodoPriorityBadge.vue';
import { todoDotClass } from '../todos/todoFormat';

const props = defineProps<{
  day: CalendarDay | undefined;
}>();

defineEmits<{
  (e: 'todoClick', todo: Todo): void;
}>();

const dayTodos = computed(() => props.day?.todos ?? []);
const todoCount = computed(() => dayTodos.value.length);

const parsedDay = computed(() => (props.day ? new Date(props.day.date + 'T00:00:00') : undefined));

const dayNumber = computed(() => parsedDay.value?.getDate() ?? '');
const dayName = computed(
  () => parsedDay.value?.toLocaleDateString('en-US', { weekday: 'long' }) ?? '',
);
const fullDate = computed(
  () =>
    parsedDay.value?.toLocaleDateString('en-US', {
      month: 'long',
      day: 'numeric',
      year: 'numeric',
    }) ?? '',
);
</script>
