<template>
  <span :class="['inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] font-black', badgeClass]">
    <span :class="['h-1.5 w-1.5 rounded-full', accentClass]"></span>
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TodoPriority } from '../../types'

const PRIORITIES: Record<TodoPriority, { label: string; color: string; accent: string }> = {
  low: { label: 'Low', color: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400', accent: 'bg-emerald-500' },
  medium: { label: 'Medium', color: 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400', accent: 'bg-amber-500' },
  high: { label: 'High', color: 'bg-orange-50 text-orange-700 dark:bg-orange-900/20 dark:text-orange-400', accent: 'bg-orange-500' },
  urgent: { label: 'Urgent', color: 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400', accent: 'bg-red-500' },
}

interface Props {
  priority: TodoPriority
}

const props = defineProps<Props>()

const config = computed(() => PRIORITIES[props.priority] || PRIORITIES.medium)

const badgeClass = computed(() => config.value.color)
const accentClass = computed(() => config.value.accent)
const label = computed(() => config.value.label)
</script>
