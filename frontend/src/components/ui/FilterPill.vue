<template>
  <button
    type="button"
    @click="onClick"
    :class="pillClass"
  >
    <slot></slot>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  active?: boolean
  variant?: 'default' | 'primary' | 'tag'
}

const props = withDefaults(defineProps<Props>(), {
  active: false,
  variant: 'default'
})

const emit = defineEmits<{
  click: []
}>()

const onClick = () => {
  emit('click')
}

const pillClass = computed(() => {
  if (props.variant === 'tag') {
    return props.active
      ? 'rounded-md bg-cyan-500 px-2 py-0.5 text-[10px] font-black text-white shadow-md transition'
      : 'rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-black text-slate-600 transition hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400'
  }
  
  return props.active
    ? 'rounded-lg px-2.5 py-1.5 text-[10px] font-black transition bg-slate-900 text-white shadow-lg shadow-slate-900/20 focus:ring-slate-200 dark:bg-white dark:text-slate-900 dark:shadow-white/10 dark:focus:ring-white/10'
    : 'rounded-lg px-2.5 py-1.5 text-[10px] font-black transition bg-white text-slate-600 hover:bg-gray-100 focus:ring-gray-200 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700 dark:focus:ring-white/10'
})
</script>
