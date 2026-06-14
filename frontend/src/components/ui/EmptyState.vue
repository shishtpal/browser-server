<template>
  <div class="rounded-xl border border-dashed border-gray-300 bg-gray-50 p-8 text-center shadow-sm transition-colors dark:border-slate-600 dark:bg-slate-800/60">
    <div class="mx-auto grid h-10 w-10 place-items-center rounded-xl transition-colors" :class="iconBgClass">
      <svg class="h-5 w-5 transition-colors" :class="iconColorClass" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path v-if="icon === 'bookmark'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
        <path v-else-if="icon === 'search'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        <path v-else-if="icon === 'clock'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        <path v-else-if="icon === 'lock'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
        <path v-else-if="icon === 'users'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        <path v-else-if="icon === 'chart'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 20V10M18 20V4M6 20v-4" />
        <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002-2h2a2 2 0 002 2M9 5h6" />
      </svg>
    </div>
    <h2 class="mt-3 text-base font-black text-slate-800 transition-colors dark:text-slate-200">{{ title }}</h2>
    <p class="mt-1 text-xs text-slate-500 transition-colors dark:text-slate-400">{{ description }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  title: string
  description: string
  icon?: 'bookmark' | 'search' | 'clock' | 'lock' | 'users' | 'chart' | 'default'
  color?: 'indigo' | 'cyan' | 'violet' | 'emerald' | 'amber' | 'rose'
}

const props = withDefaults(defineProps<Props>(), {
  icon: 'default',
  color: 'indigo'
})

const bgColors: Record<string, string> = {
  indigo: 'bg-indigo-50 dark:bg-indigo-900/20',
  cyan: 'bg-cyan-50 dark:bg-cyan-900/20',
  violet: 'bg-violet-50 dark:bg-violet-900/20',
  emerald: 'bg-emerald-50 dark:bg-emerald-900/20',
  amber: 'bg-amber-50 dark:bg-amber-900/20',
  rose: 'bg-rose-50 dark:bg-rose-900/20',
}

const textColors: Record<string, string> = {
  indigo: 'text-indigo-500 dark:text-indigo-400',
  cyan: 'text-cyan-500 dark:text-cyan-400',
  violet: 'text-violet-500 dark:text-violet-400',
  emerald: 'text-emerald-500 dark:text-emerald-400',
  amber: 'text-amber-500 dark:text-amber-400',
  rose: 'text-rose-500 dark:text-rose-400',
}

const iconBgClass = computed(() => bgColors[props.color] ?? bgColors.indigo)
const iconColorClass = computed(() => textColors[props.color] ?? textColors.indigo)
</script>
