<template>
  <div class="flex items-center gap-2">
    <div class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 shadow-sm dark:border-slate-700 dark:bg-slate-800">
      <span class="text-[10px] font-black uppercase tracking-wider text-slate-500 dark:text-slate-400">Today</span>
      <span class="ml-1.5 text-sm font-black text-slate-900 dark:text-white">{{ stats.todayCount }}</span>
    </div>
    <div class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-1.5 shadow-sm dark:border-amber-500/20 dark:bg-amber-950/30">
      <span class="text-[10px] font-black uppercase tracking-wider text-amber-600 dark:text-amber-400">Overdue</span>
      <span class="ml-1.5 text-sm font-black text-amber-700 dark:text-amber-300">{{ stats.overdueCount }}</span>
    </div>
    <div class="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-1.5 shadow-sm dark:border-emerald-500/20 dark:bg-emerald-950/30">
      <span class="text-[10px] font-black uppercase tracking-wider text-emerald-600 dark:text-emerald-400">Done</span>
      <span class="ml-1.5 text-sm font-black text-emerald-700 dark:text-emerald-300">{{ stats.completedCount }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CalendarStats } from './types'

defineProps<{
  stats: CalendarStats
}>()
</script>
