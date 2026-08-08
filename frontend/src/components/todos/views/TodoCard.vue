<template>
  <li
    :class="[
      'group rounded-xl border p-3 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md',
      todo.status === 'in_progress'
        ? 'border-blue-300 bg-blue-50/40 hover:border-blue-400 dark:border-blue-500/40 dark:bg-blue-900/10'
        : 'border-gray-200/80 bg-white hover:border-indigo-200 dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-indigo-500/30',
    ]"
  >
    <div class="flex items-start gap-2.5">
      <!-- Drag handle: the grip is the only drag origin for the mobile list -->
      <span
        class="drag-handle mt-0.5 shrink-0 cursor-grab rounded-md text-slate-300 transition active:cursor-grabbing dark:text-slate-600"
        title="Drag to reorder"
        role="button"
        aria-label="Drag to reorder"
      >
        <GripVertical class="h-4 w-4" :stroke-width="2" aria-hidden="true" />
      </span>

      <TodoStatusToggle :status="todo.status" @toggle="$emit('toggle', todo)" />

      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
          <button
            v-if="todo.screenshot_path"
            type="button"
            class="shrink-0 cursor-zoom-in transition hover:opacity-80"
            title="View screenshot"
            aria-label="View screenshot"
            @click="$emit('view-screenshot', todo)"
          >
            <img
              :src="screenshotUrl"
              alt=""
              class="h-8 w-14 rounded border border-gray-200 object-cover dark:border-slate-600"
            />
          </button>
          <span
            v-if="todo.color"
            class="h-3 w-3 shrink-0 rounded-full"
            :style="{ backgroundColor: todo.color }"
            aria-hidden="true"
          ></span>
          <span
            :class="[
              'min-w-0 text-sm leading-tight font-black break-words',
              todo.status === 'completed'
                ? 'text-slate-400 line-through dark:text-slate-500'
                : todo.status === 'in_progress'
                  ? 'text-blue-700 dark:text-blue-300'
                  : 'text-slate-900 dark:text-white',
            ]"
          >
            {{ todo.title }}
          </span>
          <Pin
            v-if="todo.pinned"
            class="h-3.5 w-3.5 shrink-0 text-indigo-500 dark:text-indigo-400"
            :stroke-width="2.5"
            aria-label="Pinned todo"
          />
          <TodoPriorityBadge :priority="todo.priority" />
        </div>

        <p
          v-if="todo.description"
          class="mt-1 line-clamp-2 text-xs leading-5 text-slate-500 transition-colors dark:text-slate-400"
          v-html="linkifyDescription(todo.description)"
        ></p>

        <div class="mt-1.5">
          <TodoMetaChips :todo="todo" />
        </div>

        <div class="mt-1 flex flex-wrap items-center gap-1">
          <button
            type="button"
            class="inline-flex min-h-7 items-center gap-1 rounded-md px-1.5 py-0.5 text-[10px] font-black text-indigo-500 transition hover:bg-indigo-50 hover:text-indigo-700 dark:hover:bg-indigo-900/20"
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
          <span
            class="mt-1 inline-block rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400"
          >
            {{ formatDate(todo.updated_at) }}
          </span>
        </div>

        <div v-if="showSubtasks" class="mt-2">
          <TodoSubtaskList :todo="todo" @toggle-subtask="$emit('toggle-subtask', $event)" />
        </div>
      </div>
    </div>

    <!-- Actions footer: icon buttons, large touch targets -->
    <div class="mt-2 flex justify-end border-t border-gray-100 pt-2 dark:border-slate-700/50">
      <TodoCardActions
        :todo="todo"
        @toggle-pin="$emit('toggle-pin', $event)"
        @archive="$emit('archive', $event)"
        @restore="$emit('restore', $event)"
        @start-edit="$emit('start-edit', $event)"
        @delete="$emit('delete', $event)"
      />
    </div>
  </li>
</template>

<script setup lang="ts">
import { ChevronRight, Plus, Pin } from '@lucide/vue';
import type { Todo } from '../../../types';
import { formatDate } from '../../../lib/utils';
import { linkifyDescription } from '../../../lib/descriptionLinks';
import { useTodoDisplay } from '../composables/useTodoDisplay';
import TodoStatusToggle from '../TodoStatusToggle.vue';
import TodoPriorityBadge from '../TodoPriorityBadge.vue';
import TodoMetaChips from '../TodoMetaChips.vue';
import TodoCardActions from '../TodoCardActions.vue';
import TodoSubtaskProgress from '../TodoSubtaskProgress.vue';
import TodoSubtaskList from './TodoSubtaskList.vue';

const props = defineProps<{
  todo: Todo;
}>();

const emit = defineEmits<{
  toggle: [todo: Todo];
  'toggle-pin': [todo: Todo];
  archive: [todo: Todo];
  restore: [todo: Todo];
  'start-edit': [todo: Todo];
  delete: [id: number];
  'view-screenshot': [todo: Todo];
  'toggle-subtask': [todo: Todo];
}>();

const { screenshotUrl, subtaskCount, subtaskDoneCount, showSubtasks, toggleSubtaskVisibility } =
  useTodoDisplay(() => props.todo);
</script>
