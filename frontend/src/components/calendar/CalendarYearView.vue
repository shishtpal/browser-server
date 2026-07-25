<template>
  <div class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4">
    <YearMonthCard
      v-for="monthIdx in 12"
      :key="monthIdx - 1"
      :year="year"
      :month-index="monthIdx - 1"
      :todos="todos"
      @month-click="emit('monthClick', $event)"
      @day-click="emit('dayClick', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import type { Todo } from '../../types'
import YearMonthCard from './YearMonthCard.vue'

defineProps<{
  year: number
  todos: Todo[]
}>()

const emit = defineEmits<{
  (e: 'monthClick', month: number): void
  (e: 'dayClick', date: string): void
}>()
</script>
