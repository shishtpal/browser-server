<template>
  <form @submit.prevent="onSubmit" class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90">
    <div class="flex items-center gap-2">
      <input
        v-model="title"
        placeholder="What needs to be done?"
        required
        class="flex-1 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
      />
      <button
        type="submit"
        class="rounded-lg bg-gradient-to-r from-indigo-600 to-violet-600 px-4 py-2 text-sm font-black text-white shadow-lg shadow-indigo-500/25 transition hover:-translate-y-0.5 hover:shadow-xl"
        :disabled="!title.trim()"
      >
        Add
      </button>
      <button
        type="button"
        @click="moreOpen = !moreOpen"
        class="rounded-lg bg-gray-100 px-3 py-2 text-xs font-black text-slate-500 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600"
      >
        {{ moreOpen ? 'Less' : 'More' }}
      </button>
    </div>

    <div v-if="moreOpen" class="mt-3 grid gap-3">
      <div class="grid grid-cols-2 gap-2">
        <input
          v-model="description"
          placeholder="Description (optional)"
          class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
        />
        <div class="flex items-center gap-2">
          <TodoPrioritySelect v-model="priority" />
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <TodoDueDatePicker v-model="dueDate" />
        <TodoTagInput v-model="tags" :suggestions="existingTags" />
      </div>
    </div>
  </form>
</template>

<script setup lang="ts">
import { ref, type PropType } from 'vue'
import type { Todo, TodoPriority } from '../../types'
import TodoPrioritySelect from './TodoPrioritySelect.vue'
import TodoDueDatePicker from './TodoDueDatePicker.vue'
import TodoTagInput from './TodoTagInput.vue'

const props = defineProps({
  existingTags: { type: Array as PropType<string[]>, default: () => [] },
})

const emit = defineEmits<{ 'submit': [data: { title: string; description?: string; priority?: TodoPriority; due_date?: string | null; tags: string[] }] }>()

const title = ref('')
const description = ref('')
const priority = ref<TodoPriority>('medium')
const dueDate = ref<string | null>(null)
const tags = ref<string[]>([])
const moreOpen = ref(false)

function onSubmit() {
  if (!title.value.trim()) return
  emit('submit', {
    title: title.value.trim(),
    description: description.value.trim() || undefined,
    priority: priority.value,
    due_date: dueDate.value,
    tags: tags.value,
  })
  reset()
}

function reset() {
  title.value = ''
  description.value = ''
  priority.value = 'medium'
  dueDate.value = null
  tags.value = []
}
</script>
