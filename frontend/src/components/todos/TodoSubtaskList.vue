<template>
  <div class="mt-2 rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-slate-700 dark:bg-slate-800/60">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <button type="button" @click="expanded = !expanded" class="inline-flex items-center gap-1 text-[10px] font-black text-slate-500 transition hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200">
          <svg class="h-3 w-3 shrink-0 transition-transform" :class="expanded ? 'rotate-90' : ''" fill="currentColor" viewBox="0 0 24 24"><path d="M9 6l6 6-6 6" /></svg>
          Subtasks ({{ subtasks.length }})
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
          <div class="group flex items-center gap-2 rounded-md bg-white p-2 transition dark:bg-slate-800">
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
              @click="onToggleSubtask(element)"
              class="grid h-4 w-4 shrink-0 place-items-center rounded-full border-2 transition"
              :class="element.status === 'completed' ? 'border-emerald-500 bg-emerald-500 text-white' : 'border-gray-300 text-transparent hover:border-indigo-400 dark:border-slate-600 dark:hover:border-indigo-400'"
            >
              <svg class="h-2.5 w-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
              </svg>
            </button>

            <!-- Inline edit mode -->
            <template v-if="editingId === element.id">
              <input
                v-model="editTitle"
                @keydown.enter="saveEdit(element)"
                @keydown.escape="cancelEdit"
                class="flex-1 rounded-md border border-indigo-400 bg-white px-2 py-0.5 text-xs font-semibold text-slate-700 focus:outline-none dark:border-indigo-500 dark:bg-slate-800 dark:text-slate-200"
                ref="editInput"
              />
              <select
                v-model="editPriority"
                class="rounded-md border border-gray-300 bg-white px-1 py-0.5 text-[10px] font-black text-slate-600 focus:border-indigo-400 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-300"
              >
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
                <option value="urgent">Urgent</option>
              </select>
              <input
                v-model="editDueDate"
                type="date"
                class="rounded-md border border-gray-300 bg-white px-1 py-0.5 text-[10px] font-black text-slate-600 focus:border-indigo-400 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-300"
              />
              <button
                type="button"
                @click="saveEdit(element)"
                class="rounded-md bg-emerald-500 px-2 py-0.5 text-[10px] font-black text-white transition hover:bg-emerald-600"
                :disabled="!editTitle.trim()"
              >
                Save
              </button>
              <button
                type="button"
                @click="cancelEdit"
                class="rounded-md bg-gray-200 px-2 py-0.5 text-[10px] font-black text-slate-600 transition hover:bg-gray-300 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600"
              >
                Cancel
              </button>
            </template>

            <!-- Display mode -->
            <template v-else>
              <span
                class="flex-1 cursor-pointer text-xs font-semibold text-slate-700 dark:text-slate-200"
                :class="{ 'line-through text-slate-400 dark:text-slate-500': element.status === 'completed' }"
                :title="'Double-click to edit'"
                @dblclick="startEdit(element)"
              >
                {{ element.title }}
              </span>
              <TodoPriorityBadge :priority="(element.priority as any)" />
              <TodoDueDateBadge v-if="element.start_date" :due-date="element.start_date" :status="element.status" />
              <button
                type="button"
                @click="startEdit(element)"
                class="grid h-5 w-5 shrink-0 place-items-center rounded text-slate-400 opacity-0 transition hover:bg-gray-100 hover:text-indigo-500 group-hover:opacity-100 dark:hover:bg-slate-700 dark:hover:text-indigo-400"
                title="Edit subtask"
              >
                <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                </svg>
              </button>
              <button
                type="button"
                @click="onRemoveSubtask(element.id)"
                class="grid h-5 w-5 shrink-0 place-items-center rounded text-slate-400 opacity-0 transition hover:bg-red-50 hover:text-red-500 group-hover:opacity-100 dark:hover:bg-red-950/30 dark:hover:text-red-400"
                title="Delete subtask"
              >
                <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </template>


          </div>
        </template>
      </draggable>

      <form @submit.prevent="onAddSubtask" class="mt-2 flex items-center gap-2">
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
import type { Todo, TodoPriority } from '../../types'
import { ref, computed, watch, nextTick, type PropType } from 'vue'
import draggable from 'vuedraggable'
import { reorderTodos, updateTodo } from '../../lib/api'
import TodoPriorityBadge from './TodoPriorityBadge.vue'
import TodoDueDateBadge from './TodoDueDateBadge.vue'
import TodoSubtaskProgress from './TodoSubtaskProgress.vue'
import { useTodoSubtasks } from '../../composables/useTodoSubtasks'

const props = defineProps({
  todo: { type: Object as PropType<Todo>, required: true },
  defaultExpanded: { type: Boolean, default: true },
})

const emit = defineEmits<{
  'toggle-subtask': [todo: Todo]
}>()

const expanded = ref(props.defaultExpanded)

const userId = computed(() => props.todo.user_id)
const { subtasks, progress, addSubtask, toggleSubtask, removeSubtask } = useTodoSubtasks(props.todo.subtasks || [], computed(() => props.todo.id), userId)

// Sync when parent re-fetches
watch(() => props.todo.subtasks, (val) => {
  subtasks.value = [...(val || [])]
}, { deep: true })

const newTitle = ref('')

// ── Inline edit state ─────────────────────────────────────────────────
const editingId = ref<number | null>(null)
const editTitle = ref('')
const editPriority = ref<TodoPriority>('medium')
const editDueDate = ref('')
const editInput = ref<HTMLInputElement | null>(null)

function startEdit(subtask: Todo) {
  editingId.value = subtask.id
  editTitle.value = subtask.title
  editPriority.value = subtask.priority || 'medium'
  editDueDate.value = subtask.start_date ? subtask.start_date.slice(0, 10) : ''
  nextTick(() => {
    editInput.value?.focus()
    editInput.value?.select()
  })
}

function cancelEdit() {
  editingId.value = null
}

async function saveEdit(subtask: Todo) {
  const title = editTitle.value.trim()
  if (!title) return
  const updates: Record<string, any> = {}
  if (title !== subtask.title) updates.title = title
  if (editPriority.value !== subtask.priority) updates.priority = editPriority.value
  const newDate = editDueDate.value || null
  const oldDate = subtask.start_date ? subtask.start_date.slice(0, 10) : null
  if (newDate !== oldDate) updates.start_date = newDate

  if (Object.keys(updates).length > 0) {
    await updateTodo(subtask.id, updates)
  }
  editingId.value = null
}

// ── Actions ───────────────────────────────────────────────────────────
async function onToggleSubtask(subtask: Todo) {
  const updated = await toggleSubtask(subtask)
  if (updated) emit('toggle-subtask', updated)
}

function onRemoveSubtask(id: number) {
  removeSubtask(id)
}

async function onSubtaskEnd(event: any) {
  if (event.oldIndex === event.newIndex) return
  await reorderTodos(subtasks.value.map((t, idx) => ({ id: t.id, position: idx })))
}

function onAddSubtask() {
  if (!newTitle.value.trim()) return
  addSubtask(newTitle.value.trim())
  newTitle.value = ''
}
</script>
