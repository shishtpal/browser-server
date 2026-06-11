<template>
  <div class="max-w-6xl mx-auto p-4">
    <h1 class="text-3xl font-bold text-gray-800 mb-6">Wallet</h1>

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
      <button @click="loadWallet" class="ml-4 underline font-medium">Retry</button>
    </div>

    <template v-else-if="selectedUserId">
      <!-- Add Form -->
      <form @submit.prevent="addEntry" class="bg-white rounded-lg shadow p-4 mb-6">
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          <input v-model="newWebsite" type="text" placeholder="Website (e.g. github.com)" required
            class="px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
          <input v-model="newUsername" type="text" placeholder="Username" required
            class="px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
          <input v-model="newPassword" type="password" placeholder="Password" required
            class="px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
          <input v-model="newDescription" type="text" placeholder="Description (optional)"
            class="px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
        </div>
        <button type="submit" class="mt-3 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition">
          Add Entry
        </button>
      </form>

      <!-- Website Filter -->
      <div class="mb-4">
        <input
          v-model="websiteFilter"
          type="text"
          placeholder="Filter by website..."
          class="w-full md:w-96 px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <!-- Empty State -->
      <div v-if="filteredEntries.length === 0" class="text-center py-12 text-gray-400">
        <svg class="mx-auto h-12 w-12 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
        </svg>
        <p v-if="walletEntries.length === 0">No saved credentials yet — add one above!</p>
        <p v-else>No entries matching that website</p>
      </div>

      <!-- Table -->
      <div v-else class="bg-white rounded-lg shadow overflow-hidden">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Website</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Username</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Password</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase hidden md:table-cell">Description</th>
                <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200">
              <tr v-for="e in filteredEntries" :key="e.id" class="hover:bg-gray-50">
                <td class="px-4 py-3 text-sm text-gray-800 font-medium">{{ e.website }}</td>
                <td class="px-4 py-3 text-sm text-gray-600">{{ e.username }}</td>
                <td class="px-4 py-3 text-sm text-gray-600 font-mono">
                  <span v-if="revealedPasswords[e.id]">{{ e.password }}</span>
                  <span v-else>••••••••</span>
                  <button @click="toggleReveal(e.id)" class="ml-2 text-gray-400 hover:text-gray-600">
                    <svg v-if="!revealedPasswords[e.id]" class="w-4 h-4 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    </svg>
                    <svg v-else class="w-4 h-4 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                    </svg>
                  </button>
                </td>
                <td class="px-4 py-3 text-sm text-gray-500 hidden md:table-cell">{{ e.description }}</td>
                <td class="px-4 py-3 text-right">
                  <button @click="openEdit(e)" class="px-3 py-1 text-sm bg-gray-200 text-gray-700 rounded hover:bg-gray-300 transition mr-1">Edit</button>
                  <button @click="removeEntry(e.id)" class="px-3 py-1 text-sm bg-red-100 text-red-700 rounded hover:bg-red-200 transition">Delete</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <!-- No User -->
    <div v-else class="text-center py-12 text-gray-400">
      <svg class="mx-auto h-12 w-12 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
      <p>Select a user to manage their wallet</p>
    </div>

    <!-- Edit Modal -->
    <div v-if="editing" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md mx-4">
        <h2 class="text-xl font-bold text-gray-800 mb-4">Edit Wallet Entry</h2>
        <div class="flex flex-col gap-3">
          <input v-model="editForm.website" type="text" placeholder="Website" required
            class="px-3 py-2 border border-gray-300 rounded-md" />
          <input v-model="editForm.username" type="text" placeholder="Username" required
            class="px-3 py-2 border border-gray-300 rounded-md" />
          <input v-model="editForm.password" type="text" placeholder="Password" required
            class="px-3 py-2 border border-gray-300 rounded-md" />
          <input v-model="editForm.description" type="text" placeholder="Description"
            class="px-3 py-2 border border-gray-300 rounded-md" />
        </div>
        <div class="flex gap-2 mt-4 justify-end">
          <button @click="editing = null" class="px-4 py-2 bg-gray-300 text-gray-700 rounded-md hover:bg-gray-400">Cancel</button>
          <button @click="saveEdit" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700">Save</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { getWallet, createWalletEntry, updateWalletEntry, deleteWalletEntry } from '../lib/api'
import { useUser } from '../composables/useUser'
import type { WalletEntry } from '../types'

const { users } = useUser()

const selectedUserId = ref<number | null>(null)
const walletEntries = ref<WalletEntry[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const websiteFilter = ref('')

const newWebsite = ref('')
const newUsername = ref('')
const newPassword = ref('')
const newDescription = ref('')

const revealedPasswords = reactive<Record<number, boolean>>({})

const editing = ref<WalletEntry | null>(null)
const editForm = ref({ website: '', username: '', password: '', description: '' })

const filteredEntries = computed(() => {
  if (!websiteFilter.value.trim()) return walletEntries.value
  const q = websiteFilter.value.toLowerCase()
  return walletEntries.value.filter(e => e.website.toLowerCase().includes(q))
})

const loadWallet = async () => {
  if (!selectedUserId.value) return
  isLoading.value = true
  error.value = null
  try {
    walletEntries.value = await getWallet(selectedUserId.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load wallet'
  } finally {
    isLoading.value = false
  }
}

const addEntry = async () => {
  if (!selectedUserId.value || !newWebsite.value.trim() || !newUsername.value.trim() || !newPassword.value) return
  try {
    await createWalletEntry({
      user_id: selectedUserId.value,
      website: newWebsite.value.trim(),
      username: newUsername.value.trim(),
      password: newPassword.value,
      description: newDescription.value.trim() || undefined,
    })
    newWebsite.value = ''
    newUsername.value = ''
    newPassword.value = ''
    newDescription.value = ''
    await loadWallet()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to add entry'
  }
}

const toggleReveal = (id: number) => {
  revealedPasswords[id] = !revealedPasswords[id]
}

const openEdit = (e: WalletEntry) => {
  editing.value = e
  editForm.value = {
    website: e.website,
    username: e.username,
    password: e.password,
    description: e.description,
  }
}

const saveEdit = async () => {
  if (!editing.value) return
  try {
    await updateWalletEntry(editing.value.id, editForm.value)
    editing.value = null
    await loadWallet()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to update entry'
  }
}

const removeEntry = async (id: number) => {
  if (!confirm('Delete this wallet entry?')) return
  try {
    await deleteWalletEntry(id)
    await loadWallet()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to delete entry'
  }
}

watch(selectedUserId, () => {
  loadWallet()
})
</script>
