<template>
  <div class="max-w-4xl mx-auto p-4">
    <h1 class="text-3xl font-bold text-gray-800 mb-6">Todos</h1>

    <!-- User Selector -->
    <div class="mb-6">
      <label class="block text-sm font-medium text-gray-700 mb-1">Select User</label>
      <select
        v-model="selectedUserId"
        class="w-full md:w-64 px-3 py-2 border border-gray-300 rounded-md shadow-xs focus:outline-hidden focus:ring-2 focus:ring-blue-500"
      >
        <option :value="null">-- Choose a user --</option>
        <option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }}</option>
      </select>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="flex justify-center py-12">
      <div class="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full"></div>
      <span class="ml-3 text-gray-500">Loading...</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
      {{ error }}
      <button @click="loadTodos" class="ml-4 underline font-medium">Retry</button>
    </div>

    <template v-else-if="selectedUserId">
      <!-- Add Form -->
      <form @submit.prevent="addTodo" class="bg-white rounded-lg shadow p-4 mb-6">
        <div class="flex flex-col md:flex-row gap-3">
          <input
            v-model="newTitle"
            type="text"
            placeholder="What needs to be done?"
            required
            class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500"
          />
          <input
            v-model="newDescription"
            type="text"
            placeholder="Description (optional)"
            class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500"
          />
          <button
            type="submit"
            class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition"
          >
            Add
          </button>
        </div>
      </form>

      <!-- Filter Tabs -->
      <div class="flex gap-2 mb-4">
        <button
          v-for="f in filters"
          :key="f.value"
          @click="activeFilter = f.value"
          :class="[
            'px-4 py-1 rounded-full text-sm font-medium transition',
            activeFilter === f.value
              ? 'bg-blue-600 text-white'
              : 'bg-gray-200 text-gray-600 hover:bg-gray-300'
          ]"
        >{{ f.label }}</button>
      </div>

      <!-- Empty State -->
      <div v-if="filteredTodos.length === 0" class="text-center py-12 text-gray-400">
        <svg class="mx-auto h-12 w-12 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002-2h2a2 2 0 002 2M9 5h6" />
        </svg>
        <p>No todos yet — add one above!</p>
      </div>

      <!-- Todo List -->
      <ul class="space-y-2">
        <li v-for="todo in filteredTodos" :key="todo.id"
          class="bg-white rounded-lg shadow p-4 flex items-start gap-3"
          :class="{ 'opacity-60': todo.completed }"
        >
          <input
            type="checkbox"
            :checked="todo.completed"
            @change="toggleTodo(todo)"
            class="mt-1 h-5 w-5 text-blue-600 rounded focus:ring-blue-500"
          />

          <div class="flex-1 min-w-0">
            <template v-if="editingId === todo.id">
              <div class="flex flex-col gap-2">
                <input v-model="editTitle" class="px-2 py-1 border border-gray-300 rounded" />
                <input v-model="editDescription" class="px-2 py-1 border border-gray-300 rounded" />
                <div class="flex gap-2">
                  <button @click="saveEdit(todo)" class="px-3 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700">Save</button>
                  <button @click="cancelEdit" class="px-3 py-1 bg-gray-300 text-gray-700 text-sm rounded hover:bg-gray-400">Cancel</button>
                </div>
              </div>
            </template>
            <template v-else>
              <span :class="{ 'line-through text-gray-400': todo.completed }" class="font-medium text-gray-800">{{ todo.title }}</span>
              <p v-if="todo.description" class="text-sm text-gray-500 mt-1">{{ todo.description }}</p>
            </template>
          </div>

          <div class="flex gap-1 shrink-0" v-if="editingId !== todo.id">
            <button @click="startEdit(todo)" class="p-1 text-gray-400 hover:text-blue-600 transition" title="Edit">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </button>
            <button @click="removeTodo(todo.id)" class="p-1 text-gray-400 hover:text-red-600 transition" title="Delete">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </li>
      </ul>
    </template>

    <!-- No User Selected -->
    <div v-else class="text-center py-12 text-gray-400">
      <svg class="mx-auto h-12 w-12 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
      <p>Select a user to manage their todos</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getTodos, createTodo, updateTodo, deleteTodo } from '../lib/api'
import { useUser } from '../composables/useUser'
import type { Todo } from '../types'

const { users } = useUser()

const selectedUserId = ref<number | null>(null)
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

watch(selectedUserId, () => {
  loadTodos()
})
</script>
