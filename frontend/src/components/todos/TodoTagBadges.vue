<template>
  <div class="flex flex-wrap items-center gap-1">
    <span
      v-for="tag in tags"
      :key="tag"
      class="rounded-md bg-cyan-50 px-2 py-0.5 text-[10px] font-black text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400"
    >
      {{ tag }}
    </span>
  </div>
</template>

<script setup lang="ts">
interface Props {
  tags: string[]
}

defineProps<Props>()
</script>
