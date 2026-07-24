<template>
  <div class="flex items-center gap-2">
    <div class="h-1.5 w-24 overflow-hidden rounded-full bg-gray-200 dark:bg-slate-700">
      <div
        class="h-full rounded-full bg-indigo-500 transition-all"
        :style="{ width: pct + '%' }"
      />
    </div>
    <span class="text-[10px] font-black text-slate-500 dark:text-slate-400">
      {{ done }}/{{ total }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  done: number
  total: number
}

const props = defineProps<Props>()

const pct = computed(() => props.total > 0 ? Math.round((props.done / props.total) * 100) : 0)
</script>
