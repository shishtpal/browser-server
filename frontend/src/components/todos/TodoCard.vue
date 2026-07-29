<template>
  <li :class="['group rounded-xl border p-3 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md', todo.status === 'in_progress' ? 'border-blue-300 bg-blue-50/40 dark:border-blue-500/40 dark:bg-blue-900/10 hover:border-blue-400' : 'border-gray-200/80 bg-white hover:border-indigo-200 dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-indigo-500/30']">
    <div class="flex items-start gap-3">
      <button
        type="button"
        :disabled="todo.status === 'archived'"
        @click="$emit('toggle', todo)"
        class="mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded-full border-2 transition disabled:cursor-default"
        :class="todo.status === 'completed' ? 'border-emerald-500 bg-emerald-500 text-white' : todo.status === 'in_progress' ? 'border-blue-500 bg-blue-500 text-white' : 'border-gray-300 text-transparent hover:border-indigo-400 dark:border-slate-600 dark:hover:border-indigo-400'"
      >
        <svg v-if="todo.status === 'completed'" class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
        </svg>
        <svg v-else-if="todo.status === 'in_progress'" class="h-3 w-3 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24" style="animation-duration: 2s">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M12 3a9 9 0 019 9" />
        </svg>
      </button>
    <div class="min-w-0 flex-1">
      <div>
        <div class="flex items-center gap-2">
          <button v-if="todo.screenshot_path" type="button" @click="$emit('viewScreenshot', todo)" class="shrink-0 cursor-zoom-in transition hover:opacity-80" title="View screenshot">
            <img :src="screenshotUrl" class="h-8 w-14 rounded border border-gray-200 object-cover dark:border-slate-600" />
          </button>
          <span v-if="todo.color" class="h-3 w-3 shrink-0 rounded-full" :style="{ backgroundColor: todo.color }"></span>
          <span :class="['block truncate text-sm font-black', todo.status === 'completed' ? 'text-slate-400 line-through dark:text-slate-500' : todo.status === 'in_progress' ? 'text-blue-700 dark:text-blue-300' : 'text-slate-900 dark:text-white']">{{ todo.title }}</span>
          <span v-if="todo.pinned" class="rounded-full bg-indigo-50 px-1.5 py-0.5 text-[10px] font-black text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300">Pinned</span>
          <TodoPriorityBadge :priority="(todo.priority as any)" />
          <span v-if="todo.status === 'in_progress'" class="rounded-full bg-blue-100 px-1.5 py-0.5 text-[10px] font-black text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">In Progress</span>
          <span v-if="todo.status === 'archived'" class="rounded-full bg-gray-100 px-1.5 py-0.5 text-[10px] font-black text-gray-500 dark:bg-slate-700 dark:text-slate-400">Archived</span>
        </div>
        <p v-if="todo.description" class="mt-0.5 line-clamp-2 text-xs leading-5 text-slate-500 transition-colors dark:text-slate-400">{{ todo.description }}</p>
        <div class="mt-1 flex flex-wrap items-center gap-1">
          <TodoDueDateBadge v-if="todo.start_date" :due-date="todo.start_date" :status="todo.status" />
          <span v-if="todo.end_date && todo.end_date !== todo.start_date" class="inline-flex items-center gap-0.5 rounded-full bg-gray-100 px-1.5 py-0.5 text-[10px] font-black text-slate-500 dark:bg-slate-700 dark:text-slate-400">
            → {{ new Date(todo.end_date).toLocaleDateString() }}
          </span>
          <span v-if="todo.rrule" class="inline-flex items-center gap-0.5 rounded-full bg-blue-50 px-1.5 py-0.5 text-[10px] font-black text-blue-600 dark:bg-blue-900/20 dark:text-blue-400" title="Recurring">
            <svg class="h-2.5 w-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><polyline points="17 1 21 5 17 9" /><path d="M3 11V9a4 4 0 014-4h14" /><polyline points="7 23 3 19 7 15" /><path d="M21 13v2a4 4 0 01-4 4H3" /></svg>
            Recurring
          </span>
          <span v-if="todo.domain" class="inline-flex items-center rounded-full bg-violet-50 px-1.5 py-0.5 text-[10px] font-black text-violet-600 dark:bg-violet-900/20 dark:text-violet-400">{{ todo.domain }}</span>
          <TodoTagBadges :tags="(todo.tags || [])" />
          <button
            v-if="subtaskCount > 0"
            type="button"
            @click="toggleSubtaskVisibility"
            class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[10px] font-black text-indigo-500 transition hover:bg-indigo-50 hover:text-indigo-700 dark:hover:bg-indigo-900/20"
          >
            <svg class="h-3 w-3 shrink-0 transition-transform" :class="showSubtasks ? 'rotate-90' : ''" fill="currentColor" viewBox="0 0 24 24"><path d="M9 6l6 6-6 6" /></svg>
            {{ subtaskCount }} subtask{{ subtaskCount !== 1 ? 's' : '' }}
          </button>
          <button
            v-else
            type="button"
            @click="showSubtasks = true"
            class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[10px] font-black text-indigo-500 transition hover:bg-indigo-50 hover:text-indigo-700 dark:hover:bg-indigo-900/20"
          >
            <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path d="M12 5v14m-7-7h14" /></svg>
            Add subtask
          </button>
          <TodoSubtaskProgress v-if="subtaskCount > 0" :done="subtaskDoneCount" :total="subtaskCount" />
          <span class="mt-1 inline-block rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400">{{ formatDate(todo.updated_at) }}</span>
        </div>
        <div v-if="showSubtasks" class="mt-2">
          <TodoSubtaskList :todo="todo" @toggle-subtask="$emit('toggle-subtask', $event)" />
        </div>
      </div>
    </div>
    <div class="flex shrink-0 flex-col gap-0.5">
      <button
        type="button"
        class="drag-handle cursor-grab active:cursor-grabbing rounded px-1 py-0.5 text-slate-400 transition hover:text-slate-600"
        title="Drag to reorder"
      >
        <svg class="h-3 w-3" fill="currentColor" viewBox="0 0 24 24">
          <circle cx="9" cy="6" r="1.5" />
          <circle cx="15" cy="6" r="1.5" />
          <circle cx="9" cy="12" r="1.5" />
          <circle cx="15" cy="12" r="1.5" />
          <circle cx="9" cy="18" r="1.5" />
          <circle cx="15" cy="18" r="1.5" />
        </svg>
      </button>
      <button type="button" @click="$emit('toggle-pin', todo)" class="rounded px-2 py-1 text-[10px] font-black transition" :class="todo.pinned ? 'bg-indigo-50 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300' : 'text-slate-500 hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10'">{{ todo.pinned ? 'Unpin' : 'Pin' }}</button>
      <button v-if="todo.status === 'archived'" type="button" @click="$emit('restore', todo)" class="rounded px-2 py-1 text-[10px] font-black text-emerald-600 transition hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/20">Restore</button>
      <button v-else-if="todo.status === 'completed'" type="button" @click="$emit('archive', todo)" class="rounded px-2 py-1 text-[10px] font-black text-amber-600 transition hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/20">Archive</button>
      <button v-if="todo.status !== 'archived'" type="button" @click="$emit('startEdit', todo)" class="rounded px-2 py-1 text-[10px] font-black text-slate-500 transition hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400">Edit</button>
      <button type="button" @click="confirmDelete" class="rounded px-2 py-1 text-[10px] font-black text-red-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">Delete</button>
    </div>
    </div>
  </li>
</template>

<script setup lang="ts">
import type { Todo } from '../../types'
import { computed, ref } from 'vue'
import { formatDate } from '../../lib/utils'
import { getScreenshotUrl } from '../../lib/api'
import TodoPriorityBadge from './TodoPriorityBadge.vue'
import TodoDueDateBadge from './TodoDueDateBadge.vue'
import TodoTagBadges from './TodoTagBadges.vue'
import TodoSubtaskProgress from './TodoSubtaskProgress.vue'
import TodoSubtaskList from './TodoSubtaskList.vue'

const props = defineProps<{
  todo: Todo
}>()

const emit = defineEmits<{
  toggle: [todo: Todo]
  'toggle-pin': [todo: Todo]
  archive: [todo: Todo]
  restore: [todo: Todo]
  startEdit: [todo: Todo]
  delete: [id: number]
  viewScreenshot: [todo: Todo]
  'toggle-subtask': [todo: Todo]
}>()

const screenshotUrl = computed(() => props.todo.screenshot_path ? getScreenshotUrl(props.todo.id) : '')

const subtaskCount = computed(() => (props.todo.subtasks || []).length)
const subtaskDoneCount = computed(() => (props.todo.subtasks || []).filter(s => s.status === 'completed').length)

const showSubtasks = ref(false)

function toggleSubtaskVisibility() {
  showSubtasks.value = !showSubtasks.value
}

function confirmDelete() {
  if (window.confirm(`Delete "${props.todo.title}"?`)) {
    emit('delete', props.todo.id)
  }
}
</script>
