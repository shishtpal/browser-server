<template>
  <!-- Mobile/tablet: horizontally snapping columns; desktop: 4-column grid -->
  <div
    class="-mx-3 flex snap-x snap-mandatory scrollbar-thin gap-3 overflow-x-auto px-3 pb-2 sm:mx-0 sm:px-0 lg:grid lg:snap-none lg:grid-cols-4 lg:overflow-visible"
  >
    <section
      v-for="p in PRIORITY_ORDER"
      :key="p"
      class="flex w-[82vw] max-w-80 shrink-0 snap-start flex-col rounded-xl border border-gray-200 bg-gray-50/80 sm:w-72 lg:w-auto lg:max-w-none dark:border-slate-700/80 dark:bg-slate-800/60"
      :aria-label="`${PRIORITY_META[p].label} priority column`"
    >
      <header
        class="mx-2 mt-2 flex items-center justify-center gap-1.5 rounded-lg px-2 py-1.5 text-center text-[10px] font-black tracking-wide uppercase"
        :class="PRIORITY_META[p].badgeClass"
      >
        <span
          class="h-1.5 w-1.5 rounded-full"
          :class="PRIORITY_META[p].dotClass"
          aria-hidden="true"
        />
        {{ PRIORITY_META[p].label }}
        <span class="tabular-nums">({{ columnCounts[p] }})</span>
      </header>

      <draggable
        v-model="columnLists[p]"
        :group="{ name: 'todos', pull: true, put: true }"
        item-key="id"
        handle=".drag-handle"
        tag="div"
        class="m-2 min-h-[200px] space-y-2"
        @change="onColumnChange(p, $event)"
      >
        <template #item="{ element: todo }">
          <TodoKanbanCard
            :todo="todo"
            @toggle="$emit('toggle', $event)"
            @toggle-subtask="$emit('toggle-subtask', $event)"
            @view-screenshot="$emit('view-screenshot', $event)"
            @start-edit="$emit('start-edit', $event)"
            @delete="$emit('delete', $event)"
          />
        </template>
      </draggable>

      <p
        v-if="columnCounts[p] === 0"
        class="m-2 flex items-center justify-center gap-1.5 rounded-lg border border-dashed border-gray-300 p-3 text-center text-[10px] font-black text-slate-400 dark:border-slate-600 dark:text-slate-500"
      >
        <Inbox class="h-3.5 w-3.5" :stroke-width="2" aria-hidden="true" />
        No {{ PRIORITY_META[p].label.toLowerCase() }} tasks
      </p>
    </section>
  </div>
</template>

<script setup lang="ts">
import type { ReorderItem, Todo, TodoPriority } from '../../../types';
import { computed, ref, watch } from 'vue';
import draggable from 'vuedraggable';
import { Inbox } from '@lucide/vue';
import TodoKanbanCard from './TodoKanbanCard.vue';
import { PRIORITY_META, PRIORITY_ORDER } from '../todoFormat';

const props = defineProps<{ todos: Todo[] }>();

const emit = defineEmits<{
  toggle: [todo: Todo];
  'toggle-subtask': [todo: Todo];
  'view-screenshot': [todo: Todo];
  'start-edit': [todo: Todo];
  delete: [id: number];
  reorder: [items: ReorderItem[]];
  'priority-change': [payload: { todo: Todo; newPriority: string; items: ReorderItem[] }];
}>();

const columnLists = ref<Record<TodoPriority, Todo[]>>(emptyColumns());

watch(
  () => props.todos,
  (newTodos) => {
    const map = emptyColumns();
    for (const todo of newTodos) map[todo.priority]?.push(todo);
    columnLists.value = map;
  },
  { immediate: true },
);

function emptyColumns(): Record<TodoPriority, Todo[]> {
  return { low: [], medium: [], high: [], urgent: [] };
}

const columnCounts = computed(() => {
  const counts = Object.fromEntries(PRIORITY_ORDER.map((p) => [p, 0])) as Record<
    TodoPriority,
    number
  >;
  props.todos.forEach((t) => {
    if (counts[t.priority] !== undefined) counts[t.priority]++;
  });
  return counts;
});

function onColumnChange(
  priority: TodoPriority,
  event: { moved?: unknown; added?: { element: Todo } },
) {
  const column = columnLists.value[priority] || [];
  const items: ReorderItem[] = column.map((t, idx) => ({ id: t.id, position: idx }));
  if (event.moved) {
    emit('reorder', items);
  }
  if (event.added?.element) {
    emit('priority-change', { todo: event.added.element, newPriority: priority, items });
  }
}
</script>

<style scoped>
.scrollbar-thin {
  scrollbar-width: thin;
}
</style>
