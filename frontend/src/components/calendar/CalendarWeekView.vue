<template>
  <div class="flex-1 overflow-auto">
    <div
      class="grid grid-cols-7 gap-px rounded-xl border border-gray-200/80 bg-gray-200/80 dark:border-slate-700/80 dark:bg-slate-700/80"
    >
      <div
        v-for="day in days"
        :key="day.date"
        class="flex flex-col bg-white dark:bg-slate-800/90"
        :class="dayCellClass(day)"
        @dragover.prevent="onDragOver(day, $event)"
        @dragleave.prevent="onDragLeave(day, $event)"
        @drop.prevent="onDrop(day, $event)"
      >
        <div
          class="sticky top-0 z-10 flex items-center justify-between border-b border-gray-100 bg-gray-50 px-2 py-1.5 dark:border-slate-700/60 dark:bg-slate-800/80"
        >
          <span
            class="text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            {{ dayLabel(day.date) }}
          </span>
          <span
            class="text-[11px] font-bold"
            :class="
              day.isToday
                ? 'flex h-5 w-5 items-center justify-center rounded-full bg-indigo-600 text-white dark:bg-indigo-400 dark:text-slate-900'
                : 'text-slate-400 dark:text-slate-500'
            "
          >
            {{ dayNumber(day.date) }}
          </span>
        </div>
        <div class="flex min-h-[180px] flex-1 flex-col gap-1 p-1.5">
          <CalendarTodoChip
            v-for="todo in day.todos"
            :key="todo.id"
            :todo="todo"
            @click="emit('todoClick', todo)"
            @drag-start="onDragStart"
            @drag-end="onDragEnd"
          />
          <button
            v-if="day.todos.length === 0"
            type="button"
            class="mt-auto flex min-h-[40px] flex-1 items-center justify-center rounded-lg border border-dashed border-gray-300 text-slate-300 transition hover:border-indigo-400 hover:text-indigo-400 dark:border-slate-600 dark:text-slate-500 dark:hover:border-indigo-400 dark:hover:text-indigo-300"
            @click.stop="emit('click', day.date)"
          >
            <span class="text-[10px] font-bold">+ Add</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CalendarDay } from './types';
import type { Todo } from '../../types';
import CalendarTodoChip from './CalendarTodoChip.vue';
import { useCalendarDragDrop, todoFromPayload } from '../../composables/useCalendarDragDrop';

const props = defineProps<{
  days: CalendarDay[];
}>();

const emit = defineEmits<{
  (e: 'click', date: string): void;
  (e: 'todoClick', todo: Todo): void;
  (e: 'todoMove', payload: { todo: Todo; date: string }): void;
}>();

function dayLabel(dateStr: string) {
  return new Date(dateStr + 'T00:00:00').toLocaleDateString('en-US', { weekday: 'short' });
}

function dayNumber(dateStr: string) {
  return new Date(dateStr + 'T00:00:00').getDate();
}

const { dragOverDate, getDragPayload, hasCalendarPayload, isDropAllowed } = useCalendarDragDrop();

function dayCellClass(day: CalendarDay) {
  const classes: string[] = [];
  if (dragOverDate.value === day.date) {
    classes.push('ring-2 ring-inset ring-indigo-400 bg-indigo-50/60 dark:bg-indigo-950/30');
  }
  return classes.join(' ');
}

function onDragOver(day: CalendarDay, event: DragEvent) {
  if (!hasCalendarPayload(event.dataTransfer)) return;
  const payload = getDragPayload(event.dataTransfer);
  if (!isDropAllowed(payload, day.date)) return;
  event.dataTransfer!.dropEffect = 'move';
  dragOverDate.value = day.date;
}

function onDragLeave(day: CalendarDay, event: DragEvent) {
  if (dragOverDate.value !== day.date) return;
  const target = event.currentTarget as HTMLElement | null;
  const related = event.relatedTarget as Node | null;
  if (target && related && target.contains(related)) return;
  dragOverDate.value = null;
}

function onDrop(day: CalendarDay, event: DragEvent) {
  const payload = getDragPayload(event.dataTransfer);
  dragOverDate.value = null;
  if (!isDropAllowed(payload, day.date)) return;
  const todo = todoFromPayload(
    payload,
    props.days.flatMap((d) => d.todos),
  );
  if (!todo) return;
  emit('todoMove', { todo, date: day.date });
}

function onDragStart() {
  dragOverDate.value = null;
}

function onDragEnd() {
  dragOverDate.value = null;
}
</script>
