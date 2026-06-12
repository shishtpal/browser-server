<template>
  <input
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :required="required"
    :disabled="disabled"
    :class="inputClass"
    @input="onInput"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  modelValue: string
  type?: 'text' | 'url' | 'email' | 'password' | 'number'
  placeholder?: string
  required?: boolean
  disabled?: boolean
  color?: 'indigo' | 'cyan' | 'violet' | 'emerald' | 'amber'
  flex?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  placeholder: '',
  required: false,
  disabled: false,
  color: 'indigo',
  flex: false
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const onInput = (e: Event) => {
  const target = e.target as HTMLInputElement
  emit('update:modelValue', target.value)
}

const inputClass = computed(() => {
  const base = 'rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:outline-none focus:ring-4 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500'
  
  const colors = {
    indigo: 'focus:border-indigo-400 focus:ring-indigo-100 dark:focus:ring-indigo-900/30',
    cyan: 'focus:border-cyan-400 focus:ring-cyan-100 dark:focus:ring-cyan-900/30',
    violet: 'focus:border-violet-400 focus:ring-violet-100 dark:focus:ring-violet-900/30',
    emerald: 'focus:border-emerald-400 focus:ring-emerald-100 dark:focus:ring-emerald-900/30',
    amber: 'focus:border-amber-400 focus:ring-amber-100 dark:focus:ring-amber-900/30'
  }
  
  const flexClass = props.flex ? 'min-w-0 flex-1' : 'w-full'
  
  return `${base} ${colors[props.color]} ${flexClass}`
})
</script>
