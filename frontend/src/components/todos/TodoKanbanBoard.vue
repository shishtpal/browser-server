<template>
  <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
    <div v-for="p in PRIORITIES" :key="p.value" class="flex flex-col rounded-xl border border-gray-200 bg-gray-50/80 dark:border-slate-700/80 dark:bg-slate-800/60">
      <div :class="['mx-2 mt-2 rounded-lg px-2 py-1 text-center text-[10px] font-black', p.color]">
        {{ p.label }} ({{ columnCounts[p.value] }})
      </div>
      <draggable
        v-model="columnLists[p.value]"
        :group="{ name: 'todos', pull: true, put: true }"
        item-key="id"
        handle=".drag-handle"
        @change="onKanbanChange(p.value, $event)"
        tag="div"
        class="m-2 space-y-2 min-h-[200px]"
      >
        <template #item="{ element: todo }">
          <TodoKanbanCard
            :todo="todo"
            :expanded="expandedId === todo.id"
            @toggle="$emit('toggle', $event)"
            @toggle-expand="$emit('toggle-expand', $event)"
            @view-screenshot="$emit('view-screenshot', $event)"
            @start-edit="$emit('start-edit', $event)"
            @save-edit="(...args) => $emit('save-edit', ...args)"
            @cancel-edit="$emit('cancel-edit')"
            @delete="$emit('delete', $event)"
          />
        </template>
      </draggable>
      <div v-if="columnCounts[p.value] === 0" class="m-2 rounded-lg border border-dashed border-gray-300 p-3 text-center text-[10px] font-black text-slate-400 dark:border-slate-600 dark:text-slate-500">
        No {{ p.label.toLowerCase() }} tasks
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { Todo } from '../../types'
import type { ReorderItem } from '../../types'
import { PRIORITIES } from '../../composables/useTodoPriority'
import draggable from 'vuedraggable'
import TodoKanbanCard from './TodoKanbanCard.vue'

const emit = defineEmits<{
  toggle: [todo: Todo]
  'view-screenshot': [todo: Todo]
  'start-edit': [todo: Todo]
  'save-edit': [todo: Todo, title: string, description: string, priority: string, dueDate: string | null, tags: string[]]
  'cancel-edit': []
  delete: [id: number]
  reorder: [items: ReorderItem[]]
  'priority-change': [payload: { todo: Todo; newPriority: string; items: ReorderItem[] }]
  'toggle-expand': [id: number]
}>()

interface Props {
  todos: Todo[]
  expandedId?: number | null
}

const props = defineProps<Props>()

const columnLists = ref<Record<string, Todo[]>>({})

watch(() => props.todos, (newTodos) => {
  const map: Record<string, Todo[]> = {}
  for (const p of PRIORITIES) {
    map[p.value] = newTodos.filter(t => t.priority === p.value)
  }
  columnLists.value = map
}, { immediate: true, deep: true })

const columnCounts = computed(() => {
  const counts: Record<string, number> = {}
  for (const p of PRIORITIES) counts[p.value] = 0
  props.todos.forEach(t => { if (counts[t.priority] !== undefined) counts[t.priority]++ })
  return counts
})

function onKanbanChange(priority: string, event: any) {
  const column = columnLists.value[priority] || []
  const items: ReorderItem[] = column.map((t, idx) => ({ id: t.id, position: idx }))
  if (event.moved) {
    emit('reorder', items)
  }
  if (event.added && event.added.element) {
    emit('priority-change', { todo: event.added.element, newPriority: priority, items })
  }
}
</script>
