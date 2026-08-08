<template>
  <div
    class="mt-2 rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-slate-700 dark:bg-slate-800/60"
  >
    <div class="flex items-center justify-between">
      <button
        type="button"
        class="inline-flex min-h-7 items-center gap-1 text-[10px] font-black text-slate-500 transition hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
        :aria-expanded="expanded"
        @click="expanded = !expanded"
      >
        <ChevronRight
          class="h-3 w-3 shrink-0 transition-transform"
          :class="{ 'rotate-90': expanded }"
          :stroke-width="2.5"
          aria-hidden="true"
        />
        Subtasks ({{ subtasks.length }})
      </button>
      <TodoSubtaskProgress :done="progress.done" :total="progress.total" />
    </div>

    <div v-if="expanded" class="mt-2 space-y-1">
      <draggable
        v-model="subtasks"
        item-key="id"
        handle=".drag-handle"
        tag="div"
        @end="onSubtaskEnd"
      >
        <template #item="{ element }">
          <div
            class="group flex items-center gap-2 rounded-md bg-white p-2 transition dark:bg-slate-800"
          >
            <span
              class="drag-handle cursor-grab text-slate-300 transition active:cursor-grabbing dark:text-slate-600"
              title="Drag to reorder"
              role="button"
              aria-label="Drag to reorder"
            >
              <GripVertical class="h-3 w-3" :stroke-width="2" aria-hidden="true" />
            </span>

            <TodoStatusToggle
              :status="element.status"
              size="sm"
              @toggle="onToggleSubtask(element)"
            />

            <!-- Inline edit mode -->
            <template v-if="editingId === element.id">
              <input
                ref="editInput"
                v-model="editTitle"
                class="min-w-0 flex-1 rounded-md border border-indigo-400 bg-white px-2 py-0.5 text-xs font-semibold text-slate-700 focus:outline-none dark:border-indigo-500 dark:bg-slate-800 dark:text-slate-200"
                @keydown.enter="saveEdit(element)"
                @keydown.escape="cancelEdit"
              />
              <select
                v-model="editPriority"
                class="hidden rounded-md border border-gray-300 bg-white px-1 py-0.5 text-[10px] font-black text-slate-600 focus:border-indigo-400 focus:outline-none sm:block dark:border-slate-600 dark:bg-slate-800 dark:text-slate-300"
                aria-label="Priority"
              >
                <option v-for="p in PRIORITY_ORDER" :key="p" :value="p">
                  {{ PRIORITY_META[p].label }}
                </option>
              </select>
              <input
                v-model="editDueDate"
                type="date"
                class="hidden rounded-md border border-gray-300 bg-white px-1 py-0.5 text-[10px] font-black text-slate-600 focus:border-indigo-400 focus:outline-none sm:block dark:border-slate-600 dark:bg-slate-800 dark:text-slate-300"
                aria-label="Due date"
              />
              <button
                type="button"
                class="grid h-6 w-6 shrink-0 place-items-center rounded-md bg-emerald-500 text-white transition hover:bg-emerald-600 disabled:opacity-40"
                :disabled="!editTitle.trim()"
                aria-label="Save subtask"
                @click="saveEdit(element)"
              >
                <Check class="h-3.5 w-3.5" :stroke-width="3" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="grid h-6 w-6 shrink-0 place-items-center rounded-md bg-gray-200 text-slate-600 transition hover:bg-gray-300 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600"
                aria-label="Cancel editing"
                @click="cancelEdit"
              >
                <X class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
              </button>
            </template>

            <!-- Display mode -->
            <template v-else>
              <span
                class="min-w-0 flex-1 cursor-pointer text-xs font-semibold break-words text-slate-700 dark:text-slate-200"
                :class="{
                  'text-slate-400 line-through dark:text-slate-500': element.status === 'completed',
                }"
                title="Double-click to edit"
                @dblclick="startEdit(element)"
              >
                {{ element.title }}
              </span>
              <TodoPriorityBadge :priority="element.priority" />
              <TodoDueDateBadge
                v-if="element.start_date"
                :due-date="element.start_date"
                :status="element.status"
              />
              <button
                type="button"
                class="grid h-7 w-7 shrink-0 place-items-center rounded-md text-slate-400 transition group-hover:opacity-100 hover:bg-gray-100 hover:text-indigo-500 sm:opacity-0 dark:hover:bg-slate-700 dark:hover:text-indigo-400"
                title="Edit subtask"
                aria-label="Edit subtask"
                @click="startEdit(element)"
              >
                <Pencil class="h-3 w-3" :stroke-width="2.25" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="grid h-7 w-7 shrink-0 place-items-center rounded-md text-slate-400 transition group-hover:opacity-100 hover:bg-red-50 hover:text-red-500 sm:opacity-0 dark:hover:bg-red-950/30 dark:hover:text-red-400"
                title="Delete subtask"
                aria-label="Delete subtask"
                @click="onRemoveSubtask(element.id)"
              >
                <Trash2 class="h-3 w-3" :stroke-width="2.25" aria-hidden="true" />
              </button>
            </template>
          </div>
        </template>
      </draggable>

      <form class="mt-2 flex items-center gap-2" @submit.prevent="onAddSubtask">
        <input
          v-model="newTitle"
          placeholder="Add subtask..."
          class="min-w-0 flex-1 rounded-md border border-gray-300 bg-white px-2 py-1.5 text-xs font-semibold text-slate-700 focus:border-indigo-400 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        />
        <button
          type="submit"
          class="inline-flex shrink-0 items-center gap-1 rounded-md bg-indigo-500 px-2.5 py-1.5 text-[10px] font-black text-white transition hover:bg-indigo-600 disabled:opacity-40"
          :disabled="!newTitle.trim()"
        >
          <Plus class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          Add
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Todo, TodoPriority } from '../../../types';
import { ref, computed, watch, nextTick } from 'vue';
import draggable from 'vuedraggable';
import { Check, ChevronRight, GripVertical, Pencil, Plus, Trash2, X } from '@lucide/vue';
import { reorderTodos } from '../../../lib/api';
import TodoStatusToggle from '../TodoStatusToggle.vue';
import TodoPriorityBadge from '../TodoPriorityBadge.vue';
import TodoDueDateBadge from '../TodoDueDateBadge.vue';
import TodoSubtaskProgress from '../TodoSubtaskProgress.vue';
import { useTodoSubtasks } from '../composables/useTodoSubtasks';
import { PRIORITY_META, PRIORITY_ORDER } from '../todoFormat';

const props = withDefaults(
  defineProps<{
    todo: Todo;
    defaultExpanded?: boolean;
  }>(),
  { defaultExpanded: true },
);

const emit = defineEmits<{
  'toggle-subtask': [todo: Todo];
}>();

const expanded = ref(props.defaultExpanded);

const userId = computed(() => props.todo.user_id);
const { subtasks, progress, addSubtask, toggleSubtask, saveSubtask, removeSubtask } =
  useTodoSubtasks(
    props.todo.subtasks || [],
    computed(() => props.todo.id),
    userId,
  );

// Sync when the parent refetches.
watch(
  () => props.todo.subtasks,
  (val) => {
    subtasks.value = [...(val || [])];
  },
  { deep: true },
);

const newTitle = ref('');

/* ------------------------------- inline edit ------------------------------- */

const editingId = ref<number | null>(null);
const editTitle = ref('');
const editPriority = ref<TodoPriority>('medium');
const editDueDate = ref('');
const editInput = ref<HTMLInputElement[] | null>(null);

function startEdit(subtask: Todo) {
  editingId.value = subtask.id;
  editTitle.value = subtask.title;
  editPriority.value = subtask.priority || 'medium';
  editDueDate.value = subtask.start_date ? subtask.start_date.slice(0, 10) : '';
  nextTick(() => {
    const input = Array.isArray(editInput.value) ? editInput.value[0] : editInput.value;
    input?.focus();
    input?.select();
  });
}

function cancelEdit() {
  editingId.value = null;
}

async function saveEdit(subtask: Todo) {
  const title = editTitle.value.trim();
  if (!title) return;

  const updates: Partial<Todo> = {};
  if (title !== subtask.title) updates.title = title;
  if (editPriority.value !== subtask.priority) updates.priority = editPriority.value;
  const newDate = editDueDate.value || null;
  const oldDate = subtask.start_date ? subtask.start_date.slice(0, 10) : null;
  if (newDate !== oldDate) updates.start_date = newDate;

  const updated = await saveSubtask(subtask, updates);
  if (updated) emit('toggle-subtask', updated);
  editingId.value = null;
}

/* --------------------------------- actions --------------------------------- */

async function onToggleSubtask(subtask: Todo) {
  const updated = await toggleSubtask(subtask);
  if (updated) emit('toggle-subtask', updated);
}

function onRemoveSubtask(id: number) {
  removeSubtask(id);
}

async function onSubtaskEnd(event: { oldIndex: number; newIndex: number }) {
  if (event.oldIndex === event.newIndex) return;
  await reorderTodos(subtasks.value.map((t, idx) => ({ id: t.id, position: idx })));
}

function onAddSubtask() {
  const title = newTitle.value.trim();
  if (!title) return;
  addSubtask(title);
  newTitle.value = '';
}
</script>
