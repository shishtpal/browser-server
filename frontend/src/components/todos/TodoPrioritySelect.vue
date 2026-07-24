<template>
  <select
    :value="modelValue"
    @input="$emit('update:modelValue', ($event.target as HTMLSelectElement).value as TodoPriority)"
    class="rounded-lg border border-gray-300 bg-gray-50 px-2 py-1.5 text-[11px] font-black text-slate-700 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
  >
    <option v-for="p in PRIORITIES" :key="p.value" :value="p.value">{{ p.label }}</option>
  </select>
</template>

<script setup lang="ts">
import type { TodoPriority } from '../../types'

const PRIORITIES = [
  { value: 'low' as TodoPriority, label: 'Low' },
  { value: 'medium' as TodoPriority, label: 'Medium' },
  { value: 'high' as TodoPriority, label: 'High' },
  { value: 'urgent' as TodoPriority, label: 'Urgent' },
]

interface Props {
  modelValue: TodoPriority
}

defineProps<Props>()
defineEmits<{ 'update:modelValue': [value: TodoPriority] }>()
</script>
