<template>
  <div>
    <!-- Editable Year Header -->
    <div class="mb-4 flex items-center justify-center gap-2">
      <button
        type="button"
        class="grid h-7 w-7 place-items-center rounded-lg border border-gray-200 text-slate-500 transition hover:bg-gray-100 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700"
        aria-label="Previous year"
        @click="emit('yearChange', year - 1)"
      >
        <svg
          class="h-3.5 w-3.5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M15 18l-6-6 6-6" />
        </svg>
      </button>

      <div class="relative">
        <h2
          v-if="!editing"
          class="cursor-text rounded-md px-2 py-0.5 text-lg font-black text-slate-900 tabular-nums transition select-none hover:bg-gray-100 dark:text-white dark:hover:bg-slate-700"
          title="Click to jump to a year"
          @click="startEdit"
        >
          {{ year }}
        </h2>
        <input
          v-else
          ref="yearInput"
          v-model="draft"
          type="number"
          min="1900"
          max="9999"
          class="w-24 rounded-md border border-indigo-400 bg-white px-2 py-0.5 text-center text-lg font-black text-slate-900 tabular-nums shadow-sm focus:ring-2 focus:ring-indigo-400 focus:outline-none dark:border-indigo-500 dark:bg-slate-800 dark:text-white"
          @keydown.enter="commit"
          @keydown.esc="cancel"
          @blur="commit"
        />
      </div>

      <button
        type="button"
        class="grid h-7 w-7 place-items-center rounded-lg border border-gray-200 text-slate-500 transition hover:bg-gray-100 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700"
        aria-label="Next year"
        @click="emit('yearChange', year + 1)"
      >
        <svg
          class="h-3.5 w-3.5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M9 18l6-6-6-6" />
        </svg>
      </button>
    </div>

    <div class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 md:gap-5 lg:gap-6">
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
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import type { Todo } from '../../types'
import YearMonthCard from './YearMonthCard.vue'

const props = defineProps<{
  year: number
  todos: Todo[]
}>()

const emit = defineEmits<{
  (e: 'monthClick', month: number): void
  (e: 'dayClick', date: string): void
  (e: 'yearChange', year: number): void
}>()

const editing = ref(false)
const draft = ref('')
const yearInput = ref<HTMLInputElement | null>(null)

function startEdit() {
  draft.value = String(props.year)
  editing.value = true
  nextTick(() => {
    yearInput.value?.focus()
    yearInput.value?.select()
  })
}

function commit() {
  if (!editing.value) return
  const parsed = parseInt(draft.value, 10)
  editing.value = false
  if (Number.isNaN(parsed)) return
  const clamped = Math.min(9999, Math.max(1900, parsed))
  if (clamped !== props.year) {
    emit('yearChange', clamped)
  }
}

function cancel() {
  editing.value = false
}
</script>
