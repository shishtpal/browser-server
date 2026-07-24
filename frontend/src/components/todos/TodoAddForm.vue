<template>
  <form @submit.prevent="onSubmit" class="overflow-hidden rounded-2xl border border-indigo-100 bg-white shadow-sm transition-colors dark:border-indigo-500/20 dark:bg-slate-800/90">
    <div class="border-b border-indigo-100 bg-gradient-to-r from-indigo-50 via-white to-violet-50 px-4 py-3 dark:border-indigo-500/20 dark:from-indigo-950/40 dark:via-slate-800 dark:to-violet-950/30 sm:px-5">
      <div class="flex items-center gap-3">
        <span class="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-indigo-600 text-white shadow-md shadow-indigo-500/20">
          <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 5v14m-7-7h14" />
          </svg>
        </span>
        <div>
          <h2 class="text-sm font-black text-slate-900 dark:text-white">Create a new todo</h2>
          <p class="text-xs text-slate-500 dark:text-slate-400">Start with a title, then add details if you need them.</p>
        </div>
      </div>
    </div>

    <div class="p-4 sm:p-5">
      <label class="block">
        <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Task title</span>
        <input
          ref="titleInput"
          v-model="title"
          placeholder="What needs to be done?"
          required
          maxlength="200"
          class="w-full rounded-xl border border-gray-300 bg-gray-50 px-3.5 py-3 text-sm font-semibold text-slate-800 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:bg-white focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-900/50 dark:text-slate-100 dark:focus:bg-slate-900 dark:focus:ring-indigo-900/30"
        />
      </label>

      <div v-if="moreOpen" class="mt-4 grid gap-4 border-t border-gray-100 pt-4 dark:border-slate-700/80">
        <label class="block">
          <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Description <span class="font-semibold text-slate-400">(optional)</span></span>
          <textarea
            v-model="description"
            rows="3"
            placeholder="Add notes or context for this task..."
            class="w-full resize-y rounded-xl border border-gray-300 bg-gray-50 px-3.5 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:bg-white focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-900/50 dark:text-slate-200 dark:focus:bg-slate-900 dark:focus:ring-indigo-900/30"
          />
        </label>

        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Priority</span>
            <TodoPrioritySelect v-model="priority" />
          </div>
          <div>
            <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Due date</span>
            <TodoDueDatePicker v-model="dueDate" />
          </div>
          <div>
            <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Tags</span>
            <TodoTagInput
              v-model="tags"
              :suggestions="existingTags"
              @remove-tag="removeTag"
              @add-suggestion="addTag"
            />
          </div>
        </div>
      </div>

      <div class="mt-4 flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-between">
        <button
          type="button"
          @click="moreOpen = !moreOpen"
          class="inline-flex items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-xs font-black text-slate-500 transition hover:bg-gray-100 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-slate-700 dark:hover:text-slate-200"
          :aria-expanded="moreOpen"
        >
          <svg class="h-3.5 w-3.5 transition-transform" :class="{ 'rotate-180': moreOpen }" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="m6 9 6 6 6-6" />
          </svg>
          {{ moreOpen ? 'Hide task details' : 'Add priority, due date, and tags' }}
        </button>
        <button
          type="submit"
          class="inline-flex items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-indigo-600 to-violet-600 px-5 py-2.5 text-sm font-black text-white shadow-lg shadow-indigo-500/25 transition hover:-translate-y-0.5 hover:shadow-xl disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:translate-y-0"
          :disabled="!title.trim()"
        >
          Create todo
          <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="m9 18 6-6-6-6" />
          </svg>
        </button>
      </div>
    </div>
  </form>
</template>

<script setup lang="ts">
import { nextTick, ref, type PropType } from 'vue'
import type { TodoPriority } from '../../types'
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
const titleInput = ref<HTMLInputElement | null>(null)

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

function removeTag(tag: string) {
  tags.value = tags.value.filter(item => item !== tag)
}

function addTag(tag: string) {
  if (!tags.value.includes(tag)) tags.value = [...tags.value, tag]
}

function reset() {
  title.value = ''
  description.value = ''
  priority.value = 'medium'
  dueDate.value = null
  tags.value = []
  nextTick(() => titleInput.value?.focus())
}
</script>
