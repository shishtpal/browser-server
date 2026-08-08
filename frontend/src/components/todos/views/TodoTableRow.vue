<template>
  <tr class="group transition hover:bg-indigo-50/60 dark:hover:bg-indigo-900/20">
    <td class="w-14 px-3 py-3">
      <div class="flex items-center gap-1.5">
        <span
          class="drag-handle cursor-grab text-slate-300 transition hover:text-slate-500 active:cursor-grabbing dark:text-slate-600 dark:hover:text-slate-400"
          title="Drag to reorder"
          role="button"
          aria-label="Drag to reorder"
        >
          <GripVertical class="h-4 w-4" :stroke-width="2" aria-hidden="true" />
        </span>
        <TodoStatusToggle :status="todo.status" @toggle="$emit('toggle', todo)" />
      </div>
    </td>

    <td class="max-w-xs px-3 py-3">
      <div class="flex items-center gap-2">
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
            class="h-6 w-10 rounded border border-gray-200 object-cover dark:border-slate-600"
          />
        </button>
        <span
          :class="[
            'block truncate text-sm font-black',
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
    </td>

    <td class="max-w-xs px-3 py-3">
      <span
        class="block truncate text-sm text-slate-500 transition-colors dark:text-slate-400"
        v-html="todo.description ? linkifyDescription(todo.description) : '—'"
      ></span>
    </td>

    <td class="px-3 py-3">
      <TodoDueDateBadge v-if="todo.start_date" :due-date="todo.start_date" :status="todo.status" />
    </td>

    <td class="px-3 py-3">
      <TodoTagBadges :tags="todo.tags || []" />
    </td>

    <td class="px-3 py-3">
      <span
        class="rounded-md bg-gray-100 px-2 py-1 text-[10px] font-bold whitespace-nowrap text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400"
        >{{ formatDate(todo.updated_at) }}</span
      >
    </td>

    <td class="px-3 py-3">
      <button
        type="button"
        class="flex items-center gap-1.5 text-[10px] font-black text-slate-500 transition hover:text-indigo-600 dark:text-slate-400 dark:hover:text-indigo-400"
        :aria-expanded="expanded"
        @click="$emit('toggle-expand', todo.id)"
      >
        <ChevronRight
          class="h-3 w-3 shrink-0 transition-transform"
          :class="{ 'rotate-90': expanded }"
          :stroke-width="2.5"
          aria-hidden="true"
        />
        <span>{{ subtaskCount }} subtask{{ subtaskCount !== 1 ? 's' : '' }}</span>
      </button>
      <TodoSubtaskProgress v-if="subtaskCount > 0" :done="subtaskDoneCount" :total="subtaskCount" />
    </td>

    <td class="px-3 py-3 text-right">
      <div class="flex items-center justify-end gap-0.5">
        <button
          type="button"
          :title="todo.pinned ? 'Unpin' : 'Pin to top'"
          :aria-label="todo.pinned ? 'Unpin todo' : 'Pin todo'"
          class="grid h-8 w-8 place-items-center rounded-lg transition"
          :class="
            todo.pinned
              ? 'bg-indigo-50 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300'
              : 'text-slate-400 hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400'
          "
          @click="$emit('toggle-pin', todo)"
        >
          <PinOff v-if="todo.pinned" class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
          <Pin v-else class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
        <button
          v-if="todo.status === 'archived'"
          type="button"
          title="Restore"
          aria-label="Restore todo"
          class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-emerald-50 hover:text-emerald-600 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-400"
          @click="$emit('restore', todo)"
        >
          <ArchiveRestore class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
        <button
          v-else-if="todo.status === 'completed'"
          type="button"
          title="Archive"
          aria-label="Archive todo"
          class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-amber-50 hover:text-amber-600 dark:hover:bg-amber-900/20 dark:hover:text-amber-400"
          @click="$emit('archive', todo)"
        >
          <Archive class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
        <button
          v-if="todo.status !== 'archived'"
          type="button"
          title="Edit"
          aria-label="Edit todo"
          class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400"
          @click="$emit('start-edit', todo)"
        >
          <Pencil class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
        <button
          type="button"
          title="Delete"
          aria-label="Delete todo"
          class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20 dark:hover:text-red-400"
          @click="$emit('delete', todo.id)"
        >
          <Trash2 class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
      </div>
    </td>
  </tr>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  Archive,
  ArchiveRestore,
  ChevronRight,
  GripVertical,
  Pencil,
  Pin,
  PinOff,
  Trash2,
} from '@lucide/vue';
import type { Todo } from '../../../types';
import { formatDate } from '../../../lib/utils';
import { getScreenshotUrl } from '../../../lib/api';
import { linkifyDescription } from '../../../lib/descriptionLinks';
import TodoStatusToggle from '../TodoStatusToggle.vue';
import TodoPriorityBadge from '../TodoPriorityBadge.vue';
import TodoDueDateBadge from '../TodoDueDateBadge.vue';
import TodoTagBadges from '../TodoTagBadges.vue';
import TodoSubtaskProgress from '../TodoSubtaskProgress.vue';

const props = defineProps<{
  todo: Todo;
  expanded?: boolean;
}>();

defineEmits<{
  toggle: [todo: Todo];
  'toggle-pin': [todo: Todo];
  archive: [todo: Todo];
  restore: [todo: Todo];
  'start-edit': [todo: Todo];
  delete: [id: number];
  'view-screenshot': [todo: Todo];
  'toggle-expand': [id: number];
}>();

const screenshotUrl = computed(() =>
  props.todo.screenshot_path ? getScreenshotUrl(props.todo.id) : '',
);

const subtaskCount = computed(() => (props.todo.subtasks || []).length);
const subtaskDoneCount = computed(
  () => (props.todo.subtasks || []).filter((s) => s.status === 'completed').length,
);
</script>
