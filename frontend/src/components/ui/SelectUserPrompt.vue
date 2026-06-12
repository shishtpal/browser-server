<template>
  <div v-if="show" class="mb-4 rounded-xl border border-dashed border-gray-300 bg-gray-50 p-6 text-center shadow-sm transition-colors dark:border-slate-600 dark:bg-slate-800/60">
    <h2 class="text-base font-black text-slate-800 transition-colors dark:text-slate-200">{{ title }}</h2>
    <p class="mt-1 text-xs text-slate-500 transition-colors dark:text-slate-400">Choose a workspace from the dropdown above.</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  title: string
  usersCount: number
  selectedUserId: number | null
}

const props = defineProps<Props>()

const show = computed(() => props.usersCount > 0 && !props.selectedUserId)
</script>
