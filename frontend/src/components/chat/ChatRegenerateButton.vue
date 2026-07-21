<template>
  <div v-if="visible" class="flex shrink-0 justify-center border-t border-slate-100 px-4 py-2 dark:border-white/5">
    <button
      class="flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-1.5 text-[0.85em] font-medium text-slate-600 transition hover:border-slate-300 hover:bg-slate-50 dark:border-white/10 dark:text-slate-400 dark:hover:border-white/20 dark:hover:bg-white/5"
      type="button"
      :disabled="disabled"
      @click="$emit('regenerate')"
    >
      <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
      Regenerate
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  visible: boolean
  disabled: boolean
}>()

defineEmits<{
  regenerate: []
}>()
</script>
