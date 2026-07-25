<template>
  <button
    type="button"
    class="flex items-center gap-1 rounded-md px-1.5 py-0.5 text-left text-[10px] font-bold transition hover:ring-1 hover:ring-indigo-400 dark:hover:ring-indigo-300"
    :class="chipClass"
    @click.stop="emit('click', todo)"
  >
    <span class="h-1.5 w-1.5 shrink-0 rounded-full" :class="priorityDot"></span>
    <span class="truncate">{{ displayTitle }}</span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Todo } from '../../types'

const props = defineProps<{
  todo: Todo
}>()

const emit = defineEmits<{
  (e: 'click', todo: Todo): void
}>()

const displayTitle = computed(() => {
  return props.todo.title.length > 14 ? props.todo.title.slice(0, 14) + '…' : props.todo.title
})

const priorityDot = computed(() => {
  const map: Record<string, string> = {
    low: 'bg-slate-400 dark:bg-slate-500',
    medium: 'bg-blue-400 dark:bg-blue-300',
    high: 'bg-amber-500 dark:bg-amber-400',
    urgent: 'bg-red-500 dark:bg-red-400',
  }
  return map[props.todo.priority] || 'bg-slate-400'
})

const chipClass = computed(() => {
  const base = 'bg-gray-100 dark:bg-slate-700/80 text-slate-700 dark:text-slate-200'
  const completed = props.todo.status === 'completed' ? 'opacity-60 line-through' : ''
  return `${base} ${completed}`
})
</script>
