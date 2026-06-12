<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Task manager" title="Todos" color="indigo">
      <template #stats>
        <StatCard :value="todos.length" label="Total" variant="dark" color="indigo" />
        <StatCard :value="activeCount" label="Active" variant="primary" color="indigo" />
        <StatCard :value="completedCount" label="Done" variant="secondary" color="indigo" />
      </template>
      <template #actions>
        <UserSelector id="todo-user" v-model="selectedUserId" :users="users" color="indigo" />
        <div class="flex gap-1">
          <FilterPill
            v-for="f in filters"
            :key="f.value"
            :active="activeFilter === f.value"
            @click="activeFilter = f.value"
          >
            {{ f.label }}
          </FilterPill>
        </div>
      </template>
    </PageHeader>

    <SelectUserPrompt title="Select a user to manage their todos" :users-count="users.length" :selected-user-id="selectedUserId" />

    <LoadingSpinner v-if="isLoading" message="Loading todos..." color="indigo" />

    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadTodos" />

    <div v-else-if="selectedUserId">
      <form @submit.prevent="addTodo" class="mb-4 rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90">
        <div class="flex items-center gap-2">
          <InputField v-model="newTitle" placeholder="What needs to be done?" required flex color="indigo" />
          <InputField v-model="newDescription" placeholder="Description" flex color="indigo" />
          <Button type="submit" variant="gradient-indigo" size="sm">Add</Button>
        </div>
      </form>

      <EmptyState
        v-if="filteredTodos.length === 0"
        title="No todos here"
        description="Add your first task above or change the filter."
        icon="default"
        color="indigo"
      />

      <div v-else>
        <div class="hidden overflow-hidden rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90 md:block">
          <table class="min-w-full divide-y divide-gray-200 transition-colors dark:divide-slate-700">
            <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
              <tr>
                <th class="w-10 px-3 py-3"></th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Title</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Description</th>
                <th class="w-28 px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Updated</th>
                <th class="w-24 px-3 py-3 text-right text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
              <TodoTableRow
                v-for="todo in filteredTodos"
                :key="todo.id"
                :todo="todo"
                :editing="editingId === todo.id"
                :initial-title="editingId === todo.id ? editTitle : ''"
                :initial-description="editingId === todo.id ? editDescription : ''"
                @toggle="toggleTodo"
                @start-edit="startEdit"
                @save-edit="saveEdit"
                @cancel-edit="cancelEdit"
                @delete="removeTodo"
              />
            </tbody>
          </table>
        </div>

        <ul class="space-y-2 md:hidden">
          <TodoCard
            v-for="todo in filteredTodos"
            :key="todo.id"
            :todo="todo"
            :editing="editingId === todo.id"
            :initial-title="editingId === todo.id ? editTitle : ''"
            :initial-description="editingId === todo.id ? editDescription : ''"
            @toggle="toggleTodo"
            @start-edit="startEdit"
            @save-edit="saveEdit"
            @cancel-edit="cancelEdit"
            @delete="removeTodo"
          />
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getTodos, createTodo, updateTodo, deleteTodo } from '../lib/api'
import { useUser } from '../composables/useUser'
import type { Todo } from '../types'
import PageHeader from './ui/PageHeader.vue'
import StatCard from './ui/StatCard.vue'
import UserSelector from './ui/UserSelector.vue'
import FilterPill from './ui/FilterPill.vue'
import LoadingSpinner from './ui/LoadingSpinner.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import EmptyState from './ui/EmptyState.vue'
import InputField from './ui/InputField.vue'
import Button from './ui/Button.vue'
import SelectUserPrompt from './ui/SelectUserPrompt.vue'
import TodoTableRow from './todos/TodoTableRow.vue'
import TodoCard from './todos/TodoCard.vue'

const { users, currentUserId, setUser, clearUser } = useUser()

const selectedUserId = ref<number | null>(currentUserId.value)
const todos = ref<Todo[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

const newTitle = ref('')
const newDescription = ref('')
const activeFilter = ref<'all' | 'active' | 'completed'>('all')
const filters = [
  { label: 'All', value: 'all' as const },
  { label: 'Active', value: 'active' as const },
  { label: 'Completed', value: 'completed' as const },
]

const editingId = ref<number | null>(null)
const editTitle = ref('')
const editDescription = ref('')

const activeCount = computed(() => todos.value.filter(t => !t.completed).length)
const completedCount = computed(() => todos.value.filter(t => t.completed).length)
const filteredTodos = computed(() => {
  if (activeFilter.value === 'active') return todos.value.filter(t => !t.completed)
  if (activeFilter.value === 'completed') return todos.value.filter(t => t.completed)
  return todos.value
})

const loadTodos = async () => {
  if (!selectedUserId.value) return
  isLoading.value = true
  error.value = null
  try {
    todos.value = await getTodos(selectedUserId.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load todos'
  } finally {
    isLoading.value = false
  }
}

const addTodo = async () => {
  if (!selectedUserId.value || !newTitle.value.trim()) return
  try {
    await createTodo({ user_id: selectedUserId.value, title: newTitle.value.trim(), description: newDescription.value.trim() || undefined })
    newTitle.value = ''
    newDescription.value = ''
    await loadTodos()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to add todo'
  }
}

const toggleTodo = async (todo: Todo) => {
  try {
    await updateTodo(todo.id, { ...todo, completed: !todo.completed })
    await loadTodos()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to update todo'
  }
}

const startEdit = (todo: Todo) => {
  editingId.value = todo.id
  editTitle.value = todo.title
  editDescription.value = todo.description
}

const cancelEdit = () => {
  editingId.value = null
}

const saveEdit = async (todo: Todo, title: string, description: string) => {
  try {
    await updateTodo(todo.id, { ...todo, title, description })
    editingId.value = null
    await loadTodos()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to update todo'
  }
}

const removeTodo = async (id: number) => {
  try {
    await deleteTodo(id)
    await loadTodos()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to delete todo'
  }
}

watch(selectedUserId, (id) => {
  if (id) {
    setUser(id)
    loadTodos()
  } else {
    clearUser()
    todos.value = []
  }
})

watch(activeFilter, () => {
  if (selectedUserId.value) loadTodos()
})

if (selectedUserId.value) {
  setUser(selectedUserId.value)
  loadTodos()
}
</script>
