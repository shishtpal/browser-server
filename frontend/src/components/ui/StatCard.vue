<template>
  <div class="rounded-xl px-3 py-2 text-center text-white shadow-lg transition-colors" :class="cardClass">
    <div class="text-sm font-black leading-none">{{ value }}</div>
    <div class="text-[10px] font-semibold leading-none mt-0.5" :class="labelClass">{{ label }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  value: string | number
  label: string
  variant?: 'dark' | 'primary' | 'secondary'
  color?: 'indigo' | 'cyan' | 'violet' | 'emerald' | 'amber' | 'rose'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'dark',
  color: 'indigo'
})

const cardClass = computed(() => {
  const darkShadows = {
    indigo: 'shadow-indigo-500/15',
    cyan: 'shadow-cyan-500/15',
    violet: 'shadow-violet-500/15',
    emerald: 'shadow-emerald-500/15',
    amber: 'shadow-amber-500/15',
    rose: 'shadow-rose-500/15',
  }
  const primaryBg = {
    indigo: 'bg-indigo-600',
    cyan: 'bg-cyan-500',
    violet: 'bg-violet-600',
    emerald: 'bg-emerald-500',
    amber: 'bg-amber-500',
    rose: 'bg-rose-600',
  }
  const primaryShadows = {
    indigo: 'shadow-indigo-500/20',
    cyan: 'shadow-cyan-500/20',
    violet: 'shadow-violet-500/20',
    emerald: 'shadow-emerald-500/20',
    amber: 'shadow-amber-500/20',
    rose: 'shadow-rose-500/20',
  }
  const variants = {
    dark: `bg-slate-900 ${darkShadows[props.color]} dark:bg-slate-950`,
    primary: `${primaryBg[props.color]} ${primaryShadows[props.color]}`,
    secondary: 'bg-emerald-500 shadow-emerald-500/20'
  }
  return variants[props.variant]
})

const labelClass = computed(() => {
  const primaryLabels = {
    indigo: 'text-indigo-100',
    cyan: 'text-cyan-100',
    violet: 'text-violet-100',
    emerald: 'text-emerald-100',
    amber: 'text-amber-100',
    rose: 'text-rose-100',
  }
  const variants = {
    dark: 'text-slate-300',
    primary: primaryLabels[props.color],
    secondary: 'text-emerald-50'
  }
  return variants[props.variant]
})
</script>
