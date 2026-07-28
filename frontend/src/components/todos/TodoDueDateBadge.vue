<template>
  <span v-if="dueDate" :class="['inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] font-black', badgeClass]">
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { isOverdue, isDueToday, isDueThisWeek } from '../../composables/useTodoDueDate'
import type { TodoStatus } from '../../types'

interface Props {
  dueDate: string | null
  status?: TodoStatus
}

const props = withDefaults(defineProps<Props>(), { status: 'pending' })

const badgeClass = computed(() => {
  const todo = { start_date: props.dueDate, status: props.status } as any
  if (props.status === 'completed' || props.status === 'archived') return 'bg-gray-100 text-gray-500'
  if (isOverdue(todo)) return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400'
  if (isDueToday(todo)) return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
  if (isDueThisWeek(todo)) return 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-400'
  return 'bg-gray-100 text-gray-600'
})

const label = computed(() => {
  if (!props.dueDate) return ''
  const todo = { start_date: props.dueDate, status: props.status } as any
  if (isOverdue(todo)) return 'Overdue'
  if (isDueToday(todo)) return 'Today'
  if (isDueThisWeek(todo)) return 'This week'
  const d = new Date(props.dueDate)
  return d.toLocaleDateString()
})
</script>
