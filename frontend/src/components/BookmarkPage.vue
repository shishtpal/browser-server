<template>
  <div class="max-w-6xl mx-auto p-4">
    <h1 class="text-3xl font-bold text-gray-800 mb-6">Bookmarks</h1>

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
      <button @click="loadBookmarks" class="ml-4 underline font-medium">Retry</button>
    </div>

    <template v-else-if="selectedUserId">
      <!-- Add Form -->
      <form @submit.prevent="addBookmark" class="bg-white rounded-lg shadow p-4 mb-6">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <input v-model="newTitle" type="text" placeholder="Title" required
            class="px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
          <input v-model="newUrl" type="url" placeholder="URL (https://...)" required
            class="px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
          <input v-model="newDescription" type="text" placeholder="Description (optional)"
            class="px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
          <input v-model="newTags" type="text" placeholder="Tags: comma, separated"
            class="px-3 py-2 border border-gray-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-blue-500" />
        </div>
        <button type="submit" class="mt-3 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition">
          Add Bookmark
        </button>
      </form>

      <!-- Active Tag Filter -->
      <div v-if="activeTagFilter" class="mb-4 flex items-center gap-2">
        <span class="text-sm text-gray-500">Filtering by tag:</span>
        <span class="px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm font-medium">{{ activeTagFilter }}</span>
        <button @click="activeTagFilter = null; loadBookmarks()" class="text-sm text-red-500 hover:underline">Clear</button>
      </div>

      <!-- Empty State -->
      <div v-if="bookmarks.length === 0" class="text-center py-12 text-gray-400">
        <svg class="mx-auto h-12 w-12 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
        </svg>
        <p>No bookmarks yet — save one above!</p>
      </div>

      <!-- Bookmark Cards Grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div v-for="b in bookmarks" :key="b.id" class="bg-white rounded-lg shadow p-4 flex flex-col">
          <h3 class="font-semibold text-gray-800 truncate">{{ b.title }}</h3>
          <a :href="b.url" target="_blank" rel="noopener" class="text-sm text-blue-600 hover:underline truncate mb-2">{{ b.url }}</a>
          <p v-if="b.description" class="text-sm text-gray-500 mb-3">{{ b.description }}</p>

          <!-- Tags -->
          <div class="flex flex-wrap gap-1 mb-3 mt-auto">
            <button
              v-for="tag in b.tags"
              :key="tag"
              @click="filterByTag(tag)"
              class="px-2 py-0.5 bg-gray-100 text-gray-600 rounded text-xs hover:bg-blue-100 hover:text-blue-700 transition cursor-pointer"
            >{{ tag }}</button>
          </div>

          <!-- Actions -->
          <div class="flex gap-2 mt-2">
            <button @click="openEdit(b)" class="px-3 py-1 text-sm bg-gray-200 text-gray-700 rounded hover:bg-gray-300 transition">Edit</button>
            <button @click="removeBookmark(b.id)" class="px-3 py-1 text-sm bg-red-100 text-red-700 rounded hover:bg-red-200 transition">Delete</button>
          </div>
        </div>
      </div>
    </template>

    <!-- No User -->
    <div v-else class="text-center py-12 text-gray-400">
      <svg class="mx-auto h-12 w-12 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
      <p>Select a user to manage their bookmarks</p>
    </div>

    <!-- Edit Modal -->
    <div v-if="editing" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md mx-4">
        <h2 class="text-xl font-bold text-gray-800 mb-4">Edit Bookmark</h2>
        <div class="flex flex-col gap-3">
          <input v-model="editForm.title" type="text" placeholder="Title" required
            class="px-3 py-2 border border-gray-300 rounded-md" />
          <input v-model="editForm.url" type="url" placeholder="URL" required
            class="px-3 py-2 border border-gray-300 rounded-md" />
          <input v-model="editForm.description" type="text" placeholder="Description"
            class="px-3 py-2 border border-gray-300 rounded-md" />
          <input v-model="editForm.tagsStr" type="text" placeholder="Tags: comma, separated"
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
import { ref, watch } from 'vue'
import { getBookmarks, createBookmark, updateBookmark, deleteBookmark } from '../lib/api'
import { useUser } from '../composables/useUser'
import type { BookmarkResponse } from '../types'

const { users } = useUser()

const selectedUserId = ref<number | null>(null)
const bookmarks = ref<BookmarkResponse[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const activeTagFilter = ref<string | null>(null)

const newTitle = ref('')
const newUrl = ref('')
const newDescription = ref('')
const newTags = ref('')

const editing = ref<BookmarkResponse | null>(null)
const editForm = ref({ title: '', url: '', description: '', tagsStr: '' })

const loadBookmarks = async () => {
  if (!selectedUserId.value) return
  isLoading.value = true
  error.value = null
  try {
    bookmarks.value = await getBookmarks(selectedUserId.value, activeTagFilter.value || undefined)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load bookmarks'
  } finally {
    isLoading.value = false
  }
}

const addBookmark = async () => {
  if (!selectedUserId.value || !newTitle.value.trim() || !newUrl.value.trim()) return
  const tagList = newTags.value
    ? newTags.value.split(',').map(t => t.trim()).filter(Boolean)
    : []
  try {
    await createBookmark({
      user_id: selectedUserId.value,
      title: newTitle.value.trim(),
      url: newUrl.value.trim(),
      description: newDescription.value.trim() || undefined,
      tags: tagList.length ? tagList : undefined,
    })
    newTitle.value = ''
    newUrl.value = ''
    newDescription.value = ''
    newTags.value = ''
    await loadBookmarks()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to add bookmark'
  }
}

const filterByTag = (tag: string) => {
  activeTagFilter.value = tag
  loadBookmarks()
}

const openEdit = (b: BookmarkResponse) => {
  editing.value = b
  editForm.value = {
    title: b.title,
    url: b.url,
    description: b.description,
    tagsStr: b.tags.join(', '),
  }
}

const saveEdit = async () => {
  if (!editing.value) return
  const tagList = editForm.value.tagsStr
    ? editForm.value.tagsStr.split(',').map(t => t.trim()).filter(Boolean)
    : []
  try {
    await updateBookmark(editing.value.id, {
      title: editForm.value.title,
      url: editForm.value.url,
      description: editForm.value.description,
      tags: tagList.length ? tagList : [],
    } as any)
    editing.value = null
    await loadBookmarks()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to update bookmark'
  }
}

const removeBookmark = async (id: number) => {
  if (!confirm('Delete this bookmark?')) return
  try {
    await deleteBookmark(id)
    await loadBookmarks()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to delete bookmark'
  }
}

watch(selectedUserId, () => {
  loadBookmarks()
})
</script>
