<template>
  <div
    class="group dark:hover:bg-slate-850 relative flex min-h-[70px] cursor-pointer flex-col bg-white p-1.5 transition-all duration-150 hover:bg-slate-50/80 sm:min-h-[110px] sm:p-2 dark:bg-slate-900"
    :class="cellClasses"
    role="button"
    :aria-label="`${day.date}, ${day.todos.length} todo${day.todos.length === 1 ? '' : 's'}`"
    @click="$emit('create', day.date)"
    @dragover.prevent="drop.onDragOver(day, $event)"
    @dragleave.prevent="drop.onDragLeave(day, $event)"
    @drop.prevent="drop.onDrop(day, $event)"
  >
    <!-- Date number + count -->
    <div class="mb-1 flex items-center justify-between">
      <span
        class="flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold transition-transform group-hover:scale-105"
        :class="dateClass"
      >
        {{ dayNumber }}
      </span>

      <span
        v-if="day.todos.length > 0"
        class="rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-500 tabular-nums dark:bg-slate-800 dark:text-slate-400"
      >
        {{ day.todos.length }}
      </span>
    </div>

    <!-- Mobile (<sm): compact priority dots -->
    <div class="mt-auto flex flex-wrap items-center gap-1 pt-1 sm:hidden">
      <span
        v-for="todo in day.todos.slice(0, 6)"
        :key="todo.id"
        class="h-1.5 w-1.5 rounded-full"
        :class="todoDotClass(todo)"
        :title="todo.title"
        aria-hidden="true"
      />
      <span
        v-if="day.todos.length > 6"
        class="text-[9px] leading-none font-bold text-slate-400 tabular-nums"
      >
        +{{ day.todos.length - 6 }}
      </span>
    </div>

    <!-- Desktop (sm+): full draggable chips -->
    <div class="hidden flex-1 flex-col gap-1 overflow-hidden sm:flex">
      <CalendarTodoChip
        v-for="todo in day.todos.slice(0, maxChips)"
        :key="todo.id"
        :todo="todo"
        @click="$emit('todoClick', todo)"
        @drag-start="drop.clearDrag"
        @drag-end="drop.clearDrag"
      />

      <button
        v-if="hiddenCount > 0"
        type="button"
        class="mt-auto flex items-center justify-between rounded-md px-1.5 py-1 text-left text-[11px] font-medium text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-200"
        @click.stop="$emit('showMore', day.date)"
      >
        <span>+{{ hiddenCount }} more</span>
        <ArrowRight class="h-2.5 w-2.5" :stroke-width="2.5" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Todo } from '../../types';
import type { CalendarDay } from './types';
import type { useCalendarDayDrop } from './composables/useCalendarDragDrop';
import { computed } from 'vue';
import { ArrowRight } from '@lucide/vue';
import { todoDotClass } from '../todos/todoFormat';
import CalendarTodoChip from './CalendarTodoChip.vue';

const props = withDefaults(
  defineProps<{
    day: CalendarDay;
    /** Drop handlers shared by the parent view. */
    drop: ReturnType<typeof useCalendarDayDrop>;
    /** Max chips before "+N more" (desktop). */
    maxChips?: number;
  }>(),
  { maxChips: 3 },
);

defineEmits<{
  (e: 'create', date: string): void;
  (e: 'showMore', date: string): void;
  (e: 'todoClick', todo: Todo): void;
}>();

const dayNumber = computed(() => new Date(props.day.date + 'T00:00:00').getDate());
const hiddenCount = computed(() => Math.max(0, props.day.todos.length - props.maxChips));

const cellClasses = computed(() => {
  const classes: string[] = [];
  if (props.drop.isDragTarget(props.day)) {
    classes.push('ring-2 ring-inset ring-violet-400 bg-violet-50/60 dark:bg-violet-950/30');
  }
  if (!props.day.isCurrentMonth) {
    classes.push('bg-slate-50/50 dark:bg-slate-950/40 opacity-40');
  }
  if (props.day.isWeekend && props.day.isCurrentMonth) {
    classes.push('bg-slate-50/30 dark:bg-slate-900/60');
  }
  return classes;
});

const dateClass = computed(() => {
  if (props.day.isToday) {
    return 'bg-violet-600 text-white font-bold shadow-md shadow-violet-500/30 dark:bg-violet-500 dark:text-white';
  }
  if (props.day.isWeekend) {
    return 'text-slate-400 dark:text-slate-500';
  }
  return 'text-slate-700 dark:text-slate-200';
});
</script>

<style scoped>
/* Subtle custom dark-mode hover background (sleek SaaS tone) */
.dark .dark\:bg-slate-850,
.dark .dark\:hover\:bg-slate-850:hover {
  background-color: rgb(24 32 47);
}
</style>
