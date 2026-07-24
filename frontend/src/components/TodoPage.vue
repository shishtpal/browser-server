<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Task manager" title="Todos" color="indigo">
      <template #stats>
        <StatCard :value="totalCount" label="Total" variant="dark" color="indigo" />
        <StatCard :value="activeCount" label="Active" variant="primary" color="indigo" />
        <StatCard :value="completedCount" label="Done" variant="secondary" color="indigo" />
        <StatCard v-if="archivedCount > 0" :value="archivedCount" label="Archived" variant="secondary" color="indigo" />
        <StatCard v-if="overdueCount > 0" :value="overdueCount" label="Overdue" variant="dark" color="amber" />
      </template>
      <template #actions>
        <UserSelector id="todo-user" v-model="selectedUserId" :users="users" color="indigo" />
        <div class="flex flex-wrap items-center gap-1.5">
          <FilterPill
            v-for="f in filters"
            :key="f.value"
            :active="activeFilter === f.value"
            @click="activeFilter = f.value"
          >
            {{ f.label }}
          </FilterPill>
          <FilterPill
            v-if="selectedPriority"
            :active="true"
            @click="clearPriority()"
            variant="tag"
          >
            Priority: {{ selectedPriority }} ✕
          </FilterPill>
          <FilterPill
            v-if="dueDateFilter"
            :active="true"
            @click="clearDueDateFilter()"
            variant="tag"
          >
            {{ dueDateLabel }} ✕
          </FilterPill>
          <FilterPill
            v-if="selectedTag"
            :active="true"
            @click="selectTag(null)"
            variant="tag"
          >
            {{ selectedTag }} ✕
          </FilterPill>
        </div>
        <div class="flex flex-wrap items-center gap-1.5">
          <TodoViewToggle :view="view" @update:view="view = $event" />
          <TodoSortBar v-if="view === 'list'" :sort-field="sortField" :sort-dir="sortDir" @update:sort-field="setSort($event)" @toggle-dir="toggleSortDir()" />
        </div>
        <div class="flex flex-wrap items-center gap-1.5">
          <select
            v-model="dueDateFilter"
            class="rounded-lg border border-gray-300 bg-gray-50 px-2 py-1.5 text-[11px] font-black text-slate-700 focus:border-indigo-400 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          >
            <option :value="null">All due dates</option>
            <option value="overdue">Overdue</option>
            <option value="today">Due today</option>
            <option value="this_week">Due this week</option>
          </select>
          <select
            v-model="selectedTag"
            class="rounded-lg border border-gray-300 bg-gray-50 px-2 py-1.5 text-[11px] font-black text-slate-700 focus:border-indigo-400 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          >
            <option :value="null">All tags</option>
            <option v-for="t in allTags" :key="t" :value="t">{{ t }}</option>
          </select>
          <select
            v-model="selectedPriority"
            class="rounded-lg border border-gray-300 bg-gray-50 px-2 py-1.5 text-[11px] font-black text-slate-700 focus:border-indigo-400 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          >
            <option :value="null">All priorities</option>
            <option v-for="p in allPriorityOptions" :key="p.value" :value="p.value">{{ p.label }}</option>
          </select>
        </div>
      </template>
    </PageHeader>

    <SelectUserPrompt title="Select a user to manage their todos" :users-count="users.length" :selected-user-id="selectedUserId" />

    <LoadingSpinner v-if="isLoading" message="Loading todos..." color="indigo" />

    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadTodos" />

    <div v-else-if="selectedUserId">
      <TodoAddForm v-if="activeFilter !== 'archived'" class="mb-4" @submit="handleAddTodo" :existing-tags="allTags" />

      <div v-else class="mb-4 flex items-center gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-500/20 dark:bg-amber-950/30 dark:text-amber-300">
        <svg class="h-5 w-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M7 8V5h10v3m-9 0v11h8V8m-5 4h2" />
        </svg>
        <span><strong>Archived todos</strong> are hidden from your normal workspace. Restore one to make it active again.</span>
      </div>

      <div class="mb-4 rounded-2xl border border-gray-200/80 bg-white/90 p-3 shadow-sm dark:border-slate-700/80 dark:bg-slate-800/90">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
          <label class="relative min-w-0 flex-1">
            <span class="sr-only">Search todos</span>
            <svg class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m21 21-4.35-4.35m1.35-5.65a7 7 0 1 1-14 0 7 7 0 0 1 14 0Z" />
            </svg>
            <input
              v-model="searchQuery"
              type="search"
              placeholder="Search titles, descriptions, or tags..."
              class="w-full rounded-xl border border-gray-300 bg-gray-50 py-2.5 pl-10 pr-10 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-900/50 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
            />
            <button
              v-if="searchQuery"
              type="button"
              class="absolute right-2 top-1/2 grid h-7 w-7 -translate-y-1/2 place-items-center rounded-lg text-slate-400 transition hover:bg-gray-200 hover:text-slate-700 dark:hover:bg-slate-700 dark:hover:text-slate-200"
              aria-label="Clear todo search"
              @click="searchQuery = ''"
            >
              <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18 18 6M6 6l12 12" />
              </svg>
            </button>
          </label>
          <p class="shrink-0 text-xs font-bold text-slate-500 dark:text-slate-400" aria-live="polite">
            {{ resultSummary }}
          </p>
        </div>
      </div>

      <EmptyState
        v-if="displayedTodos.length === 0"
        :title="searchQuery ? 'No matching todos' : activeFilter === 'archived' ? 'Archive is empty' : 'No todos here'"
        :description="searchQuery ? `Nothing matches “${searchQuery}”. Try another search.` : activeFilter === 'archived' ? 'Completed todos you archive will appear here.' : 'Create your first task above or change the filters.'"
        :icon="searchQuery ? 'search' : 'default'"
        color="indigo"
      />

      <div v-else>
        <div v-if="view === 'list'" class="hidden overflow-hidden rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90 md:block">
          <table class="min-w-full divide-y divide-gray-200 transition-colors dark:divide-slate-700">
            <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
              <tr>
                <th class="w-14 px-3 py-3"></th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Title</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Description</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Due date</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Tags</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Updated</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Subtasks</th>
                <th class="w-44 px-3 py-3 text-right text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Actions</th>
              </tr>
            </thead>
            <draggable v-model="listTodos" item-key="id" handle=".drag-handle" @end="onListDragEnd" tag="tbody" class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
              <template #item="{ element: todo }">
                <TodoTableRow
                  :todo="todo"
                  :editing="editingId === todo.id"
                  :initial-title="editingId === todo.id ? editTitle : ''"
                  :initial-description="editingId === todo.id ? editDescription : ''"
                  :initial-priority="editingId === todo.id ? editPriority : ''"
                  :initial-due-date="editingId === todo.id ? editDueDate : null"
                  :initial-tags="editingId === todo.id ? editTags : []"
                  :expanded="expandedTodoId === todo.id"
                  @toggle="toggleTodo"
                  @toggle-pin="togglePinned"
                  @archive="archiveTodo"
                  @restore="restoreTodo"
                  @toggle-expand="toggleExpand"
                  @start-edit="startEdit"
                  @saveEdit="saveEdit"
                  @cancel-edit="cancelEdit"
                  @delete="removeTodo"
                  @view-screenshot="openScreenshot"
                />
              </template>
            </draggable>
          </table>
          <div v-if="expandedTodo" class="mt-1 rounded-xl border border-gray-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-800/90">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-black text-slate-700 dark:text-slate-200">Subtasks</h3>
              <button type="button" @click="toggleExpand(expandedTodo.id)" class="text-xs font-black text-slate-400 transition hover:text-slate-600">Close</button>
            </div>
            <TodoSubtaskList :todo="expandedTodo" @toggle-subtask="toggleTodo" />
          </div>
        </div>

        <div v-else-if="view === 'kanban'">
          <TodoKanbanBoard
            :todos="displayedTodos"
            :expanded-id="expandedTodoId"
            @toggle="toggleTodo"
            @toggle-pin="togglePinned"
            @archive="archiveTodo"
            @restore="restoreTodo"
            @toggle-expand="toggleExpand"
            @view-screenshot="openScreenshot"
            @start-edit="(t: Todo) => startEdit(t)"
            @save-edit="saveEdit"
            @cancel-edit="cancelEdit"
            @delete="removeTodo"
            @reorder="onKanbanReorder"
            @priority-change="onKanbanPriorityChange"
          />
        </div>

        <draggable v-if="view === 'list'" v-model="listTodos" item-key="id" handle=".drag-handle" @end="onListDragEnd" tag="ul" class="space-y-2 md:hidden">
          <template #item="{ element: todo }">
            <TodoCard
              :todo="todo"
              :editing="editingId === todo.id"
              :initial-title="editingId === todo.id ? editTitle : ''"
              :initial-description="editingId === todo.id ? editDescription : ''"
              :initial-priority="editingId === todo.id ? editPriority : ''"
              :initial-due-date="editingId === todo.id ? editDueDate : null"
              :initial-tags="editingId === todo.id ? editTags : []"
              :expanded="expandedTodoId === todo.id"
              @toggle="toggleTodo"
              @toggle-pin="togglePinned"
              @archive="archiveTodo"
              @restore="restoreTodo"
              @toggle-expand="toggleExpand"
              @start-edit="startEdit"
              @saveEdit="saveEdit"
              @cancel-edit="cancelEdit"
              @delete="removeTodo"
              @view-screenshot="openScreenshot"
            />
          </template>
        </draggable>
      </div>
    </div>

    <Modal :open="screenshotModal.open" :title="screenshotModal.title" @close="screenshotModal.open = false" fullscreen>
      <img :src="screenshotModal.url" class="w-full h-full rounded-lg border border-gray-200 object-contain dark:border-slate-700" />
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { Todo, TodoPriority } from '../types'
import draggable from 'vuedraggable'
import { useUser } from '../composables/useUser'
import { useTodos } from '../composables/useTodos'
import PageHeader from './ui/PageHeader.vue'
import StatCard from './ui/StatCard.vue'
import UserSelector from './ui/UserSelector.vue'
import FilterPill from './ui/FilterPill.vue'
import LoadingSpinner from './ui/LoadingSpinner.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import EmptyState from './ui/EmptyState.vue'
import SelectUserPrompt from './ui/SelectUserPrompt.vue'
import Modal from './ui/Modal.vue'
import TodoTableRow from './todos/TodoTableRow.vue'
import TodoCard from './todos/TodoCard.vue'
import TodoKanbanBoard from './todos/TodoKanbanBoard.vue'
import TodoViewToggle from './todos/TodoViewToggle.vue'
import TodoSortBar from './todos/TodoSortBar.vue'
import TodoAddForm from './todos/TodoAddForm.vue'
import TodoSubtaskList from './todos/TodoSubtaskList.vue'
import { getScreenshotUrl, reorderTodos, updateTodo } from '../lib/api'

const allPriorityOptions: { value: TodoPriority; label: string }[] = [
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
  { value: 'urgent', label: 'Urgent' },
]

const { users, currentUserId, setUser, clearUser } = useUser()
const selectedUserId = ref<number | null>(currentUserId.value)

const {
  todos,
  isLoading,
  error,
  newTitle,
  newDescription,
  newPriority,
  newDueDate,
  newTags,
  newMoreOpen,
  activeFilter,
  searchQuery,
  filters,
  editingId,
  editTitle,
  editDescription,
  editPriority,
  editDueDate,
  editTags,
  totalCount,
  activeCount,
  completedCount,
  archivedCount,
  overdueCount,
  displayedTodos,
  loadTodos,
  addTodo,
  toggleTodo,
  togglePinned,
  archiveTodo,
  restoreTodo,
  startEdit,
  cancelEdit,
  saveEdit,
  removeTodo,
  priority,
  dueDate,
  tags,
  sort,
  subtasks,
  reorder,
  expandedTodoId,
} = useTodos(selectedUserId)

// Vue only unwraps refs exposed as top-level template bindings. Keeping these
// nested would make v-for iterate the ComputedRef object instead of its value.
const { selectedPriority, clearPriority } = priority
const { dueDateFilter, clearDueDateFilter } = dueDate
const { allTags, selectedTag, selectTag } = tags
const { sortField, sortDir, setSort, toggleDir: toggleSortDir } = sort

watch(selectedUserId, (id) => {
  if (id) {
    setUser(id)
    loadTodos()
  } else {
    clearUser()
    todos.value = []
  }
})

if (selectedUserId.value) {
  setUser(selectedUserId.value)
  loadTodos()
}

const view = ref<'list' | 'kanban'>('list')

const listTodos = ref<Todo[]>([])

watch(displayedTodos, (val) => {
  listTodos.value = [...val]
}, { immediate: true })

async function onListDragEnd(event: any) {
  if (event.oldIndex === event.newIndex) return
  await reorderTodos(listTodos.value.map((t, idx) => ({ id: t.id, position: idx })))
  await loadTodos()
}

function toggleExpand(id: number) {
  expandedTodoId.value = expandedTodoId.value === id ? null : id
}

const expandedTodo = computed(() => {
  if (!expandedTodoId.value) return null
  return todos.value.find(t => t.id === expandedTodoId.value) || null
})

async function onKanbanReorder(items: { id: number; position: number }[]) {
  await reorderTodos(items)
  await loadTodos()
}

async function onKanbanPriorityChange(payload: { todo: Todo; newPriority: string; items: { id: number; position: number }[] }) {
  await reorderTodos(payload.items)
  await updateTodo(payload.todo.id, { priority: payload.newPriority })
  await loadTodos()
}

const screenshotModal = ref<{ open: boolean; url: string; title: string }>({
  open: false,
  url: '',
  title: '',
})

function openScreenshot(todo: Todo) {
  screenshotModal.value = {
    open: true,
    url: getScreenshotUrl(todo.id),
    title: todo.title,
  }
}

function handleAddTodo(data: { title: string; description?: string; priority?: string; due_date?: string | null; tags?: string[] }) {
  addTodo(data)
}

const dueDateLabel = computed(() => {
  if (!dueDate.dueDateFilter.value) return ''
  const labels: Record<string, string> = { overdue: 'Overdue', today: 'Today', this_week: 'This week' }
  return labels[dueDate.dueDateFilter.value] || ''
})

const resultSummary = computed(() => {
  const count = displayedTodos.value.length
  if (!searchQuery.value.trim()) return `${count} ${count === 1 ? 'todo' : 'todos'}`
  return `${count} ${count === 1 ? 'result' : 'results'}`
})
</script>
