<template>
  <button
    type="button"
    class="inline-flex items-center gap-0.5 rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-black text-slate-600 transition hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400"
    :title="`Filter by #${tag}`"
    @click.stop="$emit('filter', tag)"
  >
    <Hash class="h-2.5 w-2.5 opacity-60" :stroke-width="2.5" aria-hidden="true" />
    {{ tag }}
  </button>
</template>

<script setup lang="ts">
import { Hash } from '@lucide/vue';

defineProps<{ tag: string }>();
defineEmits<{ filter: [tag: string] }>();
</script>
