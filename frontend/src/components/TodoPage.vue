<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Task manager" title="Todos" color="indigo">
      <template #stats>
        <StatCard :value="totalCount" label="Total" variant="dark" color="indigo" />
        <StatCard :value="activeCount" label="Active" variant="primary" color="indigo" />
        <StatCard v-if="inProgressCount > 0" :value="inProgressCount" label="In Progress" variant="primary" color="blue" />
        <StatCard :value="completedCount" label="Done" variant="secondary" color="indigo" />
        <StatCard v-if="archivedCount > 0" :value="archivedCount" label="Archived" variant="secondary" color="indigo" />
        <StatCard v-if="overdueCount > 0" :value="overdueCount" label="Overdue" variant="dark" color="amber" />
      </template>
      <template #actions>
        <UserSelector id="todo-user" v-model="selectedUserId" :users="users" color="indigo" />
        <Button variant="gradient-indigo" size="sm" :disabled="!selectedUserId" @click="openCreateModal()">
          <span class="flex items-center gap-1">
            <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 5v14M5 12h14" />
            </svg>
            Add Todo
          </span>
        </Button>
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
      <div v-if="activeFilter === 'archived'" class="mb-4 flex items-center gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-500/20 dark:bg-amber-950/30 dark:text-amber-300">
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
            <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
              <template v-for="todo in listTodos" :key="todo.id">
                <TodoTableRow
                  :todo="todo"
                  :expanded="!collapsedTodoIds.has(todo.id)"
                  draggable="true"
                  @toggle="toggleTodo"
                  @toggle-pin="togglePinned"
                  @archive="archiveTodo"
                  @restore="restoreTodo"
                  @toggle-expand="toggleSubtaskRow"
                  @start-edit="openEditModal"
                  @delete="confirmDelete"
                  @view-screenshot="openScreenshot"
                  @toggle-subtask="toggleTodo"
                  @mousedown="onRowMouseDown"
                  @dragstart="onRowDragStart($event, todo.id)"
                  @dragover.prevent="onRowDragOver($event, todo.id)"
                  @drop="onRowDrop($event, todo.id)"
                  @dragend="onRowDragEnd"
                />
                <tr v-if="(todo.subtasks?.length || 0) > 0 && !collapsedTodoIds.has(todo.id)" class="bg-indigo-50/40 dark:bg-slate-800/40">
                  <td colspan="8" class="px-4 py-3">
                    <TodoSubtaskList :todo="todo" :default-expanded="true" @toggle-subtask="onSubtaskToggled" />
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>

        <div v-else-if="view === 'kanban'">
          <TodoKanbanBoard
            :todos="displayedTodos"
            @toggle="toggleTodo"
            @toggle-subtask="onSubtaskToggled"
            @toggle-pin="togglePinned"
            @archive="archiveTodo"
            @restore="restoreTodo"
            @view-screenshot="openScreenshot"
            @start-edit="openEditModal"
            @delete="confirmDelete"
            @reorder="onKanbanReorder"
            @priority-change="onKanbanPriorityChange"
          />
        </div>

        <draggable v-if="view === 'list'" v-model="listTodos" item-key="id" handle=".drag-handle" @end="onListDragEnd" tag="ul" class="space-y-2 md:hidden">
          <template #item="{ element: todo }">
            <TodoCard
              :todo="todo"
              @toggle="toggleTodo"
              @toggle-pin="togglePinned"
              @archive="archiveTodo"
              @restore="restoreTodo"
              @start-edit="openEditModal"
              @delete="confirmDelete"
              @view-screenshot="openScreenshot"
              @toggle-subtask="onSubtaskToggled"
            />
          </template>
        </draggable>
      </div>
    </div>

    <Modal :open="screenshotModal.open" :title="screenshotModal.title" @close="screenshotModal.open = false" fullscreen>
      <img :src="screenshotModal.url" class="w-full h-full rounded-lg border border-gray-200 object-contain dark:border-slate-700" />
    </Modal>

    <CalendarTodoModal
      :open="modalOpen"
      :editing-todo="editingTodo"
      :initial-due-date="modalDueDate"
      :user-id="selectedUserId!"
      @close="closeModal"
      @submit="handleCreate"
      @update="handleUpdate"
      @delete="handleDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { Todo, TodoPriority, CreateTodoInput } from '../types'
import draggable from 'vuedraggable'
import { useUser } from '../composables/useUser'
import { useTodos } from '../composables/useTodos'
import PageHeader from './ui/PageHeader.vue'
import StatCard from './ui/StatCard.vue'
import UserSelector from './ui/UserSelector.vue'
import Button from './ui/Button.vue'
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
import TodoSubtaskList from './todos/TodoSubtaskList.vue'
import CalendarTodoModal from './calendar/CalendarTodoModal.vue'
import { useModal } from '@browser-server/shared-modal'
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
  activeFilter,
  searchQuery,
  filters,
  totalCount,
  activeCount,
  inProgressCount,
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
  removeTodo,
  priority,
  dueDate,
  tags,
  sort,
  reorder,
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

// ── Native HTML5 drag for desktop table rows ──────────────────────────
const dragId = ref<number | null>(null)
const dragAllowed = ref(false)

function onRowMouseDown(event: MouseEvent) {
  // Allow drag only when initiated from the drag handle
  dragAllowed.value = !!(event.target && (event.target as HTMLElement).closest('.drag-handle'))
}

function onRowDragStart(event: DragEvent, id: number) {
  if (!dragAllowed.value) {
    event.preventDefault()
    return
  }
  dragId.value = id
  if (event.dataTransfer) {
    event.dataTransfer.setData('text/plain', String(id))
    event.dataTransfer.effectAllowed = 'move'
  }
}

function onRowDragOver(event: DragEvent, id: number) {
  if (dragId.value === null || dragId.value === id) return
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
}

function onRowDrop(_event: DragEvent, id: number) {
  if (dragId.value === null || dragId.value === id) {
    dragId.value = null
    return
  }
  const fromIdx = listTodos.value.findIndex(t => t.id === dragId.value)
  const toIdx = listTodos.value.findIndex(t => t.id === id)
  dragId.value = null
  if (fromIdx === -1 || toIdx === -1) return
  const moved = listTodos.value.splice(fromIdx, 1)[0]
  listTodos.value.splice(toIdx, 0, moved)
  reorderTodos(listTodos.value.map((t, idx) => ({ id: t.id, position: idx }))).then(loadTodos)
}

function onRowDragEnd() {
  dragId.value = null
  dragAllowed.value = false
}

const collapsedTodoIds = ref<Set<number>>(new Set())

function toggleSubtaskRow(id: number) {
  const next = new Set(collapsedTodoIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  collapsedTodoIds.value = next
}

function onSubtaskToggled() {
  loadTodos()
}

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

// ── New / Edit todo modal ──────────────────────────────────────────────
const modalOpen = ref(false)
const editingTodo = ref<Todo | null>(null)
const modalDueDate = ref('')

function openCreateModal() {
  if (!selectedUserId.value) return
  editingTodo.value = null
  modalDueDate.value = ''
  modalOpen.value = true
}

function openEditModal(todo: Todo) {
  editingTodo.value = todo
  modalDueDate.value = todo.start_date || ''
  modalOpen.value = true
}

function closeModal() {
  modalOpen.value = false
  editingTodo.value = null
}

async function handleCreate(data: CreateTodoInput) {
  await addTodo(data)
}

async function handleUpdate(id: number, data: Partial<Todo>) {
  await updateTodo(id, data)
  await loadTodos()
}

async function handleDelete() {
  if (!editingTodo.value) return
  const id = editingTodo.value.id
  closeModal()
  confirmDelete(id)
}

// ── Delete confirmation (imperative modal) ────────────────────────────
const { confirmDelete: modalConfirmDelete } = useModal()

async function confirmDelete(id: number) {
  const confirmed = await modalConfirmDelete(
    'Delete this todo?',
    "This action cannot be undone. The todo and its data will be permanently removed.",
  )
  if (confirmed) {
    await removeTodo(id)
  }
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
