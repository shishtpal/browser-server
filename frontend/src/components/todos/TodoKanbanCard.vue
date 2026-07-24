<template>
  <div class="relative group rounded-lg border border-gray-200 bg-white p-2.5 shadow-sm transition hover:-translate-y-0.5 hover:border-indigo-200 hover:shadow-md dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-indigo-500/30">
    <button
      type="button"
      class="drag-handle absolute top-1 right-1 rounded px-1 py-0.5 text-slate-400 opacity-0 transition group-hover:opacity-100"
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
    <div class="flex items-start gap-2 pr-6">
      <button
        type="button"
        @click="$emit('toggle', todo)"
        class="mt-0.5 grid h-4 w-4 shrink-0 place-items-center rounded-full border-2 transition"
        :class="todo.completed ? 'border-emerald-500 bg-emerald-500 text-white' : 'border-gray-300 text-transparent hover:border-indigo-400 dark:border-slate-600 dark:hover:border-indigo-400'"
      >
        <svg class="h-2.5 w-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
        </svg>
      </button>
      <div class="min-w-0 flex-1">
        <span :class="['block truncate text-xs font-black', todo.completed ? 'text-slate-400 line-through dark:text-slate-500' : 'text-slate-900 dark:text-white']">
          {{ todo.title }}
        </span>
        <div class="mt-1 flex flex-wrap items-center gap-1">
          <TodoPriorityBadge :priority="todo.priority as any" />
          <TodoDueDateBadge v-if="todo.due_date" :due-date="todo.due_date" :completed="todo.completed" />
          <TodoTagBadges :tags="(todo.tags || [])" />
          <button
            v-if="(todo.subtasks?.length || 0) > 0"
            type="button"
            @click="$emit('toggle-expand', todo.id)"
            class="text-[9px] font-black text-indigo-500 transition hover:text-indigo-700"
          >
            {{ expanded ? '−' : '+' }} {{ todo.subtasks?.length }}
          </button>
          <TodoSubtaskProgress v-else-if="todo.parent_id" :done="0" :total="0" />
        </div>
        <div v-if="expanded" class="mt-2 border-t border-gray-100 pt-2 dark:border-slate-700">
          <TodoSubtaskList :todo="todo" @toggle-subtask="$emit('toggle', $event)" />
        </div>
      </div>
      <div class="flex shrink-0 gap-0.5">
        <button type="button" @click="confirmDelete" class="rounded px-1 py-0.5 text-[10px] font-black text-red-400 transition hover:text-red-600 hidden group-hover:flex">✕</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, type PropType } from 'vue'
import type { Todo } from '../../types'
import TodoPriorityBadge from './TodoPriorityBadge.vue'
import TodoDueDateBadge from './TodoDueDateBadge.vue'
import TodoTagBadges from './TodoTagBadges.vue'
import TodoSubtaskProgress from './TodoSubtaskProgress.vue'
import TodoSubtaskList from './TodoSubtaskList.vue'

const props = defineProps({
  todo: { type: Object as PropType<Todo>, required: true },
  expanded: { type: Boolean, default: false },
})

const subtaskCount = computed(() => (props.todo.subtasks || []).length)
const subtaskDoneCount = computed(() => (props.todo.subtasks || []).filter(s => s.completed).length)

const emit = defineEmits<{
  toggle: [todo: Todo]
  'start-edit': [todo: Todo]
  'save-edit': [todo: Todo, title: string, description: string, priority: string, dueDate: string | null, tags: string[]]
  'cancel-edit': []
  delete: [id: number]
  'toggle-expand': [id: number]
}>()

function confirmDelete() {
  if (window.confirm(`Delete "${props.todo.title}"?`)) {
    emit('delete', props.todo.id)
  }
}
</script>
