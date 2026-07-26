<template>
  <tr class="group transition hover:bg-indigo-50/60 dark:hover:bg-indigo-900/20">
    <td class="w-14 px-3 py-3">
      <div class="flex items-center gap-1.5">
        <button
          type="button"
          class="drag-handle cursor-grab active:cursor-grabbing text-slate-400 transition hover:text-slate-600"
          title="Drag to reorder"
        >
          <svg class="h-3.5 w-3.5" fill="currentColor" viewBox="0 0 24 24">
            <circle cx="9" cy="6" r="1.5" />
            <circle cx="15" cy="6" r="1.5" />
            <circle cx="9" cy="12" r="1.5" />
            <circle cx="15" cy="12" r="1.5" />
            <circle cx="9" cy="18" r="1.5" />
            <circle cx="15" cy="18" r="1.5" />
          </svg>
        </button>
        <button
          type="button"
          :aria-label="todo.status === 'archived' ? 'Archived todo' : todo.status === 'completed' ? 'Mark as active' : 'Mark as completed'"
          :disabled="todo.status === 'archived'"
          @click="$emit('toggle', todo)"
          :class="['grid h-5 w-5 place-items-center rounded-full border-2 transition disabled:cursor-default', todo.status === 'completed' ? 'border-emerald-500 bg-emerald-500 text-white' : 'border-gray-300 text-transparent hover:border-indigo-400 dark:border-slate-600 dark:hover:border-indigo-400']"
        >
          <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
          </svg>
        </button>
      </div>
    </td>
    <td class="max-w-xs px-3 py-3">
      <div class="flex items-center gap-2">
        <button v-if="todo.screenshot_path" type="button" @click="$emit('viewScreenshot', todo)" class="shrink-0 cursor-zoom-in transition hover:opacity-80" title="View screenshot">
          <img :src="screenshotUrl" class="h-6 w-10 rounded border border-gray-200 object-cover dark:border-slate-600" />
        </button>
        <span :class="['block truncate text-sm font-black', todo.status === 'completed' ? 'text-slate-400 line-through dark:text-slate-500' : 'text-slate-900 dark:text-white']">{{ todo.title }}</span>
        <span v-if="todo.pinned" class="inline-flex items-center gap-1 rounded-full bg-indigo-50 px-1.5 py-0.5 text-[10px] font-black text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300" title="Pinned todo">
          <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m14 4 6 6-3 1-4 4-1 5-3-3-3-3 5-1 4-4 1-3Z" /></svg>
          Pinned
        </span>
        <TodoPriorityBadge :priority="(todo.priority as any)" />
      </div>
    </td>
    <td class="max-w-xs px-3 py-3">
      <span class="block truncate text-sm text-slate-500 transition-colors dark:text-slate-400">{{ todo.description || '—' }}</span>
    </td>
    <td class="px-3 py-3">
      <div class="flex flex-wrap items-center gap-1">
        <TodoDueDateBadge v-if="todo.start_date" :due-date="todo.start_date" :status="todo.status" />
      </div>
    </td>
    <td class="px-3 py-3">
      <div class="flex flex-wrap items-center gap-1">
        <TodoTagBadges :tags="(todo.tags || [])" />
      </div>
    </td>
    <td class="px-3 py-3">
      <span class="whitespace-nowrap rounded-md bg-gray-100 px-2 py-1 text-[10px] font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400">{{ formatDate(todo.updated_at) }}</span>
    </td>
    <td class="px-3 py-3">
      <button
        type="button"
        @click="$emit('toggle-expand', todo.id)"
        class="text-[10px] font-black text-slate-500 transition hover:text-indigo-600 dark:text-slate-400 dark:hover:text-indigo-400"
      >
        {{ (todo.subtasks?.length || 0) }} subtask{{ (todo.subtasks?.length || 0) !== 1 ? 's' : '' }}
      </button>
    </td>
    <td class="px-3 py-3 text-right">
      <div class="flex justify-end gap-1">
        <button type="button" @click="$emit('toggle-pin', todo)" class="rounded-lg px-2 py-1.5 text-xs font-black transition" :class="todo.pinned ? 'bg-indigo-50 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300' : 'text-slate-500 hover:bg-indigo-50 hover:text-indigo-600 dark:text-slate-400 dark:hover:bg-indigo-500/10'">{{ todo.pinned ? 'Unpin' : 'Pin' }}</button>
        <button v-if="todo.status === 'archived'" type="button" @click="$emit('restore', todo)" class="rounded-lg px-2 py-1.5 text-xs font-black text-emerald-600 transition hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/20">Restore</button>
        <button v-else-if="todo.status === 'completed'" type="button" @click="$emit('archive', todo)" class="rounded-lg px-2 py-1.5 text-xs font-black text-amber-600 transition hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/20">Archive</button>
        <button v-if="todo.status !== 'archived'" type="button" @click="$emit('startEdit', todo)" class="rounded-lg px-2 py-1.5 text-xs font-black text-slate-500 transition hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400">Edit</button>
        <button type="button" @click="$emit('delete', todo.id)" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-red-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">Delete</button>
      </div>
    </td>
  </tr>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatDate } from '../../lib/utils'
import { getScreenshotUrl } from '../../lib/api'
import type { Todo } from '../../types'
import TodoPriorityBadge from './TodoPriorityBadge.vue'
import TodoDueDateBadge from './TodoDueDateBadge.vue'
import TodoTagBadges from './TodoTagBadges.vue'

interface Props {
  todo: Todo
  expanded?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  toggle: [todo: Todo]
  'toggle-pin': [todo: Todo]
  archive: [todo: Todo]
  restore: [todo: Todo]
  startEdit: [todo: Todo]
  delete: [id: number]
  viewScreenshot: [todo: Todo]
  'toggle-expand': [id: number]
}>()

const screenshotUrl = computed(() => props.todo.screenshot_path ? getScreenshotUrl(props.todo.id) : '')
</script>
