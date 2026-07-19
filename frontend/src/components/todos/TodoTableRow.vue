<template>
  <tr class="group transition hover:bg-indigo-50/60 dark:hover:bg-indigo-900/20">
    <td class="px-3 py-3">
      <button
        type="button"
        :aria-label="todo.completed ? 'Mark as active' : 'Mark as completed'"
        @click="emit('toggle', todo)"
        :class="['grid h-5 w-5 place-items-center rounded-full border-2 transition', todo.completed ? 'border-emerald-500 bg-emerald-500 text-white' : 'border-gray-300 text-transparent hover:border-indigo-400 dark:border-slate-600 dark:hover:border-indigo-400']"
      >
        <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
        </svg>
      </button>
    </td>
    <td v-if="editing" class="px-3 py-2" colspan="2">
      <div class="flex gap-2">
        <input v-model="localTitle" class="flex-1 rounded-lg border border-gray-300 bg-gray-50 px-3 py-1.5 text-sm font-semibold text-slate-700 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30" />
        <input v-model="localDescription" class="flex-[2] rounded-lg border border-gray-300 bg-gray-50 px-3 py-1.5 text-sm font-semibold text-slate-700 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30" />
      </div>
      <div class="mt-2 flex gap-2">
        <button type="button" @click="emit('saveEdit', todo, localTitle, localDescription)" class="rounded-lg bg-emerald-500 px-3 py-1 text-xs font-black text-white transition hover:bg-emerald-600">Save</button>
        <button type="button" @click="emit('cancelEdit')" class="rounded-lg bg-gray-100 px-3 py-1 text-xs font-black text-slate-600 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600">Cancel</button>
      </div>
    </td>
    <td v-else class="px-3 py-3">
      <div class="flex items-center gap-2">
        <button v-if="todo.screenshot_path" type="button" @click="emit('viewScreenshot', todo)" class="shrink-0 cursor-zoom-in transition hover:opacity-80" title="View screenshot">
          <img :src="screenshotUrl" class="h-6 w-10 rounded border border-gray-200 object-cover dark:border-slate-600" />
        </button>
        <span :class="['block truncate text-sm font-black', todo.completed ? 'text-slate-400 line-through dark:text-slate-500' : 'text-slate-900 dark:text-white']">{{ todo.title }}</span>
        <span v-if="todo.domain" class="shrink-0 rounded-full bg-indigo-100 px-2 py-0.5 text-[10px] font-bold text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400">{{ todo.domain }}</span>
      </div>
    </td>
    <td v-show="!editing" class="max-w-xs px-3 py-3">
      <span class="block truncate text-sm text-slate-500 transition-colors dark:text-slate-400">{{ todo.description || '—' }}</span>
    </td>
    <td v-show="!editing" class="px-3 py-3">
      <span class="whitespace-nowrap rounded-md bg-gray-100 px-2 py-1 text-[10px] font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400">{{ formatDate(todo.updated_at) }}</span>
    </td>
    <td v-show="!editing" class="px-3 py-3 text-right">
      <div class="flex justify-end gap-1">
        <button type="button" @click="emit('startEdit', todo)" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-slate-500 transition hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400">Edit</button>
        <button type="button" @click="confirmDelete" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-red-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">Delete</button>
      </div>
    </td>
  </tr>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { formatDate } from '../../lib/utils'
import { getScreenshotUrl } from '../../lib/api'
import type { Todo } from '../../types'

interface Props {
  todo: Todo
  editing: boolean
  initialTitle?: string
  initialDescription?: string
}

const props = defineProps<Props>()

const screenshotUrl = computed(() => props.todo.screenshot_path ? getScreenshotUrl(props.todo.id) : '')

const emit = defineEmits<{
  toggle: [todo: Todo]
  startEdit: [todo: Todo]
  saveEdit: [todo: Todo, title: string, description: string]
  cancelEdit: []
  delete: [id: number]
  viewScreenshot: [todo: Todo]
}>()

const localTitle = ref('')
const localDescription = ref('')

watch(() => props.editing, (val) => {
  if (val) {
    localTitle.value = props.initialTitle ?? props.todo.title
    localDescription.value = props.initialDescription ?? props.todo.description
  }
})

function confirmDelete() {
  if (window.confirm(`Delete "${props.todo.title}"?`)) {
    emit('delete', props.todo.id)
  }
}
</script>
