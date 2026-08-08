<template>
  <div class="flex h-full flex-col select-none">
    <!-- Weekday headers -->
    <div class="mb-2 grid grid-cols-7">
      <div
        v-for="name in dayNames"
        :key="name.full"
        class="py-1.5 text-center text-xs font-semibold tracking-wider text-slate-400 uppercase dark:text-slate-500"
      >
        <span class="hidden sm:inline">{{ name.full }}</span>
        <span class="sm:hidden">{{ name.short }}</span>
      </div>
    </div>

    <!-- Seamless 1px grid -->
    <div
      class="grid flex-1 grid-cols-7 gap-px overflow-hidden rounded-2xl border border-slate-200 bg-slate-200 shadow-sm dark:border-slate-800 dark:bg-slate-800"
      :style="{ gridTemplateRows: `repeat(${rowCount}, minmax(0, 1fr))` }"
    >
      <CalendarDayCell
        v-for="day in days"
        :key="day.date"
        :day="day"
        :drop="drop"
        @create="$emit('click', $event)"
        @show-more="$emit('showMore', $event)"
        @todo-click="$emit('todoClick', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { CalendarDay } from './types';
import type { Todo } from '../../types';
import CalendarDayCell from './CalendarDayCell.vue';
import { useCalendarDayDrop } from './composables/useCalendarDragDrop';

const props = defineProps<{
  days: CalendarDay[];
}>();

const emit = defineEmits<{
  (e: 'click', date: string): void;
  (e: 'showMore', date: string): void;
  (e: 'todoClick', todo: Todo): void;
  (e: 'todoMove', payload: { todo: Todo; date: string }): void;
}>();

const dayNames = [
  { full: 'Sun', short: 'S' },
  { full: 'Mon', short: 'M' },
  { full: 'Tue', short: 'T' },
  { full: 'Wed', short: 'W' },
  { full: 'Thu', short: 'T' },
  { full: 'Fri', short: 'F' },
  { full: 'Sat', short: 'S' },
];

const rowCount = computed(() => Math.ceil(props.days.length / 7));

// Shared drop logic (also used by the week view).
const drop = useCalendarDayDrop(
  computed(() => props.days),
  (payload) => emit('todoMove', payload),
);
</script>
