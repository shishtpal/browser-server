<template>
  <article
    :class="[
      'flex flex-col rounded-xl border p-4 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md',
      todo.status === 'in_progress'
        ? 'border-blue-300 bg-blue-50/40 dark:border-blue-500/40 dark:bg-blue-900/10 hover:border-blue-400'
        : 'border-gray-200/80 bg-white hover:border-indigo-200 dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-indigo-500/30',
    ]"
  >
    <!-- Header: status toggle + title metadata -->
    <div class="flex items-start gap-3">
      <button
        type="button"
        :disabled="todo.status === 'archived'"
        :aria-label="statusAriaLabel"
        :title="statusAriaLabel"
        class="mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded-full border-2 transition disabled:cursor-default"
        :class="statusToggleClass"
        @click="$emit('toggle', todo)"
      >
        <svg v-if="todo.status === 'completed'" class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
        </svg>
        <svg v-else-if="todo.status === 'in_progress'" class="h-3 w-3 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24" style="animation-duration: 2s" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M12 3a9 9 0 019 9" />
        </svg>
      </button>
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <span v-if="todo.color" class="h-3 w-3 shrink-0 rounded-full" :style="{ backgroundColor: todo.color }"></span>
          <h3
            :class="[
              'text-sm font-black leading-tight',
              todo.status === 'completed' ? 'text-slate-400 line-through dark:text-slate-500' : todo.status === 'in_progress' ? 'text-blue-700 dark:text-blue-300' : 'text-slate-900 dark:text-white',
            ]"
          >
            {{ todo.title }}
          </h3>
          <span v-if="todo.pinned" class="inline-flex items-center gap-1 rounded-full bg-indigo-50 px-1.5 py-0.5 text-[10px] font-black text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300" title="Pinned todo">
            <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m14 4 6 6-3 1-4 4-1 5-3-3-3-3 5-1 4-4 1-3Z" />
            </svg>
            Pinned
          </span>
          <TodoPriorityBadge :priority="todo.priority" />
          <span
            v-if="todo.status === 'in_progress'"
            class="rounded-full bg-blue-100 px-1.5 py-0.5 text-[10px] font-black text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
          >
            In Progress
          </span>
          <span
            v-else-if="todo.status === 'archived'"
            class="rounded-full bg-gray-100 px-1.5 py-0.5 text-[10px] font-black text-gray-500 dark:bg-slate-700 dark:text-slate-400"
          >
            Archived
          </span>
        </div>
        <div class="mt-1 flex items-center gap-1.5 text-[10px] font-black text-slate-500 dark:text-slate-400">
          <span class="rounded-md bg-gray-100 px-1.5 py-0.5 transition-colors dark:bg-slate-700">Status: {{ statusLabel }}</span>
        </div>
      </div>
    </div>

    <!-- Screenshot -->
    <button
      v-if="todo.screenshot_path"
      type="button"
      class="mt-3 w-full cursor-zoom-in overflow-hidden rounded-lg border border-gray-200 transition hover:opacity-90 dark:border-slate-600"
      title="View screenshot"
      aria-label="View screenshot"
      @click="$emit('view-screenshot', todo)"
    >
      <img :src="screenshotUrl" class="h-28 w-full object-cover" alt="" />
    </button>

    <!-- Description -->
    <div v-if="todo.description" class="mt-3">
      <div
        class="break-words text-sm leading-relaxed text-slate-600 dark:text-slate-300"
        v-html="linkifyDescription(todo.description)"
      ></div>
    </div>

    <!-- Scheduling metadata -->
    <div class="mt-3 flex flex-wrap items-center gap-1.5">
      <TodoDueDateBadge v-if="todo.start_date" :due-date="todo.start_date" :status="todo.status" />
      <span
        v-if="todo.end_date && todo.end_date !== todo.start_date"
        class="inline-flex items-center gap-0.5 rounded-full bg-gray-100 px-1.5 py-0.5 text-[10px] font-black text-slate-500 dark:bg-slate-700 dark:text-slate-400"
        title="End date"
      >
        → {{ endDateLabel }}
      </span>
      <span
        v-if="todo.rrule"
        class="inline-flex items-center gap-0.5 rounded-full bg-blue-50 px-1.5 py-0.5 text-[10px] font-black text-blue-600 dark:bg-blue-900/20 dark:text-blue-400"
        title="Recurring"
      >
        <svg class="h-2.5 w-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" aria-hidden="true">
          <polyline points="17 1 21 5 17 9" />
          <path d="M3 11V9a4 4 0 014-4h14" />
          <polyline points="7 23 3 19 7 15" />
          <path d="M21 13v2a4 4 0 01-4 4H3" />
        </svg>
        Recurring
      </span>
    </div>

    <!-- Domain & tags -->
    <div class="mt-3 flex flex-wrap items-center gap-1.5">
      <span
        v-if="todo.domain"
        class="inline-flex items-center rounded-full bg-violet-50 px-1.5 py-0.5 text-[10px] font-black text-violet-600 dark:bg-violet-900/20 dark:text-violet-400"
      >
        {{ todo.domain }}
      </span>
      <TodoTagBadges :tags="todo.tags || []" />
    </div>

    <!-- Timestamps -->
    <div class="mt-3 flex flex-wrap items-center gap-2 text-[10px] font-bold text-slate-500 transition-colors dark:text-slate-400">
      <span class="rounded-md bg-gray-100 px-1.5 py-0.5 transition-colors dark:bg-slate-700" title="Created">Created {{ formatDate(todo.created_at) }}</span>
      <span class="rounded-md bg-gray-100 px-1.5 py-0.5 transition-colors dark:bg-slate-700" title="Last updated">Updated {{ formatDate(todo.updated_at) }}</span>
    </div>

    <!-- Subtasks -->
    <div class="mt-3">
      <div class="flex items-center gap-2">
        <button
          v-if="subtaskCount > 0"
          type="button"
          class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[10px] font-black text-indigo-500 transition hover:bg-indigo-50 hover:text-indigo-700 dark:hover:bg-indigo-900/20"
          aria-label="Toggle subtasks"
          @click="toggleSubtaskVisibility"
        >
          <svg class="h-3 w-3 shrink-0 transition-transform" :class="showSubtasks ? 'rotate-90' : ''" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M9 6l6 6-6 6" />
          </svg>
          {{ subtaskCount }} subtask{{ subtaskCount !== 1 ? 's' : '' }}
        </button>
        <button
          v-else
          type="button"
          class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[10px] font-black text-indigo-500 transition hover:bg-indigo-50 hover:text-indigo-700 dark:hover:bg-indigo-900/20"
          aria-label="Add subtask"
          @click="showSubtasks = true"
        >
          <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" aria-hidden="true">
            <path d="M12 5v14m-7-7h14" />
          </svg>
          Add subtask
        </button>
        <TodoSubtaskProgress v-if="subtaskCount > 0" :done="subtaskDoneCount" :total="subtaskCount" />
      </div>
      <div v-if="showSubtasks" class="mt-2">
        <TodoSubtaskList :todo="todo" :default-expanded="false" @toggle-subtask="$emit('toggle-subtask', $event)" />
      </div>
    </div>

    <!-- Actions -->
    <div class="mt-auto flex flex-wrap items-center gap-1.5 border-t border-gray-100 pt-3 transition-colors dark:border-slate-700/50">
      <button
        type="button"
        class="rounded-lg px-2.5 py-1.5 text-[10px] font-black transition"
        :class="todo.pinned ? 'bg-indigo-50 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300' : 'text-slate-500 hover:bg-indigo-50 hover:text-indigo-600 dark:text-slate-400 dark:hover:bg-indigo-500/10'"
        @click="$emit('toggle-pin', todo)"
      >
        {{ todo.pinned ? 'Unpin' : 'Pin' }}
      </button>
      <button
        v-if="todo.status === 'archived'"
        type="button"
        class="rounded-lg px-2.5 py-1.5 text-[10px] font-black text-emerald-600 transition hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/20"
        @click="$emit('restore', todo)"
      >
        Restore
      </button>
      <button
        v-else-if="todo.status === 'completed'"
        type="button"
        class="rounded-lg px-2.5 py-1.5 text-[10px] font-black text-amber-600 transition hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/20"
        @click="$emit('archive', todo)"
      >
        Archive
      </button>
      <button
        v-if="todo.status !== 'archived'"
        type="button"
        class="rounded-lg px-2.5 py-1.5 text-[10px] font-black text-slate-500 transition hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400"
        @click="$emit('start-edit', todo)"
      >
        Edit
      </button>
      <button
        type="button"
        class="rounded-lg px-2.5 py-1.5 text-[10px] font-black text-red-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
        @click="confirmDelete"
      >
        Delete
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import type { Todo } from '../../types'
import { formatDate } from '../../lib/utils'
import { linkifyDescription } from '../../lib/descriptionLinks'
import { useTodoDisplay } from '../../composables/useTodoDisplay'
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
  'view-screenshot': [todo: Todo]
  'start-edit': [todo: Todo]
  delete: [id: number]
  'toggle-subtask': [todo: Todo]
}>()

const { screenshotUrl, subtaskCount, subtaskDoneCount, showSubtasks, toggleSubtaskVisibility, confirmDelete, statusLabel, statusAriaLabel, statusToggleClass, endDateLabel } = useTodoDisplay(
  () => props.todo,
  (id) => emit('delete', id),
)
</script>
