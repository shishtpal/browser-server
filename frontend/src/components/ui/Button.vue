<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :class="buttonClass"
    @click="onClick"
  >
    <span v-if="loading" class="flex items-center gap-1.5">
      <span class="inline-block h-3 w-3 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
      {{ loadingText }}
    </span>
    <slot v-else></slot>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost' | 'gradient-indigo' | 'gradient-cyan' | 'gradient-violet' | 'gradient-emerald' | 'gradient-amber'
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  loading?: boolean
  loadingText?: string
  type?: 'button' | 'submit' | 'reset'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  disabled: false,
  loading: false,
  loadingText: 'Loading...',
  type: 'button'
})

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()

const onClick = (e: MouseEvent) => {
  if (!props.disabled && !props.loading) {
    emit('click', e)
  }
}

const buttonClass = computed(() => {
  const base = 'font-black transition focus:outline-none focus:ring-4'
  
  const sizes = {
    sm: 'px-2.5 py-1.5 text-xs rounded-lg',
    md: 'px-4 py-2 text-sm rounded-lg',
    lg: 'px-5 py-2.5 text-base rounded-xl'
  }
  
  const variants = {
    primary: 'bg-slate-900 text-white shadow-lg hover:bg-slate-800 focus:ring-slate-200 dark:bg-white dark:text-slate-900 dark:hover:bg-gray-100 dark:focus:ring-white/10',
    secondary: 'bg-gray-100 text-slate-700 hover:bg-gray-200 focus:ring-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600',
    danger: 'bg-red-50 text-red-700 hover:bg-red-100 focus:ring-red-200 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30',
    ghost: 'text-slate-500 hover:bg-gray-100 focus:ring-gray-200 dark:text-slate-400 dark:hover:bg-slate-700',
    'gradient-indigo': 'bg-gradient-to-r from-indigo-600 to-violet-600 text-white shadow-lg shadow-indigo-500/25 hover:-translate-y-0.5 hover:shadow-xl focus:ring-indigo-200 dark:focus:ring-indigo-900/40',
    'gradient-cyan': 'bg-gradient-to-r from-cyan-500 to-blue-600 text-white shadow-lg shadow-cyan-500/25 hover:-translate-y-0.5 hover:shadow-xl focus:ring-cyan-200 dark:focus:ring-cyan-900/40',
    'gradient-violet': 'bg-gradient-to-r from-violet-600 to-fuchsia-600 text-white shadow-lg shadow-violet-500/25 hover:-translate-y-0.5 hover:shadow-xl focus:ring-violet-200 dark:focus:ring-violet-900/40',
    'gradient-emerald': 'bg-gradient-to-r from-emerald-500 to-teal-600 text-white shadow-lg shadow-emerald-500/25 hover:-translate-y-0.5 hover:shadow-xl focus:ring-emerald-200 dark:focus:ring-emerald-900/40',
    'gradient-amber': 'bg-gradient-to-r from-amber-500 to-orange-600 text-white shadow-lg shadow-orange-500/25 hover:-translate-y-0.5 hover:shadow-xl focus:ring-amber-200 dark:focus:ring-amber-900/40'
  }
  
  const disabledClass = props.disabled || props.loading ? 'disabled:cursor-not-allowed disabled:opacity-40' : ''
  
  return `${base} ${sizes[props.size]} ${variants[props.variant]} ${disabledClass}`
})
</script>
