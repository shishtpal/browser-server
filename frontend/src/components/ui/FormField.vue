<template>
  <label class="block w-full">
    <span class="mb-1.5 block text-xs font-bold text-slate-700 dark:text-slate-300">
      {{ label }}
      <span v-if="required" class="text-rose-500">*</span>
    </span>
    <slot></slot>
    <p v-if="helpText" class="mt-1 text-[10px] text-slate-500 dark:text-slate-400">
      {{ helpText }}
    </p>
  </label>
</template>

<script setup lang="ts">
defineProps<{
  label: string;
  required?: boolean;
  helpText?: string;
}>();
</script>
