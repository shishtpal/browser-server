<template>
  <div class="flex flex-col gap-4">
    <!-- Day summary header -->
    <div class="flex items-center gap-3 rounded-xl border border-gray-200/80 bg-white p-3 dark:border-slate-700/80 dark:bg-slate-800/90">
      <div
        class="flex h-12 w-12 items-center justify-center rounded-xl text-xl font-black"
        :class="day?.isToday ? 'bg-indigo-600 text-white dark:bg-indigo-400 dark:text-slate-900' : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'"
      >
        {{ dayNumber }}
      </div>
      <div>
        <div class="text-sm font-black text-slate-800 dark:text-slate-200">{{ dayName }}</div>
        <div class="text-xs text-slate-500 dark:text-slate-400">
          {{ fullDate }} · {{ todoCount }} todo{{ todoCount !== 1 ? 's' : '' }}
        </div>
      </div>
    </div>

    <!-- Timeline -->
    <div class="flex-1 overflow-auto">
      <div class="flex flex-col gap-px rounded-xl border border-gray-200/80 bg-gray-200/80 dark:border-slate-700/80 dark:bg-slate-700/80">
        <div
          v-for="hour in hours"
          :key="hour"
          class="flex border-b border-gray-200/80 bg-white last:border-b-0 dark:border-slate-700/80 dark:bg-slate-800/90"
        >
          <div class="w-16 shrink-0 border-r border-gray-100 px-2 py-2 text-right text-[10px] font-black text-slate-400 dark:border-slate-700/60 dark:text-slate-500">
            {{ formatHour(hour) }}
          </div>
          <div class="flex-1 min-h-[60px] p-1.5">
            <div v-if="hour === 0" class="flex flex-col gap-1.5">
              <button
                v-for="todo in dayTodos"
                :key="todo.id"
                type="button"
                class="rounded-lg px-3 py-2 text-left transition hover:ring-2 hover:ring-indigo-400"
                :class="todoClass(todo)"
                @click.stop="emit('todoClick', todo)"
              >
                <div class="flex items-center gap-1.5">
                  <span class="h-2 w-2 shrink-0 rounded-full" :class="priorityDot(todo.priority)"></span>
                  <span class="text-xs font-bold text-slate-700 dark:text-slate-200">{{ todo.title }}</span>
                </div>
                <div v-if="todo.description" class="mt-0.5 truncate text-[10px] text-slate-500 dark:text-slate-400">{{ todo.description }}</div>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CalendarDay } from './types'
import type { Todo } from '../../types'

const props = defineProps<{
  day: CalendarDay | undefined
}>()

const emit = defineEmits<{
  (e: 'todoClick', todo: Todo): void
}>()

const hours = Array.from({ length: 24 }, (_, i) => i)

const dayTodos = computed(() => props.day?.todos ?? [])
const todoCount = computed(() => dayTodos.value.length)

const dayNumber = computed(() => {
  if (!props.day) return ''
  return new Date(props.day.date + 'T00:00:00').getDate()
})

const dayName = computed(() => {
  if (!props.day) return ''
  return new Date(props.day.date + 'T00:00:00').toLocaleDateString('en-US', { weekday: 'long' })
})

const fullDate = computed(() => {
  if (!props.day) return ''
  return new Date(props.day.date + 'T00:00:00').toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })
})

function formatHour(hour: number) {
  if (hour === 0) return '12 AM'
  if (hour < 12) return `${hour} AM`
  if (hour === 12) return '12 PM'
  return `${hour - 12} PM`
}

function todoClass(todo: Todo) {
  const base = 'bg-gray-100 dark:bg-slate-700/80'
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
