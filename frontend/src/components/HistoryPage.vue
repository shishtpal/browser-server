<template>
  <div class="max-w-4xl mx-auto p-4">
    <h1 class="text-3xl font-bold text-gray-800 mb-6">History</h1>

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
      <button @click="loadHistory" class="ml-4 underline font-medium">Retry</button>
    </div>

    <template v-else-if="selectedUserId">
      <!-- Add Form -->
      <form @submit.prevent="addEntry" class="bg-white rounded-lg shadow p-4 mb-6">
        <div class="flex flex-col md:flex-row gap-3">
          <input v-model="newUrl" type="url" placeholder="URL (https://...)" required
            class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
          <input v-model="newTitle" type="text" placeholder="Title" required
            class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
          <input v-model="newDuration" type="number" min="0" placeholder="Duration (sec)"
            class="w-36 px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
          <button type="submit" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition">
            Add
          </button>
        </div>
      </form>

      <!-- URL Filter -->
      <div class="mb-4">
        <input
          v-model="urlFilter"
          type="text"
          placeholder="Filter by URL..."
          class="w-full md:w-96 px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <!-- Empty State -->
      <div v-if="filteredHistory.length === 0" class="text-center py-12 text-gray-400">
        <svg class="mx-auto h-12 w-12 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p v-if="historyEntries.length === 0">No history yet — add an entry above!</p>
        <p v-else>No entries matching that URL</p>
      </div>

      <!-- Timeline -->
      <div class="space-y-2">
        <div v-for="h in filteredHistory" :key="h.id" class="bg-white rounded-lg shadow p-4">
          <div class="flex items-start justify-between gap-3">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <svg class="w-4 h-4 text-gray-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064" />
                </svg>
                <a :href="h.url" target="_blank" rel="noopener" class="text-blue-600 hover:underline font-medium truncate">{{ h.title }}</a>
              </div>
              <a :href="h.url" target="_blank" rel="noopener" class="text-sm text-gray-500 truncate block ml-6">{{ h.url }}</a>
              <div class="flex items-center gap-3 ml-6 mt-1">
                <span class="text-xs text-gray-400">{{ formatDate(h.visited_at) }}</span>
                <span v-if="h.duration > 0" class="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded">{{ formatDuration(h.duration) }}</span>
              </div>
            </div>
            <button @click="removeEntry(h.id)" class="p-1 text-gray-400 hover:text-red-600 transition shrink-0" title="Delete">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </template>

    <!-- No User -->
    <div v-else class="text-center py-12 text-gray-400">
      <svg class="mx-auto h-12 w-12 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
      <p>Select a user to view their browsing history</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getHistory, createHistory, deleteHistory } from '../lib/api'
import { useUser } from '../composables/useUser'
import type { History } from '../types'

const { users } = useUser()

const selectedUserId = ref<number | null>(null)
const historyEntries = ref<History[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const urlFilter = ref('')

const newUrl = ref('')
const newTitle = ref('')
const newDuration = ref('')

const filteredHistory = computed(() => {
  if (!urlFilter.value.trim()) return historyEntries.value
  const q = urlFilter.value.toLowerCase()
  return historyEntries.value.filter(h => h.url.toLowerCase().includes(q))
})

const loadHistory = async () => {
  if (!selectedUserId.value) return
  isLoading.value = true
  error.value = null
  try {
    historyEntries.value = await getHistory(selectedUserId.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load history'
  } finally {
    isLoading.value = false
  }
}

const addEntry = async () => {
  if (!selectedUserId.value || !newUrl.value.trim() || !newTitle.value.trim()) return
  try {
    await createHistory({
      user_id: selectedUserId.value,
      url: newUrl.value.trim(),
      title: newTitle.value.trim(),
      duration: newDuration.value ? Number(newDuration.value) : 0,
    })
    newUrl.value = ''
    newTitle.value = ''
    newDuration.value = ''
    await loadHistory()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to add history entry'
  }
}

const removeEntry = async (id: number) => {
  if (!confirm('Delete this history entry?')) return
  try {
    await deleteHistory(id)
    await loadHistory()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to delete entry'
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

const formatDuration = (sec: number) => {
  if (sec < 60) return `${sec}s`
  const m = Math.floor(sec / 60)
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  return `${h}h ${m % 60}m`
}

watch(selectedUserId, () => {
  loadHistory()
})
</script>
