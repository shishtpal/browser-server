<template>
  <div
    :class="[
      'group relative rounded-lg border p-2.5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md',
      todo.status === 'in_progress'
        ? 'border-blue-300 bg-blue-50/40 hover:border-blue-400 dark:border-blue-500/40 dark:bg-blue-900/10'
        : 'border-gray-200 bg-white hover:border-indigo-200 dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-indigo-500/30',
    ]"
  >
    <!-- Drag grip + quick actions (always visible on touch, hover-revealed on desktop) -->
    <div class="absolute top-1 right-1 flex items-center gap-0.5">
      <button
        type="button"
        class="grid h-6 w-6 place-items-center rounded-md text-slate-400 transition hover:bg-indigo-50 hover:text-indigo-600 sm:opacity-0 sm:group-hover:opacity-100 dark:hover:bg-indigo-500/10"
        title="Edit todo"
        aria-label="Edit todo"
        @click="$emit('start-edit', todo)"
      >
        <Pencil class="h-3 w-3" :stroke-width="2.25" aria-hidden="true" />
      </button>
      <button
        type="button"
        class="grid h-6 w-6 place-items-center rounded-md text-slate-400 transition hover:bg-red-50 hover:text-red-500 sm:opacity-0 sm:group-hover:opacity-100 dark:hover:bg-red-900/20"
        title="Delete todo"
        aria-label="Delete todo"
        @click="$emit('delete', todo.id)"
      >
        <Trash2 class="h-3 w-3" :stroke-width="2.25" aria-hidden="true" />
      </button>
      <span
        class="drag-handle grid h-6 w-6 cursor-grab place-items-center rounded-md text-slate-300 transition active:cursor-grabbing dark:text-slate-600"
        title="Drag to reorder"
        role="button"
        aria-label="Drag to reorder"
      >
        <GripVertical class="h-3 w-3" :stroke-width="2" aria-hidden="true" />
      </span>
    </div>

    <div class="flex items-start gap-2 pr-20">
      <TodoStatusToggle
        :status="todo.status"
        size="sm"
        class="mt-0.5"
        @toggle="$emit('toggle', todo)"
      />
      <div class="min-w-0 flex-1">
        <span
          :class="[
            'block truncate text-xs font-black',
            todo.status === 'completed'
              ? 'text-slate-400 line-through dark:text-slate-500'
              : todo.status === 'in_progress'
                ? 'text-blue-700 dark:text-blue-300'
                : 'text-slate-900 dark:text-white',
          ]"
        >
          {{ todo.title }}
        </span>
        <div class="mt-1.5">
          <TodoMetaChips :todo="todo" class="gap-1 text-[9px]" />
        </div>
        <div class="mt-1 flex flex-wrap items-center gap-1">
          <TodoPriorityBadge :priority="todo.priority" />
          <button
            type="button"
            class="inline-flex min-h-6 items-center gap-1 rounded-md px-1 py-0.5 text-[9px] font-black text-indigo-500 transition hover:text-indigo-700"
            @click="toggleSubtaskVisibility"
          >
            <ChevronRight
              v-if="subtaskCount > 0"
              class="h-3 w-3 shrink-0 transition-transform"
              :class="{ 'rotate-90': showSubtasks }"
              :stroke-width="2.5"
              aria-hidden="true"
            />
            <Plus v-else class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
            {{
              subtaskCount > 0
                ? `${subtaskCount} subtask${subtaskCount !== 1 ? 's' : ''}`
                : 'Add subtask'
            }}
          </button>
          <TodoSubtaskProgress
            v-if="subtaskCount > 0"
            :done="subtaskDoneCount"
            :total="subtaskCount"
          />
        </div>
        <div v-if="showSubtasks" class="mt-2 border-t border-gray-100 pt-2 dark:border-slate-700">
          <TodoSubtaskList :todo="todo" @toggle-subtask="$emit('toggle-subtask', $event)" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChevronRight, GripVertical, Pencil, Plus, Trash2 } from '@lucide/vue';
import type { Todo } from '../../../types';
import { useTodoDisplay } from '../composables/useTodoDisplay';
import TodoStatusToggle from '../TodoStatusToggle.vue';
import TodoPriorityBadge from '../TodoPriorityBadge.vue';
import TodoMetaChips from '../TodoMetaChips.vue';
import TodoSubtaskProgress from '../TodoSubtaskProgress.vue';
import TodoSubtaskList from './TodoSubtaskList.vue';

const props = defineProps<{ todo: Todo }>();

defineEmits<{
  toggle: [todo: Todo];
  'toggle-subtask': [todo: Todo];
  'view-screenshot': [todo: Todo];
  'start-edit': [todo: Todo];
  delete: [id: number];
}>();

const { subtaskCount, subtaskDoneCount, showSubtasks, toggleSubtaskVisibility } = useTodoDisplay(
  () => props.todo,
);
</script>
