<template>
  <div
    v-if="visible"
    class="flex shrink-0 justify-center border-t border-slate-100 px-4 py-2 dark:border-white/5"
  >
    <button
      class="flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-1.5 text-[0.85em] font-medium text-slate-600 transition hover:border-slate-300 hover:bg-slate-50 dark:border-white/10 dark:text-slate-400 dark:hover:border-white/20 dark:hover:bg-white/5"
      type="button"
      :disabled="disabled"
      @click="$emit('regenerate')"
    >
      <RotateCw class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
      Regenerate
    </button>
  </div>
</template>

<script setup lang="ts">
import { RotateCw } from '@lucide/vue';

defineProps<{
  visible: boolean;
  disabled: boolean;
}>();

defineEmits<{
  regenerate: [];
}>();
</script>
