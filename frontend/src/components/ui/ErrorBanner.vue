<template>
  <div class="mb-4 rounded-xl border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700 shadow-sm transition-colors dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-400">
    {{ message }}
    <button v-if="onRetry" type="button" @click="onRetry" class="ml-2 underline decoration-red-300 underline-offset-4 transition-colors dark:decoration-red-800">Retry</button>
  </div>
</template>

<script setup lang="ts">
interface Props {
  message: string
  onRetry?: () => void
}

defineProps<Props>()
</script>
