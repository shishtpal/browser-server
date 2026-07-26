<template>
  <div
    :class="cardClasses"
    class="rounded-2xl p-3 sm:p-4 cursor-pointer transition-all duration-200 group hover:-translate-y-0.5"
    @click="emit('monthClick', monthIndex)"
  >
    <!-- Header: Month Name & Badges -->
    <div class="flex items-center justify-between mb-3">
      <div class="flex items-center gap-2">
        <h3 class="text-sm font-semibold tracking-tight" :class="labelClass">
          {{ label }}
        </h3>
        <span 
          v-if="isCurrentMonth" 
          class="flex h-4 items-center rounded-full bg-indigo-600 px-1.5 text-[9px] font-bold uppercase tracking-wide text-white shadow-sm shadow-indigo-500/30 dark:bg-indigo-500"
        >
          Today
        </span>
      </div>
      
      <span 
        v-if="totalTodos > 0" 
        class="text-[10px] font-semibold tabular-nums px-1.5 py-0.5 rounded-full" 
        :class="countClass"
      >
        {{ totalTodos }}
      </span>
    </div>

    <!-- Mini Calendar Grid -->
    <div class="grid grid-cols-7 gap-y-1 gap-x-0.5">
      <div
        v-for="d in dayHeaders"
        :key="d"
        class="text-[9px] text-center text-slate-400 dark:text-slate-500 font-medium select-none pb-0.5"
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

// PERFORMANCE OPTIMIZATION: Pre-calculate counts to avoid O(N*M) nested loops
const todoCountMap = computed(() => {
  const map = new Map<string, number>()
  props.todos.forEach((t) => {
    if (t.start_date && t.status !== 'completed') {
      map.set(t.start_date, (map.get(t.start_date) || 0) + 1)
    }
  })
  return map
})

const totalTodos = computed(() => {
  const startStr = formatDate(monthStart.value)
  const endStr = formatDate(monthEnd.value)
  return props.todos.filter((t) => {
    if (!t.start_date || t.status === 'completed') return false
    return t.start_date >= startStr && t.start_date <= endStr
  }).length
})

const dayData = computed(() => {
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
    days.push({
      date: dateStr,
      day: current.getDate(),
      count: todoCountMap.value.get(dateStr) || 0, // Use optimized map
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
    return 'border-2 border-indigo-500/80 dark:border-indigo-400/60 bg-indigo-50/50 dark:bg-indigo-950/20 shadow-md shadow-indigo-200/50 dark:shadow-indigo-900/30'
  }
  if (totalTodos.value > 0) {
    return 'border border-slate-200 dark:border-slate-700/80 bg-white dark:bg-slate-900/50 hover:border-indigo-300 dark:hover:border-indigo-700 hover:shadow-lg hover:shadow-slate-200/50 dark:hover:shadow-slate-900/50'
  }
  return 'border border-slate-200 dark:border-slate-800/80 bg-white dark:bg-slate-900/30 hover:border-slate-300 dark:hover:border-slate-700'
})

const labelClass = computed(() => {
  if (isCurrentMonth.value) return 'text-indigo-700 dark:text-indigo-300'
  if (totalTodos.value > 0) return 'text-slate-800 dark:text-slate-200'
  return 'text-slate-500 dark:text-slate-500'
})

const countClass = computed(() => {
  if (totalTodos.value >= 10) {
    return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/60 dark:text-indigo-300'
  }
  if (totalTodos.value >= 5) {
    return 'bg-indigo-50 text-indigo-600 dark:bg-indigo-900/40 dark:text-indigo-400'
  }
  return 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400'
})

function dayClasses(day: { isToday: boolean; count: number; isWeekend: boolean; isCurrentMonth: boolean }) {
  const base = 'aspect-square rounded-md flex items-center justify-center text-[10px] font-medium transition-colors cursor-pointer'
  
  if (!day.isCurrentMonth) {
    return `${base} text-slate-300 dark:text-slate-700`
  }
  
  if (day.isToday) {
    return `${base} bg-indigo-600 text-white font-bold shadow-sm shadow-indigo-500/40 dark:bg-indigo-500`
  } 
  
  if (day.count > 0) {
    return `${base} ${getHeatmapColor(day.count)}`
  } 
  
  if (day.isWeekend) {
    return `${base} text-slate-400 dark:text-slate-500 hover:bg-slate-50 dark:hover:bg-slate-800/50`
  } 
  
  return `${base} text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800/50`
}

// Cohesive Indigo Heatmap matching the app's primary theme
function getHeatmapColor(count: number): string {
  if (count >= 4) return 'bg-indigo-500 text-white dark:bg-indigo-600 font-bold'
  if (count >= 3) return 'bg-indigo-200 text-indigo-900 dark:bg-indigo-700/60 text-white'
  if (count >= 2) return 'bg-indigo-100 text-indigo-800 dark:bg-indigo-800/50 text-indigo-200'
  return 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 text-indigo-400'
}
</script>