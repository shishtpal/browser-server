<template>
  <div class="max-w-4xl mx-auto p-4">
    <h1 class="text-3xl font-bold text-gray-800 mb-6">Users</h1>

    <!-- Add Form -->
    <form @submit.prevent="addUser" class="bg-white rounded-lg shadow p-4 mb-6">
      <h2 class="text-lg font-semibold text-gray-700 mb-3">Create User</h2>
      <div class="flex flex-col md:flex-row gap-3">
        <input v-model="newUsername" type="text" placeholder="Username" required
          class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
        <input v-model="newEmail" type="email" placeholder="Email (optional)"
          class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
        <button type="submit" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition">
          Create
        </button>
      </div>
      <div v-if="successMsg" class="mt-2 text-sm text-green-600">{{ successMsg }}</div>
    </form>

    <!-- Loading -->
    <div v-if="isLoading" class="flex justify-center py-12">
      <div class="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full"></div>
      <span class="ml-3 text-gray-500">Loading...</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
      {{ error }}
      <button @click="loadUsers" class="ml-4 underline font-medium">Retry</button>
    </div>

    <!-- Empty State -->
    <div v-else-if="usersList.length === 0" class="text-center py-12 text-gray-400">
      <svg class="mx-auto h-12 w-12 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
      </svg>
      <p>No users yet — create one above!</p>
    </div>

    <!-- Users Table -->
    <div v-else class="bg-white rounded-lg shadow overflow-hidden">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">ID</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Username</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Email</th>
            <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
          <tr v-for="u in usersList" :key="u.id" class="hover:bg-gray-50">
            <td class="px-4 py-3 text-sm text-gray-500">{{ u.id }}</td>
            <td class="px-4 py-3 text-sm font-medium text-gray-800">{{ u.username }}</td>
            <td class="px-4 py-3 text-sm text-gray-600">{{ u.email || '—' }}</td>
            <td class="px-4 py-3 text-right">
              <button @click="removeUser(u.id)" class="px-3 py-1 text-sm bg-red-100 text-red-700 rounded hover:bg-red-200 transition">
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { getUsers, createUser, deleteUser } from '../lib/api'
import type { User } from '../types'

const usersList = ref<User[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const successMsg = ref('')

const newUsername = ref('')
const newEmail = ref('')

const loadUsers = async () => {
  isLoading.value = true
  error.value = null
  try {
    usersList.value = await getUsers()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load users'
  } finally {
    isLoading.value = false
  }
}

const addUser = async () => {
  if (!newUsername.value.trim()) return
  try {
    await createUser({
      username: newUsername.value.trim(),
      email: newEmail.value.trim() || undefined,
    })
    successMsg.value = `User "${newUsername.value.trim()}" created!`
    newUsername.value = ''
    newEmail.value = ''
    await loadUsers()
    setTimeout(() => { successMsg.value = '' }, 3000)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to create user'
  }
}

const removeUser = async (id: number) => {
  if (!confirm('Delete this user? This will remove all their data (todos, bookmarks, history, wallet entries).')) return
  try {
    await deleteUser(id)
    await loadUsers()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to delete user'
  }
}

loadUsers()
</script>
