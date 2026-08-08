<template>
  <div class="space-y-2">
    <label
      class="flex items-center gap-1.5 text-xs font-semibold text-slate-600 dark:text-slate-400"
    >
      <PenLine class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
      Type your answer:
    </label>
    <textarea
      :value="modelValue"
      rows="3"
      placeholder="Enter answer text..."
      class="w-full rounded-xl border border-slate-300 bg-white p-3 text-sm text-slate-900 shadow-sm transition focus:border-violet-500 focus:ring-2 focus:ring-violet-500/20 focus:outline-none dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
      @input="$emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
    ></textarea>
  </div>
</template>

<script setup lang="ts">
import { PenLine } from '@lucide/vue';

defineProps<{ modelValue: string }>();
defineEmits<{ 'update:modelValue': [value: string] }>();
</script>
