<template>
  <!-- The 7 columns never squeeze below a usable width; mobile scrolls sideways. -->
  <div class="flex-1 scrollbar-thin overflow-x-auto">
    <div
      class="grid min-w-[640px] grid-cols-7 gap-px rounded-xl border border-gray-200/80 bg-gray-200/80 dark:border-slate-700/80 dark:bg-slate-700/80"
    >
      <div
        v-for="day in days"
        :key="day.date"
        class="flex flex-col bg-white dark:bg-slate-800/90"
        :class="cellClass(day)"
        @dragover.prevent="drop.onDragOver(day, $event)"
        @dragleave.prevent="drop.onDragLeave(day, $event)"
        @drop.prevent="drop.onDrop(day, $event)"
      >
        <button
          type="button"
          class="sticky top-0 z-10 flex w-full items-center justify-between gap-1 border-b border-gray-100 bg-gray-50 px-2 py-1.5 transition hover:bg-violet-50 dark:border-slate-700/60 dark:bg-slate-800/80 dark:hover:bg-slate-700/60"
          :aria-label="`Add todo on ${day.date}`"
          @click="$emit('click', day.date)"
        >
          <span
            class="text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            {{ dayLabel(day.date) }}
          </span>
          <span
            class="text-[11px] font-bold tabular-nums"
            :class="
              day.isToday
                ? 'flex h-5 w-5 items-center justify-center rounded-full bg-violet-600 text-white dark:bg-violet-500'
                : 'text-slate-400 dark:text-slate-500'
            "
          >
            {{ dayNumber(day.date) }}
          </span>
        </button>

        <div class="flex min-h-[180px] flex-1 flex-col gap-1 p-1.5">
          <CalendarTodoChip
            v-for="todo in day.todos"
            :key="todo.id"
            :todo="todo"
            @click="$emit('todoClick', todo)"
            @drag-start="drop.clearDrag"
            @drag-end="drop.clearDrag"
          />
          <button
            v-if="day.todos.length === 0"
            type="button"
            class="mt-auto flex min-h-[40px] flex-1 items-center justify-center gap-1 rounded-lg border border-dashed border-gray-300 text-slate-300 transition hover:border-violet-400 hover:text-violet-500 dark:border-slate-600 dark:text-slate-500 dark:hover:border-violet-500 dark:hover:text-violet-400"
            @click.stop="$emit('click', day.date)"
          >
            <Plus class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
            <span class="text-[10px] font-bold">Add</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Plus } from '@lucide/vue';
import type { CalendarDay } from './types';
import type { Todo } from '../../types';
import CalendarTodoChip from './CalendarTodoChip.vue';
import { useCalendarDayDrop } from './composables/useCalendarDragDrop';

const props = defineProps<{
  days: CalendarDay[];
}>();

const emit = defineEmits<{
  (e: 'click', date: string): void;
  (e: 'todoClick', todo: Todo): void;
  (e: 'todoMove', payload: { todo: Todo; date: string }): void;
}>();

const drop = useCalendarDayDrop(
  computed(() => props.days),
  (payload) => emit('todoMove', payload),
);

function dayLabel(dateStr: string) {
  return new Date(dateStr + 'T00:00:00').toLocaleDateString('en-US', { weekday: 'short' });
}

function dayNumber(dateStr: string) {
  return new Date(dateStr + 'T00:00:00').getDate();
}

function cellClass(day: CalendarDay) {
  return drop.isDragTarget(day)
    ? 'ring-2 ring-inset ring-violet-400 bg-violet-50/60 dark:bg-violet-950/30'
    : '';
}
</script>

<style scoped>
.scrollbar-thin {
  scrollbar-width: thin;
}
</style>
