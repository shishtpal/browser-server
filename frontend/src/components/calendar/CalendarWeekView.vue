<template>
  <div class="flex-1 overflow-auto">
    <div class="grid grid-cols-7 gap-px rounded-xl border border-gray-200/80 bg-gray-200/80 dark:border-slate-700/80 dark:bg-slate-700/80">
      <div
        v-for="day in days"
        :key="day.date"
        class="flex flex-col bg-white dark:bg-slate-800/90"
      >
        <div class="sticky top-0 z-10 flex items-center justify-between border-b border-gray-100 bg-gray-50 px-2 py-1.5 dark:border-slate-700/60 dark:bg-slate-800/80">
          <span class="text-[10px] font-black uppercase tracking-wider text-slate-500 dark:text-slate-400">{{ dayLabel(day.date) }}</span>
          <span
            class="text-[11px] font-bold"
            :class="day.isToday ? 'flex h-5 w-5 items-center justify-center rounded-full bg-indigo-600 text-white dark:bg-indigo-400 dark:text-slate-900' : 'text-slate-400 dark:text-slate-500'"
          >{{ dayNumber(day.date) }}</span>
        </div>
        <div class="flex flex-1 flex-col gap-1 p-1.5 min-h-[180px]">
          <button
            v-for="todo in day.todos"
            :key="todo.id"
            type="button"
            class="rounded-md px-2 py-1.5 text-left text-[11px] font-bold transition hover:ring-1 hover:ring-indigo-400"
            :class="todoClass(todo)"
            @click.stop="emit('todoClick', todo)"
          >
            <div class="flex items-center gap-1">
              <span class="h-1.5 w-1.5 shrink-0 rounded-full" :class="priorityDot(todo.priority)"></span>
              <span class="truncate">{{ todo.title }}</span>
            </div>
            <div class="flex items-center gap-1">
              <span v-if="todo.domain" class="text-[9px] text-violet-500 dark:text-violet-400">{{ todo.domain }}</span>
              <div v-if="todo.status === 'completed'" class="text-[9px] text-slate-400 line-through">Done</div>
            </div>
          </button>
          <button
            v-if="day.todos.length === 0"
            type="button"
            class="mt-auto flex-1 min-h-[40px] rounded-lg border border-dashed border-gray-300 text-slate-300 transition hover:border-indigo-400 hover:text-indigo-400 dark:border-slate-600 dark:text-slate-500 dark:hover:border-indigo-400 dark:hover:text-indigo-300 flex items-center justify-center"
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
import type { CalendarDay } from './types'
import type { Todo } from '../../types'

defineProps<{
  days: CalendarDay[]
}>()

const emit = defineEmits<{
  (e: 'click', date: string): void
  (e: 'todoClick', todo: Todo): void
}>()

function dayLabel(dateStr: string) {
  return new Date(dateStr + 'T00:00:00').toLocaleDateString('en-US', { weekday: 'short' })
}

function dayNumber(dateStr: string) {
  return new Date(dateStr + 'T00:00:00').getDate()
}

function todoClass(todo: Todo) {
  const base = 'bg-gray-100 dark:bg-slate-700/80 text-slate-700 dark:text-slate-200'
  return todo.status === 'completed' ? `${base} opacity-60` : base
}

function priorityDot(priority: string) {
  const map: Record<string, string> = {
    low: 'bg-slate-400 dark:bg-slate-500',
    medium: 'bg-blue-400 dark:bg-blue-300',
    high: 'bg-amber-500 dark:bg-amber-400',
    urgent: 'bg-red-500 dark:bg-red-400',
  }
  return map[priority] || 'bg-slate-400'
}
</script>
