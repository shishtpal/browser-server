<template>
  <div>
    <!-- Desktop table (md+) -->
    <div
      class="hidden overflow-hidden rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors md:block dark:border-slate-700/80 dark:bg-slate-800/90"
    >
      <table class="min-w-full divide-y divide-gray-200 transition-colors dark:divide-slate-700">
        <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
          <tr>
            <th
              v-for="col in columns"
              :key="col.label"
              class="px-3 py-3 text-left text-[10px] font-black tracking-wide text-slate-500 uppercase transition-colors dark:text-slate-400"
              :class="[col.align === 'right' ? 'text-right' : '', col.width ?? '']"
            >
              {{ col.label }}
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
          <template v-for="todo in list" :key="todo.id">
            <TodoTableRow
              :todo="todo"
              :expanded="expandedIds.has(todo.id)"
              draggable="true"
              @toggle="$emit('toggle', $event)"
              @toggle-pin="$emit('toggle-pin', $event)"
              @archive="$emit('archive', $event)"
              @restore="$emit('restore', $event)"
              @start-edit="$emit('start-edit', $event)"
              @delete="$emit('delete', $event)"
              @view-screenshot="$emit('view-screenshot', $event)"
              @toggle-expand="$emit('toggle-expand', $event)"
              @mousedown="$emit('row-mousedown', $event)"
              @dragstart="$emit('row-dragstart', $event, todo.id)"
              @dragover.prevent="$emit('row-dragover', $event, todo.id)"
              @drop="$emit('row-drop', $event, todo.id)"
              @dragend="$emit('row-dragend')"
            />
            <tr v-if="expandedIds.has(todo.id)" class="bg-indigo-50/40 dark:bg-slate-800/40">
              <td :colspan="columns.length" class="px-4 py-3">
                <TodoSubtaskList
                  :todo="todo"
                  :default-expanded="true"
                  @toggle-subtask="$emit('toggle-subtask', $event)"
                />
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <!-- Mobile card list (md-): drag-to-reorder via the grip handle -->
    <draggable
      v-model="list"
      item-key="id"
      handle=".drag-handle"
      tag="ul"
      class="space-y-2 md:hidden"
      @end="$emit('mobile-dragend', $event as { oldIndex: number; newIndex: number })"
    >
      <template #item="{ element: todo }">
        <TodoCard
          :todo="todo"
          @toggle="$emit('toggle', $event)"
          @toggle-pin="$emit('toggle-pin', $event)"
          @archive="$emit('archive', $event)"
          @restore="$emit('restore', $event)"
          @start-edit="$emit('start-edit', $event)"
          @delete="$emit('delete', $event)"
          @view-screenshot="$emit('view-screenshot', $event)"
          @toggle-subtask="$emit('toggle-subtask', $event)"
        />
      </template>
    </draggable>
  </div>
</template>

<script setup lang="ts">
import draggable from 'vuedraggable';
import type { Todo } from '../../../types';
import TodoTableRow from './TodoTableRow.vue';
import TodoCard from './TodoCard.vue';
import TodoSubtaskList from './TodoSubtaskList.vue';

defineProps<{
  expandedIds: Set<number>;
}>();

/** The reorderable list (vuedraggable mutates it); the page persists changes. */
const list = defineModel<Todo[]>({ required: true });

defineEmits<{
  toggle: [todo: Todo];
  'toggle-pin': [todo: Todo];
  archive: [todo: Todo];
  restore: [todo: Todo];
  'start-edit': [todo: Todo];
  delete: [id: number];
  'view-screenshot': [todo: Todo];
  'toggle-expand': [id: number];
  'toggle-subtask': [todo: Todo];
  'row-mousedown': [event: MouseEvent];
  'row-dragstart': [event: DragEvent, id: number];
  'row-dragover': [event: DragEvent, id: number];
  'row-drop': [event: DragEvent, id: number];
  'row-dragend': [];
  'mobile-dragend': [event: { oldIndex: number; newIndex: number }];
}>();

const columns: { label: string; align?: 'left' | 'right'; width?: string }[] = [
  { label: '', width: 'w-14' },
  { label: 'Title' },
  { label: 'Description' },
  { label: 'Due date' },
  { label: 'Tags' },
  { label: 'Updated' },
  { label: 'Subtasks' },
  { label: 'Actions', align: 'right', width: 'w-44' },
];
</script>
