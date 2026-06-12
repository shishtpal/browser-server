<template>
  <div class="flex justify-center py-16">
    <div class="h-8 w-8 animate-spin rounded-full border-4 border-t-transparent" :class="colorClass"></div>
    <span class="ml-3 self-center text-sm font-semibold text-slate-600 transition-colors dark:text-slate-400">{{ message }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  message?: string
  color?: 'indigo' | 'cyan' | 'violet' | 'emerald' | 'amber'
}

const props = withDefaults(defineProps<Props>(), {
  message: 'Loading...',
  color: 'indigo'
})

const colorClass = computed(() => {
  const colors = {
    indigo: 'border-indigo-500',
    cyan: 'border-cyan-500',
    violet: 'border-violet-500',
    emerald: 'border-emerald-500',
    amber: 'border-amber-500'
  }
  return colors[props.color]
})
</script>
