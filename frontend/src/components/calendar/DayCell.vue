<template>
  <div
    class="flex min-h-[72px] cursor-pointer flex-col rounded-xl border border-gray-200/80 bg-white p-1.5 transition-colors hover:bg-slate-50/50 dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:bg-slate-800/70"
    :class="cellClass"
    @click="onCellClick"
  >
    <div class="flex items-center justify-between">
      <span class="text-[11px] font-black" :class="dateClass">{{ dayNumber }}</span>
      <span v-if="todoCount > 0" class="text-[9px] font-bold text-slate-400 dark:text-slate-500">{{
        todoCount
      }}</span>
    </div>
    <div class="mt-1 flex flex-col gap-0.5 overflow-hidden">
      <CalendarTodoChip
        v-for="todo in visibleTodos"
        :key="todo.id"
        :todo="todo"
        @click.stop="emit('todoClick', todo)"
      />
      <button
        v-if="todoCount > 3"
        type="button"
        class="text-left text-[9px] font-bold text-indigo-500 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
        @click.stop="emit('showMore', day.date)"
      >
        +{{ todoCount - 3 }} more
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CalendarDay } from './types'
import type { Todo } from '../../types'
import CalendarTodoChip from './CalendarTodoChip.vue'

const props = defineProps<{
  day: CalendarDay
}>()

const emit = defineEmits<{
  (e: 'click', date: string): void
  (e: 'showMore', date: string): void
  (e: 'todoClick', todo: Todo): void
}>()

const dayNumber = computed(() => {
  return new Date(props.day.date + 'T00:00:00').getDate()
})

const visibleTodos = computed(() => {
  return props.day.todos.slice(0, 3)
})

const todoCount = computed(() => props.day.todos.length)

const cellClass = computed(() => {
  const classes: string[] = []
  if (!props.day.isCurrentMonth) {
    classes.push('opacity-40 dark:opacity-30')
  }
  if (props.day.isToday) {
    classes.push('ring-2 ring-indigo-500 dark:ring-indigo-400')
  }
  return classes.join(' ')
})

const dateClass = computed(() => {
  if (props.day.isToday)
    return 'flex h-5 w-5 items-center justify-center rounded-full bg-indigo-600 text-white dark:bg-indigo-400 dark:text-slate-900'
  if (props.day.isWeekend) return 'text-slate-500 dark:text-slate-400'
  return 'text-slate-700 dark:text-slate-200'
})

function onCellClick() {
  emit('click', props.day.date)
}
</script>
