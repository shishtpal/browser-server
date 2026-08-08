<template>
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div class="flex items-center gap-2">
      <button
        type="button"
        class="grid h-8 w-8 place-items-center rounded-lg border border-gray-200 text-slate-600 transition hover:bg-gray-100 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700"
        aria-label="Previous period"
        @click="$emit('navigate', -1)"
      >
        <svg
          class="h-4 w-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M15 18l-6-6 6-6" />
        </svg>
      </button>
      <button
        type="button"
        class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-black text-slate-700 transition hover:bg-gray-100 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700"
        @click="$emit('today')"
      >
        Today
      </button>
      <button
        type="button"
        class="grid h-8 w-8 place-items-center rounded-lg border border-gray-200 text-slate-600 transition hover:bg-gray-100 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700"
        aria-label="Next period"
        @click="$emit('navigate', 1)"
      >
        <svg
          class="h-4 w-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M9 18l6-6-6-6" />
        </svg>
      </button>
      <h2 class="text-sm font-black text-slate-900 dark:text-white">{{ periodLabel }}</h2>
    </div>
    <div
      class="flex items-center gap-1 rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-slate-600 dark:bg-slate-800"
    >
      <button
        v-for="v in views"
        :key="v"
        type="button"
        class="rounded-md px-3 py-1.5 text-xs font-bold transition"
        :class="
          v === currentView
            ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-700 dark:text-white'
            : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
        "
        @click="$emit('changeView', v)"
      >
        {{ v.charAt(0).toUpperCase() + v.slice(1) }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CalendarView } from './types'

defineProps<{
  periodLabel: string
  currentView: CalendarView
}>()

defineEmits<{
  (e: 'navigate', dir: number): void
  (e: 'today'): void
  (e: 'changeView', view: CalendarView): void
}>()

const views: CalendarView[] = ['day', 'week', 'month', 'year']
</script>
