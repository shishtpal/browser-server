<template>
  <div class="flex items-center gap-2">
    <select
      :value="sortField"
      @change="$emit('update:sortField', ($event.target as HTMLSelectElement).value as any)"
      class="rounded-lg border border-gray-300 bg-gray-50 px-2 py-1.5 text-[11px] font-black text-slate-700 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
    >
      <option v-for="opt in options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
    </select>
    <button
      type="button"
      @click="$emit('toggle-dir')"
      class="rounded-lg bg-gray-100 px-2 py-1.5 text-[10px] font-black text-slate-500 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600"
    >
      {{ sortDir === 'asc' ? '↑' : '↓' }}
    </button>
  </div>
</template>

<script setup lang="ts">
import type { TodoSortField } from '../../composables/useTodoSort'

const options = [
  { value: 'position' as TodoSortField, label: 'Position' },
  { value: 'priority' as TodoSortField, label: 'Priority' },
  { value: 'due_date' as TodoSortField, label: 'Due date' },
  { value: 'created_at' as TodoSortField, label: 'Created' },
  { value: 'title' as TodoSortField, label: 'Title' },
]

interface Props {
  sortField: TodoSortField
  sortDir: 'asc' | 'desc'
}

defineProps<Props>()
defineEmits<{ 'update:sortField': [value: TodoSortField]; 'toggle-dir': [] }>()
</script>
