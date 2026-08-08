<template>
  <article
    :class="[
      'flex flex-col rounded-xl border p-4 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md',
      todo.status === 'in_progress'
        ? 'border-blue-300 bg-blue-50/40 hover:border-blue-400 dark:border-blue-500/40 dark:bg-blue-900/10'
        : 'border-gray-200/80 bg-white hover:border-indigo-200 dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-indigo-500/30',
    ]"
  >
    <!-- Header: status toggle + title metadata -->
    <div class="flex items-start gap-3">
      <TodoStatusToggle :status="todo.status" class="mt-0.5" @toggle="$emit('toggle', todo)" />
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <span
            v-if="todo.color"
            class="h-3 w-3 shrink-0 rounded-full"
            :style="{ backgroundColor: todo.color }"
            aria-hidden="true"
          ></span>
          <h3
            :class="[
              'text-sm leading-tight font-black break-words',
              todo.status === 'completed'
                ? 'text-slate-400 line-through dark:text-slate-500'
                : todo.status === 'in_progress'
                  ? 'text-blue-700 dark:text-blue-300'
                  : 'text-slate-900 dark:text-white',
            ]"
          >
            {{ todo.title }}
          </h3>
          <Pin
            v-if="todo.pinned"
            class="h-3.5 w-3.5 shrink-0 text-indigo-500 dark:text-indigo-400"
            :stroke-width="2.5"
            aria-label="Pinned todo"
          />
          <TodoPriorityBadge :priority="todo.priority" />
          <span class="rounded-full px-1.5 py-0.5 text-[10px] font-black" :class="statusBadgeClass">
            {{ statusLabel }}
          </span>
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
        class="text-sm leading-relaxed break-words text-slate-600 dark:text-slate-300"
        v-html="linkifyDescription(todo.description)"
      ></div>
    </div>

    <!-- Meta chips -->
    <div class="mt-3">
      <TodoMetaChips :todo="todo" />
    </div>

    <!-- Timestamps -->
    <div
      class="mt-3 flex flex-wrap items-center gap-2 text-[10px] font-bold text-slate-500 transition-colors dark:text-slate-400"
    >
      <span
        class="rounded-md bg-gray-100 px-1.5 py-0.5 transition-colors dark:bg-slate-700"
        title="Created"
      >
        Created {{ formatDate(todo.created_at) }}
      </span>
      <span
        class="rounded-md bg-gray-100 px-1.5 py-0.5 transition-colors dark:bg-slate-700"
        title="Last updated"
      >
        Updated {{ formatDate(todo.updated_at) }}
      </span>
    </div>

    <!-- Subtasks -->
    <div class="mt-3">
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="inline-flex min-h-7 items-center gap-1 rounded-md px-2 py-1 text-[10px] font-black text-indigo-500 transition hover:bg-indigo-50 hover:text-indigo-700 dark:hover:bg-indigo-900/20"
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
      <div v-if="showSubtasks" class="mt-2">
        <TodoSubtaskList
          :todo="todo"
          :default-expanded="false"
          @toggle-subtask="$emit('toggle-subtask', $event)"
        />
      </div>
    </div>

    <!-- Actions -->
    <div
      class="mt-auto flex justify-end border-t border-gray-100 pt-3 transition-colors dark:border-slate-700/50"
    >
      <TodoCardActions
        :todo="todo"
        @toggle-pin="$emit('toggle-pin', $event)"
        @archive="$emit('archive', $event)"
        @restore="$emit('restore', $event)"
        @start-edit="$emit('start-edit', $event)"
        @delete="$emit('delete', $event)"
      />
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ChevronRight, Pin, Plus } from '@lucide/vue';
import type { Todo } from '../../../types';
import { formatDate } from '../../../lib/utils';
import { linkifyDescription } from '../../../lib/descriptionLinks';
import { useTodoDisplay } from '../composables/useTodoDisplay';
import { STATUS_META } from '../todoFormat';
import TodoStatusToggle from '../TodoStatusToggle.vue';
import TodoPriorityBadge from '../TodoPriorityBadge.vue';
import TodoMetaChips from '../TodoMetaChips.vue';
import TodoCardActions from '../TodoCardActions.vue';
import TodoSubtaskProgress from '../TodoSubtaskProgress.vue';
import TodoSubtaskList from './TodoSubtaskList.vue';

const props = defineProps<{
  todo: Todo;
}>();

defineEmits<{
  toggle: [todo: Todo];
  'toggle-pin': [todo: Todo];
  archive: [todo: Todo];
  restore: [todo: Todo];
  'view-screenshot': [todo: Todo];
  'start-edit': [todo: Todo];
  delete: [id: number];
  'toggle-subtask': [todo: Todo];
}>();

const { screenshotUrl, subtaskCount, subtaskDoneCount, showSubtasks, toggleSubtaskVisibility } =
  useTodoDisplay(() => props.todo);

const statusLabel = computed(() => STATUS_META[props.todo.status]?.label ?? props.todo.status);
const statusBadgeClass = computed(
  () => STATUS_META[props.todo.status]?.badgeClass ?? 'bg-gray-100 text-gray-500',
);
</script>
