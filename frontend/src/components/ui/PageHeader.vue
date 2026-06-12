<template>
  <div class="mb-4 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
    <div class="flex items-center gap-3">
      <div>
        <p class="mb-1 inline-flex rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.2em] transition-colors" :class="badgeClass">
          {{ badge }}
        </p>
        <h1 class="text-2xl font-black tracking-tight text-slate-900 transition-colors dark:text-white sm:text-3xl">{{ title }}</h1>
      </div>
      <div v-if="$slots.stats" class="flex items-center gap-2">
        <slot name="stats"></slot>
      </div>
    </div>
    <div v-if="$slots.actions" class="flex items-center gap-3">
      <slot name="actions"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  badge: string
  title: string
  color?: 'indigo' | 'cyan' | 'violet' | 'emerald' | 'amber'
}

const props = withDefaults(defineProps<Props>(), {
  color: 'indigo'
})

const badgeClass = computed(() => {
  const colors = {
    indigo: 'bg-indigo-50 text-indigo-600 dark:bg-indigo-900/20 dark:text-indigo-400',
    cyan: 'bg-cyan-50 text-cyan-700 dark:bg-cyan-900/20 dark:text-cyan-400',
    violet: 'bg-violet-50 text-violet-700 dark:bg-violet-900/20 dark:text-violet-400',
    emerald: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400',
    amber: 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
  }
  return colors[props.color]
})
</script>
