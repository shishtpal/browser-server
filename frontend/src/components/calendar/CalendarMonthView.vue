<template>
  <div class="flex h-full flex-col select-none">
    <!-- Day name headers -->
    <div class="mb-2 grid grid-cols-7 gap-px">
      <div
        v-for="name in dayNames"
        :key="name"
        class="py-2 text-center text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500"
      >
        {{ name }}
      </div>
    </div>

    <!-- Seamless 1px Grid Container -->
    <div
      class="grid flex-1 grid-cols-7 gap-px overflow-hidden rounded-2xl border border-slate-200 bg-slate-200 dark:border-slate-800 dark:bg-slate-800 shadow-sm"
      :style="{ gridTemplateRows: `repeat(${rowCount}, minmax(0, 1fr))` }"
    >
      <div
        v-for="day in weekDays"
        :key="day.date"
        class="group relative flex min-h-[70px] sm:min-h-[110px] flex-col bg-white p-1.5 sm:p-2 transition-all duration-150 hover:bg-slate-50/80 dark:bg-slate-900 dark:hover:bg-slate-850 cursor-pointer"
        :class="cellClass(day)"
        @click="onCellClick(day)"
        @dragover.prevent="onDragOver(day, $event)"
        @dragleave.prevent="onDragLeave(day, $event)"
        @drop.prevent="onDrop(day, $event)"
      >
        <!-- Top Row: Date & Count Badge -->
        <div class="flex items-center justify-between mb-1">
          <span
            class="flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold transition-transform group-hover:scale-105"
            :class="dateClass(day)"
          >
            {{ day.dayNumber }}
          </span>

          <!-- Total tasks indicator (visible on desktop or when non-empty) -->
          <span
            v-if="day.todos.length > 0"
            class="rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-500 dark:bg-slate-800 dark:text-slate-400"
          >
            {{ day.todos.length }}
          </span>
        </div>

        <!-- MOBILE VIEW (< 640px): Compact Priority Dots -->
        <div class="flex sm:hidden flex-wrap gap-1 mt-auto pt-1">
          <span
            v-for="todo in day.todos.slice(0, 6)"
            :key="todo.id"
            class="h-1.5 w-1.5 rounded-full"
            :class="getPriorityDotClass(todo.priority)"
          />
          <span
            v-if="day.todos.length > 6"
            class="text-[9px] font-bold leading-none text-slate-400"
          >
            +{{ day.todos.length - 6 }}
          </span>
        </div>

        <!-- DESKTOP VIEW (>= 640px): Full Interactive Chips -->
        <div class="hidden sm:flex flex-1 flex-col gap-1 overflow-hidden">
          <CalendarTodoChip
            v-for="todo in day.todos.slice(0, 3)"
            :key="todo.id"
            :todo="todo"
            @click="emit('todoClick', todo)"
            @drag-start="onDragStart"
            @drag-end="onDragEnd"
          />

          <!-- "+X more" Button -->
          <button
            v-if="day.todos.length > 3"
            type="button"
            class="mt-auto flex items-center justify-between rounded-md px-1.5 py-1 text-left text-[11px] font-medium text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-200"
            @click.stop="emit('showMore', day.date)"
          >
            <span>+{{ day.todos.length - 3 }} more</span>
            <span class="text-[9px]">→</span>
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
import { useCalendarDragDrop, todoFromPayload } from '../../composables/useCalendarDragDrop'

const props = defineProps<{
  days: CalendarDay[]
}>()

const emit = defineEmits<{
  (e: 'click', date: string): void
  (e: 'showMore', date: string): void
  (e: 'todoClick', todo: Todo): void
  (e: 'todoMove', payload: { todo: Todo; date: string }): void
}>()

const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

const { dragOverDate, getDragPayload, hasCalendarPayload, isDropAllowed } = useCalendarDragDrop()

const weekDays = computed(() => {
  return props.days.map((day) => ({
    ...day,
    dayNumber: new Date(day.date + 'T00:00:00').getDate(),
  }))
})

const rowCount = computed(() => Math.ceil(props.days.length / 7))

function cellClass(day: CalendarDay & { dayNumber: number }) {
  const classes: string[] = []
  if (dragOverDate.value === day.date) {
    classes.push('ring-2 ring-inset ring-indigo-400 bg-indigo-50/60 dark:bg-indigo-950/30')
  }
  if (!day.isCurrentMonth) {
    classes.push('bg-slate-50/50 dark:bg-slate-950/40 opacity-40')
  }
  if (day.isWeekend && day.isCurrentMonth) {
    classes.push('bg-slate-50/30 dark:bg-slate-900/60')
  }
  return classes.join(' ')
}

function onDragOver(day: CalendarDay & { dayNumber: number }, event: DragEvent) {
  if (!hasCalendarPayload(event.dataTransfer)) return
  const payload = getDragPayload(event.dataTransfer)
  if (!isDropAllowed(payload, day.date)) return
  event.dataTransfer!.dropEffect = 'move'
  dragOverDate.value = day.date
}

function onDragLeave(day: CalendarDay & { dayNumber: number }, event: DragEvent) {
  if (dragOverDate.value !== day.date) return
  const target = event.currentTarget as HTMLElement | null
  const related = event.relatedTarget as Node | null
  if (target && related && target.contains(related)) return
  dragOverDate.value = null
}

function onDrop(day: CalendarDay & { dayNumber: number }, event: DragEvent) {
  const payload = getDragPayload(event.dataTransfer)
  dragOverDate.value = null
  if (!isDropAllowed(payload, day.date)) return
  const todo = todoFromPayload(payload, props.days.flatMap((d) => d.todos))
  if (!todo) return
  emit('todoMove', { todo, date: day.date })
}

function onDragStart() {
  dragOverDate.value = null
}

function onDragEnd() {
  dragOverDate.value = null
}

function dateClass(day: CalendarDay & { dayNumber: number }) {
  if (day.isToday) {
    return 'bg-indigo-600 text-white font-bold shadow-md shadow-indigo-500/30 dark:bg-indigo-500 dark:text-white'
  }
  if (day.isWeekend) {
    return 'text-slate-400 dark:text-slate-500'
  }
  return 'text-slate-700 dark:text-slate-200'
}

function getPriorityDotClass(priority: string) {
  const map: Record<string, string> = {
    low: 'bg-slate-400 dark:bg-slate-500',
    medium: 'bg-blue-500 dark:bg-blue-400',
    high: 'bg-amber-500 dark:bg-amber-400',
    urgent: 'bg-red-500 dark:bg-red-400',
  }
  return map[priority] || 'bg-slate-400'
}

function onCellClick(day: CalendarDay & { dayNumber: number }) {
  emit('click', day.date)
}
</script>

<style scoped>
/* Subtle custom dark mode background color to match sleek SaaS UIs */
.dark .dark\:bg-slate-850 {
  background-color: rgb(24 32 47);
}
</style>