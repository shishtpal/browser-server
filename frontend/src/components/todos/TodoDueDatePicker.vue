<template>
  <div class="flex items-center gap-1">
    <input
      type="date"
      :value="modelValue"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value ? ($event.target as HTMLInputElement).value : null)"
      class="rounded-lg border border-gray-300 bg-gray-50 px-2 py-1.5 text-[11px] font-black text-slate-700 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
    />
    <button
      v-if="modelValue"
      type="button"
      @click="$emit('update:modelValue', null)"
      class="rounded-lg bg-gray-100 px-1.5 py-1 text-[10px] font-black text-slate-500 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600"
    >
      Clear
    </button>
  </div>
</template>

<script setup lang="ts">
interface Props {
  modelValue: string | null
}

defineProps<Props>()
defineEmits<{ 'update:modelValue': [value: string | null] }>()
</script>
