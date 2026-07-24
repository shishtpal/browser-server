<template>
  <div class="mt-2 rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-slate-700 dark:bg-slate-800/60">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <button type="button" @click="expanded = !expanded" class="text-[10px] font-black text-slate-500 transition hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200">
          {{ expanded ? '−' : '+' }} Subtasks ({{ subtaskList.length }})
        </button>
        <TodoSubtaskProgress :done="progress.done" :total="progress.total" />
      </div>
    </div>

    <div v-if="expanded" class="mt-2 space-y-1">
      <draggable
        v-model="subtasks"
        item-key="id"
        handle=".drag-handle"
        @end="onSubtaskEnd"
        tag="div"
      >
        <template #item="{ element }">
          <div class="flex items-center gap-2 rounded-md bg-white p-2 transition dark:bg-slate-800">
            <button class="drag-handle cursor-grab active:cursor-grabbing text-slate-400 transition hover:text-slate-600" title="Drag to reorder">
              <svg class="h-3 w-3" fill="currentColor" viewBox="0 0 24 24">
                <circle cx="9" cy="6" r="1.5" />
                <circle cx="15" cy="6" r="1.5" />
                <circle cx="9" cy="12" r="1.5" />
                <circle cx="15" cy="12" r="1.5" />
                <circle cx="9" cy="18" r="1.5" />
                <circle cx="15" cy="18" r="1.5" />
              </svg>
            </button>
            <button
              type="button"
              @click="$emit('toggle-subtask', element)"
              class="grid h-4 w-4 place-items-center rounded-full border-2 transition"
              :class="element.completed ? 'border-emerald-500 bg-emerald-500 text-white' : 'border-gray-300 text-transparent hover:border-indigo-400 dark:border-slate-600 dark:hover:border-indigo-400'"
            >
              <svg class="h-2.5 w-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
              </svg>
            </button>
            <span class="flex-1 text-xs font-semibold text-slate-700 dark:text-slate-200" :class="{ 'line-through text-slate-400 dark:text-slate-500': element.completed }">
              {{ element.title }}
            </span>
            <TodoPriorityBadge :priority="(element.priority as any)" />
            <TodoDueDateBadge v-if="element.due_date" :due-date="element.due_date" :completed="element.completed" />
            <div v-if="isLoading" class="h-3 w-3 animate-spin rounded-full border-2 border-indigo-500 border-t-transparent"></div>
          </div>
        </template>
      </draggable>

      <form @submit.prevent="onAddSubtask" class="flex items-center gap-2">
        <input
          v-model="newTitle"
          placeholder="Add subtask..."
          class="flex-1 rounded-md border border-gray-300 bg-white px-2 py-1 text-xs font-semibold text-slate-700 focus:border-indigo-400 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        />
        <button
          type="submit"
          class="rounded-md bg-indigo-500 px-2 py-1 text-[10px] font-black text-white transition hover:bg-indigo-600"
          :disabled="!newTitle.trim()"
        >
          Add
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, type PropType } from 'vue'
import draggable from 'vuedraggable'
import { reorderTodos } from '../../lib/api'
import TodoPriorityBadge from './TodoPriorityBadge.vue'
import TodoDueDateBadge from './TodoDueDateBadge.vue'
import TodoSubtaskProgress from './TodoSubtaskProgress.vue'
import { useTodoSubtasks } from '../../composables/useTodoSubtasks'
import type { Todo } from '../../types'

const props = defineProps({
  todo: { type: Object as PropType<Todo>, required: true },
})

defineEmits<{
  'toggle-subtask': [todo: Todo]
}>()

const expanded = ref(false)

const userId = computed(() => props.todo.user_id)
const { subtasks, progress, isLoading, loadSubtasks, addSubtask, toggleSubtask, removeSubtask } = useTodoSubtasks(computed(() => props.todo.id), userId)

onMounted(() => {
  loadSubtasks()
})

const subtaskList = computed(() => subtasks.value)

const newTitle = ref('')

async function onSubtaskEnd(event: any) {
  if (event.oldIndex === event.newIndex) return
  await reorderTodos(subtasks.value.map((t, idx) => ({ id: t.id, position: idx })))
  await loadSubtasks()
}

function onAddSubtask() {
  if (!newTitle.value.trim()) return
  addSubtask(newTitle.value.trim())
  newTitle.value = ''
}
</script>
