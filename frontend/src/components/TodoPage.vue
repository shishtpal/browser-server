<template>
  <div class="mx-auto max-w-6xl px-4 py-6 sm:px-6 lg:px-8">
    <section class="mb-6 overflow-hidden rounded-[2rem] border border-gray-200/80 bg-white/90 p-5 shadow-2xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-8">
      <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p class="mb-2 inline-flex rounded-full bg-indigo-50 px-3 py-1 text-xs font-bold uppercase tracking-[0.2em] text-indigo-600 transition-colors dark:bg-indigo-900/20 dark:text-indigo-400">Task manager</p>
          <h1 class="text-3xl font-black tracking-tight text-slate-900 transition-colors dark:text-white sm:text-5xl">Todos</h1>
          <p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600 transition-colors dark:text-slate-400 sm:text-base">Plan, track, and finish what matters with a clean personal task board.</p>
        </div>
        <div class="grid grid-cols-3 gap-3 sm:max-w-sm">
          <div class="rounded-3xl bg-slate-900 p-4 text-center text-white shadow-lg shadow-indigo-500/15 transition-colors dark:bg-slate-950">
            <div class="text-2xl font-black">{{ todos.length }}</div>
            <div class="text-xs font-semibold text-slate-300">Total</div>
          </div>
          <div class="rounded-3xl bg-indigo-600 p-4 text-center text-white shadow-lg shadow-indigo-500/20">
            <div class="text-2xl font-black">{{ activeCount }}</div>
            <div class="text-xs font-semibold text-indigo-100">Active</div>
          </div>
          <div class="rounded-3xl bg-emerald-500 p-4 text-center text-white shadow-lg shadow-emerald-500/20">
            <div class="text-2xl font-black">{{ completedCount }}</div>
            <div class="text-xs font-semibold text-emerald-50">Done</div>
          </div>
        </div>
      </div>
    </section>

    <section class="mb-6 rounded-3xl border border-gray-200 bg-white/90 p-4 shadow-xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-5">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div class="min-w-0 flex-1">
          <label class="text-sm font-bold text-slate-700 transition-colors dark:text-slate-300" for="todo-user">Select user</label>
          <select
            id="todo-user"
            v-model="selectedUserId"
            class="mt-2 w-full rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
          >
            <option :value="null">Choose a user</option>
            <option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }}</option>
          </select>
          <p v-if="users.length === 0" class="mt-2 text-sm text-slate-500 transition-colors dark:text-slate-400">No users yet. <a href="/users" class="font-bold text-indigo-600 transition-colors hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300">Create one</a> to start.</p>
        </div>
        <div class="rounded-2xl bg-gray-100 px-4 py-3 text-sm text-slate-500 transition-colors dark:bg-slate-700/50 dark:text-slate-400">
          Last updated: <span class="font-semibold text-slate-700 transition-colors dark:text-slate-300">{{ selectedUserLabel }}</span>
        </div>
      </div>
    </section>

    <div v-if="isLoading" class="flex justify-center py-16">
      <div class="h-10 w-10 animate-spin rounded-full border-4 border-indigo-500 border-t-transparent"></div>
      <span class="ml-3 self-center font-semibold text-slate-600 transition-colors dark:text-slate-400">Loading todos...</span>
    </div>

    <div v-else-if="error" class="mb-6 rounded-3xl border border-red-200 bg-red-50 p-4 text-sm font-semibold text-red-700 shadow-sm transition-colors dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-400">
      {{ error }}
      <button type="button" @click="loadTodos" class="ml-2 underline decoration-red-300 underline-offset-4 transition-colors dark:decoration-red-800">Retry</button>
    </div>

    <template v-else-if="selectedUserId">
      <form @submit.prevent="addTodo" class="mb-6 rounded-3xl border border-gray-200 bg-white p-4 shadow-xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-5">
        <div class="grid gap-3 md:grid-cols-[1fr_1fr_auto]">
          <input
            v-model="newTitle"
            type="text"
            placeholder="What needs to be done?"
            required
            class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
          />
          <input
            v-model="newDescription"
            type="text"
            placeholder="Description (optional)"
            class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
          />
          <button type="submit" class="rounded-2xl bg-gradient-to-r from-indigo-600 to-violet-600 px-5 py-3 text-sm font-black text-white shadow-lg shadow-indigo-500/25 transition hover:-translate-y-0.5 hover:shadow-xl hover:shadow-indigo-500/30 focus:outline-none focus:ring-4 focus:ring-indigo-200 dark:focus:ring-indigo-900/40">
            Add todo
          </button>
        </div>
      </form>

      <div class="mb-4 flex flex-wrap gap-2">
        <button
          v-for="f in filters"
          :key="f.value"
          type="button"
          @click="activeFilter = f.value"
          :class="[
            'rounded-full px-4 py-2 text-sm font-black transition focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:focus:ring-indigo-900/30',
            activeFilter === f.value
              ? 'bg-slate-900 text-white shadow-lg shadow-slate-900/20 dark:bg-white dark:text-slate-900 dark:shadow-white/10'
              : 'bg-white text-slate-600 hover:bg-gray-100 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700'
          ]"
        >{{ f.label }}</button>
      </div>

      <div v-if="filteredTodos.length === 0" class="rounded-[2rem] border border-dashed border-gray-300 bg-gray-50 p-10 text-center shadow-sm backdrop-blur-xl transition-colors dark:border-slate-600 dark:bg-slate-800/60">
        <div class="mx-auto grid h-14 w-14 place-items-center rounded-3xl bg-indigo-50 text-indigo-500 transition-colors dark:bg-indigo-900/20 dark:text-indigo-400">
          <svg class="h-7 w-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002-2h2a2 2 0 002 2M9 5h6" />
          </svg>
        </div>
        <h2 class="mt-4 text-lg font-black text-slate-800 transition-colors dark:text-slate-200">No todos here</h2>
        <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">Add your first task above or change the filter.</p>
      </div>

      <ul v-else class="space-y-3">
        <li v-for="todo in filteredTodos" :key="todo.id" class="group rounded-3xl border border-gray-200/80 bg-white p-4 shadow-sm transition hover:-translate-y-0.5 hover:border-indigo-200 hover:shadow-xl hover:shadow-gray-900/10 dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-indigo-500/30 dark:hover:shadow-slate-950/20">
          <div class="flex items-start gap-3">
            <button
              type="button"
              :aria-label="todo.completed ? 'Mark as active' : 'Mark as completed'"
              @click="toggleTodo(todo)"
              :class="['mt-0.5 grid h-6 w-6 shrink-0 place-items-center rounded-full border-2 transition focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:focus:ring-indigo-900/30', todo.completed ? 'border-emerald-500 bg-emerald-500 text-white' : 'border-gray-300 text-transparent hover:border-indigo-400 dark:border-slate-600 dark:hover:border-indigo-400']"
            >
              <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
              </svg>
            </button>

            <div class="min-w-0 flex-1">
              <template v-if="editingId === todo.id">
                <div class="grid gap-3 sm:grid-cols-2">
                  <input v-model="editTitle" class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30" />
                  <input v-model="editDescription" class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30" />
                </div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <button type="button" @click="saveEdit(todo)" class="rounded-full bg-emerald-500 px-4 py-2 text-sm font-black text-white shadow-sm transition hover:bg-emerald-600">Save</button>
                  <button type="button" @click="cancelEdit" class="rounded-full bg-gray-100 px-4 py-2 text-sm font-black text-slate-600 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600">Cancel</button>
                </div>
              </template>
              <template v-else>
                <div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                  <div class="min-w-0">
                    <span :class="['block truncate text-base font-black sm:text-lg', todo.completed ? 'text-slate-400 line-through dark:text-slate-500' : 'text-slate-900 dark:text-white']">{{ todo.title }}</span>
                    <p v-if="todo.description" class="mt-1 line-clamp-2 text-sm leading-6 text-slate-500 transition-colors dark:text-slate-400">{{ todo.description }}</p>
                  </div>
                  <span class="hidden shrink-0 rounded-full bg-gray-100 px-3 py-1 text-xs font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400 sm:inline-flex">{{ formatDate(todo.updated_at) }}</span>
                </div>
              </template>
            </div>

            <div class="flex gap-1 shrink-0" v-if="editingId !== todo.id">
              <button type="button" @click="startEdit(todo)" class="grid h-9 w-9 place-items-center rounded-2xl text-slate-400 transition hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400" title="Edit">
                <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
              </button>
              <button type="button" @click="removeTodo(todo.id)" class="grid h-9 w-9 place-items-center rounded-2xl text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400" title="Delete">
                <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
        </li>
      </ul>
    </template>

    <div v-else class="rounded-[2rem] border border-dashed border-gray-300 bg-gray-50 p-10 text-center shadow-sm backdrop-blur-xl transition-colors dark:border-slate-600 dark:bg-slate-800/60">
      <div class="mx-auto grid h-16 w-16 place-items-center rounded-3xl bg-gray-100 text-slate-400 transition-colors dark:bg-slate-700 dark:text-slate-500">
        <svg class="h-8 w-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </div>
      <h2 class="mt-4 text-lg font-black text-slate-800 transition-colors dark:text-slate-200">Choose a workspace</h2>
      <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">Select a user to manage their todos.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getTodos, createTodo, updateTodo, deleteTodo } from '../lib/api'
import { useUser } from '../composables/useUser'
import type { Todo } from '../types'

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
const selectedUserLabel = computed(() => users.value.find(u => u.id === selectedUserId.value)?.username || 'Not selected')

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

const saveEdit = async (todo: Todo) => {
  try {
    await updateTodo(todo.id, { ...todo, title: editTitle.value, description: editDescription.value })
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

const formatDate = (dateStr: string) => {
  const d = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'Just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  return d.toLocaleDateString()
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
