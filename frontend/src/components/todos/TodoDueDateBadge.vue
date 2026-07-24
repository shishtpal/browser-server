<template>
  <span v-if="dueDate" :class="['inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] font-black', badgeClass]">
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { isOverdue, isDueToday, isDueThisWeek } from '../../composables/useTodoDueDate'

interface Props {
  dueDate: string | null
  completed?: boolean
}

const props = withDefaults(defineProps<Props>(), { completed: false })

const badgeClass = computed(() => {
  if (props.completed) return 'bg-gray-100 text-gray-500'
  if (isOverdue({ due_date: props.dueDate, completed: props.completed } as any)) return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400'
  if (isDueToday({ due_date: props.dueDate, completed: props.completed } as any)) return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
  if (isDueThisWeek({ due_date: props.dueDate, completed: props.completed } as any)) return 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-400'
  return 'bg-gray-100 text-gray-600'
})

const label = computed(() => {
  if (!props.dueDate) return ''
  if (isOverdue({ due_date: props.dueDate, completed: props.completed } as any)) return 'Overdue'
  if (isDueToday({ due_date: props.dueDate, completed: props.completed } as any)) return 'Today'
  if (isDueThisWeek({ due_date: props.dueDate, completed: props.completed } as any)) return 'This week'
  const d = new Date(props.dueDate)
  return d.toLocaleDateString()
})
</script>
