<template>
  <select
    :id="id"
    :value="modelValue"
    class="rounded-xl border border-gray-300 bg-gray-50 px-3 py-2 text-xs font-semibold text-slate-700 shadow-sm transition focus:outline-none focus:ring-4 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
    :class="focusClass"
    @change="onChange"
  >
    <option :value="null">All users</option>
    <option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }}</option>
  </select>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  id: string
  modelValue: number | null
  users: Array<{ id: number; username: string }>
  color?: 'indigo' | 'cyan' | 'violet' | 'emerald' | 'amber'
}

const props = withDefaults(defineProps<Props>(), {
  color: 'indigo'
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const onChange = (e: Event) => {
  const target = e.target as HTMLSelectElement
  emit('update:modelValue', target.value && target.value !== 'null' ? Number(target.value) : null)
}

const focusClass = computed(() => {
  const colors = {
    indigo: 'focus:border-indigo-400 focus:ring-indigo-100 dark:focus:ring-indigo-900/30',
    cyan: 'focus:border-cyan-400 focus:ring-cyan-100 dark:focus:ring-cyan-900/30',
    violet: 'focus:border-violet-400 focus:ring-violet-100 dark:focus:ring-violet-900/30',
    emerald: 'focus:border-emerald-400 focus:ring-emerald-100 dark:focus:ring-emerald-900/30',
    amber: 'focus:border-amber-400 focus:ring-amber-100 dark:focus:ring-amber-900/30'
  }
  return colors[props.color]
})
</script>
