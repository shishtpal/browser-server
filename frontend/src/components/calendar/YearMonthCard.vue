<template>
  <div
    :class="cardClasses"
    class="rounded-xl p-3 cursor-pointer transition-all hover:shadow-sm"
    @click="emit('monthClick', monthIndex)"
  >
    <div class="flex items-center justify-between mb-2">
      <h3 class="text-sm font-black" :class="labelClass">
        {{ label }}
        <span v-if="isCurrentMonth" class="ml-1.5 text-[9px] font-bold uppercase tracking-wide text-indigo-600 dark:text-indigo-400">Today</span>
      </h3>
      <span v-if="totalTodos > 0" :class="countClass">
        {{ totalTodos }} todo{{ totalTodos !== 1 ? 's' : '' }}
      </span>
    </div>

    <div class="grid grid-cols-7 gap-0.5">
      <div
        v-for="d in dayHeaders"
        :key="d"
        class="text-[8px] text-center text-slate-400 dark:text-slate-500 font-bold pb-0.5 select-none"
      >
        {{ d }}
      </div>

      <div
        v-for="(day, i) in dayData"
        :key="i"
        :class="dayClasses(day)"
        @click.stop="emit('dayClick', day.date)"
      >
        {{ day.day }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Todo } from '../../types'

const props = defineProps<{
  year: number
  monthIndex: number
  todos: Todo[]
}>()

const emit = defineEmits<{
  (e: 'monthClick', month: number): void
  (e: 'dayClick', date: string): void
}>()

const dayHeaders = ['S', 'M', 'T', 'W', 'T', 'F', 'S']

const monthStart = computed(() => new Date(props.year, props.monthIndex, 1))
const monthEnd = computed(() => new Date(props.year, props.monthIndex + 1, 0))

const isCurrentMonth = computed(() => {
  const now = new Date()
  return props.year === now.getFullYear() && props.monthIndex === now.getMonth()
})

const label = computed(() => {
  return monthStart.value.toLocaleDateString('en-US', { month: 'short' })
})

function formatDate(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

const monthTodos = computed(() => {
  const startStr = formatDate(monthStart.value)
  const endStr = formatDate(monthEnd.value)
  return props.todos.filter((t) => {
    if (!t.start_date) return false
    return t.start_date >= startStr && t.start_date <= endStr
  })
})

const totalTodos = computed(() => monthTodos.value.filter((t) => t.status !== 'completed').length)

const dayData = computed(() => {
  // Build a 6-week grid aligned to weekday
  const gridStart = new Date(monthStart.value)
  gridStart.setDate(gridStart.getDate() - gridStart.getDay())
  const gridEnd = new Date(monthEnd.value)
  gridEnd.setDate(gridEnd.getDate() + (6 - gridEnd.getDay()))

  const days: Array<{
    date: string
    day: number
    count: number
    isCurrentMonth: boolean
    isToday: boolean
    isWeekend: boolean
  }> = []

  const today = new Date()
  const current = new Date(gridStart)
  while (current <= gridEnd) {
    const dateStr = formatDate(current)
    const count = props.todos.filter((t) => t.start_date === dateStr && t.status !== 'completed').length
    days.push({
      date: dateStr,
      day: current.getDate(),
      count,
      isCurrentMonth: current.getMonth() === props.monthIndex,
      isToday: current.getFullYear() === today.getFullYear() && current.getMonth() === today.getMonth() && current.getDate() === today.getDate(),
      isWeekend: current.getDay() === 0 || current.getDay() === 6,
    })
    current.setDate(current.getDate() + 1)
  }
  return days
})

const cardClasses = computed(() => {
  if (isCurrentMonth.value) {
    return 'border-2 border-indigo-500 dark:border-indigo-400 bg-indigo-50 dark:bg-indigo-950/30 shadow-md'
  }
  if (totalTodos.value > 0) {
    return 'border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800/60 hover:border-indigo-400 dark:hover:border-indigo-600 hover:shadow-md'
  }
  return 'border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800/30 hover:border-slate-300 dark:hover:border-slate-600'
})

const labelClass = computed(() => {
  if (isCurrentMonth.value) return 'text-indigo-700 dark:text-indigo-300'
  if (totalTodos.value > 0) return 'text-slate-800 dark:text-slate-200'
  return 'text-slate-500 dark:text-slate-500'
})

const countClass = computed(() => {
  if (totalTodos.value >= 10) {
    return 'text-[10px] bg-indigo-100 text-indigo-700 dark:bg-indigo-900/50 dark:text-indigo-300 font-bold px-1.5 py-0.5 rounded-full'
  }
  if (totalTodos.value >= 5) {
    return 'text-[10px] bg-indigo-50 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400 font-bold px-1.5 py-0.5 rounded-full'
  }
  return 'text-[10px] bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300 font-bold px-1.5 py-0.5 rounded-full'
})

function dayClasses(day: { isToday: boolean; count: number; isWeekend: boolean; isCurrentMonth: boolean }) {
  const classes = ['aspect-square rounded-sm flex items-center justify-center text-[8px] font-bold transition-colors cursor-pointer']
  if (!day.isCurrentMonth) {
    classes.push('text-slate-300 dark:text-slate-600')
    return classes.join(' ')
  }
  if (day.isToday) {
    classes.push('bg-indigo-600 text-white ring-2 ring-indigo-400 ring-offset-1 dark:ring-offset-slate-800')
  } else if (day.count > 0) {
    classes.push(getHeatmapColor(day.count))
  } else if (day.isWeekend) {
    classes.push('text-slate-400 dark:text-slate-500')
  } else {
    classes.push('text-slate-600 dark:text-slate-400')
  }
  return classes.join(' ')
}

function getHeatmapColor(count: number): string {
  if (count >= 4) return 'bg-emerald-500 dark:bg-emerald-600/70 text-white'
  if (count >= 3) return 'bg-emerald-400 dark:bg-emerald-700/60 text-white'
  if (count >= 2) return 'bg-emerald-300 dark:bg-emerald-800/50 text-emerald-900 dark:text-emerald-200'
  return 'bg-emerald-200 dark:bg-emerald-900/40 text-emerald-800 dark:text-emerald-300'
}
</script>
