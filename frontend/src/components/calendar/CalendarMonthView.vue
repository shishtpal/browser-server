<template>
  <div class="flex h-full flex-col">
    <!-- Day name headers -->
    <div class="mb-1.5 grid grid-cols-7 gap-1.5">
      <div
        v-for="name in dayNames"
        :key="name"
        class="text-center text-[10px] font-black uppercase tracking-wider text-slate-500 dark:text-slate-400"
      >
        {{ name }}
      </div>
    </div>
    <!-- Day grid — fills remaining height -->
    <div class="grid flex-1 grid-cols-7 gap-1.5" :style="{ gridTemplateRows: `repeat(${rowCount}, minmax(0, 1fr))` }">
      <div
        v-for="day in weekDays"
        :key="day.date"
        class="flex flex-col overflow-hidden rounded-xl border border-gray-200/80 bg-white p-1.5 transition-colors hover:bg-slate-50/50 dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:bg-slate-800/70 cursor-pointer"
        :class="cellClass(day)"
        @click="onCellClick(day)"
      >
        <div class="flex items-center justify-between">
          <span class="text-[11px] font-black" :class="dateClass(day)">{{ day.dayNumber }}</span>
          <span v-if="day.todos.length > 0" class="text-[9px] font-bold text-slate-400 dark:text-slate-500">{{ day.todos.length }}</span>
        </div>
        <div class="mt-1 flex flex-1 flex-col gap-0.5 overflow-hidden">
          <CalendarTodoChip
            v-for="todo in day.todos.slice(0, 3)"
            :key="todo.id"
            :todo="todo"
            @click.stop="emit('todoClick', todo)"
          />
          <button
            v-if="day.todos.length > 3"
            type="button"
            class="text-left text-[9px] font-bold text-indigo-500 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
            @click.stop="emit('showMore', day.date)"
          >
            +{{ day.todos.length - 3 }} more
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CalendarDay } from './types'
import type { Todo } from '../../types'
import CalendarTodoChip from './CalendarTodoChip.vue'

const props = defineProps<{
  days: CalendarDay[]
}>()

const emit = defineEmits<{
  (e: 'click', date: string): void
  (e: 'showMore', date: string): void
  (e: 'todoClick', todo: Todo): void
}>()

const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

const weekDays = computed(() => {
  return props.days.map((day) => ({
    ...day,
    dayNumber: new Date(day.date + 'T00:00:00').getDate(),
  }))
})

const rowCount = computed(() => Math.ceil(props.days.length / 7))

function cellClass(day: CalendarDay & { dayNumber: number }) {
  const classes: string[] = []
  if (!day.isCurrentMonth) classes.push('opacity-40 dark:opacity-30')
  if (day.isToday) classes.push('ring-2 ring-indigo-500 dark:ring-indigo-400')
  return classes.join(' ')
}

function dateClass(day: CalendarDay & { dayNumber: number }) {
  if (day.isToday) return 'flex h-5 w-5 items-center justify-center rounded-full bg-indigo-600 text-white dark:bg-indigo-400 dark:text-slate-900'
  if (day.isWeekend) return 'text-slate-500 dark:text-slate-400'
  return 'text-slate-700 dark:text-slate-200'
}

function onCellClick(day: CalendarDay & { dayNumber: number }) {
  emit('click', day.date)
}
</script>
