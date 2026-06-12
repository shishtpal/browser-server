<template>
  <div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
    <section class="mb-6 overflow-hidden rounded-[2rem] border border-gray-200/80 bg-white/90 p-5 shadow-2xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-8">
      <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p class="mb-2 inline-flex rounded-full bg-cyan-50 px-3 py-1 text-xs font-bold uppercase tracking-[0.2em] text-cyan-700 transition-colors dark:bg-cyan-900/20 dark:text-cyan-400">Saved links</p>
          <h1 class="text-3xl font-black tracking-tight text-slate-900 transition-colors dark:text-white sm:text-5xl">Bookmarks</h1>
          <p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600 transition-colors dark:text-slate-400 sm:text-base">Collect useful pages, tag them, and jump back faster.</p>
        </div>
        <div class="grid grid-cols-2 gap-3 sm:max-w-sm sm:grid-cols-2">
          <div class="rounded-3xl bg-slate-900 p-4 text-center text-white shadow-lg shadow-cyan-500/15 transition-colors dark:bg-slate-950">
            <div class="text-2xl font-black">{{ bookmarks.length }}</div>
            <div class="text-xs font-semibold text-slate-300">Saved</div>
          </div>
          <div class="rounded-3xl bg-cyan-500 p-4 text-center text-white shadow-lg shadow-cyan-500/20">
            <div class="text-2xl font-black">{{ allTags.length }}</div>
            <div class="text-xs font-semibold text-cyan-50">Tags</div>
          </div>
        </div>
      </div>
    </section>

    <section class="mb-6 rounded-3xl border border-gray-200 bg-white/90 p-4 shadow-xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-5">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div class="min-w-0 flex-1">
          <label class="text-sm font-bold text-slate-700 transition-colors dark:text-slate-300" for="bookmark-user">Select user</label>
          <select
            id="bookmark-user"
            v-model="selectedUserId"
            class="mt-2 w-full rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
          >
            <option :value="null">Choose a user</option>
            <option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }}</option>
          </select>
          <p v-if="users.length === 0" class="mt-2 text-sm text-slate-500 transition-colors dark:text-slate-400">No users yet. <a href="/users" class="font-bold text-cyan-700 transition-colors hover:text-cyan-800 dark:text-cyan-400 dark:hover:text-cyan-300">Create one</a> to start.</p>
        </div>
      </div>
    </section>

    <div v-if="isLoading" class="flex justify-center py-16">
      <div class="h-10 w-10 animate-spin rounded-full border-4 border-cyan-500 border-t-transparent"></div>
      <span class="ml-3 self-center font-semibold text-slate-600 transition-colors dark:text-slate-400">Loading bookmarks...</span>
    </div>

    <div v-else-if="error" class="mb-6 rounded-3xl border border-red-200 bg-red-50 p-4 text-sm font-semibold text-red-700 shadow-sm transition-colors dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-400">
      {{ error }}
      <button type="button" @click="loadBookmarks" class="ml-2 underline decoration-red-300 underline-offset-4 transition-colors dark:decoration-red-800">Retry</button>
    </div>

    <template v-else-if="selectedUserId">
      <form @submit.prevent="addBookmark" class="mb-6 rounded-3xl border border-gray-200 bg-white p-4 shadow-xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-5">
        <div class="grid gap-3 md:grid-cols-2">
          <input v-model="newTitle" type="text" placeholder="Title" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30" />
          <input v-model="newUrl" type="url" placeholder="URL (https://...)" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30" />
          <input v-model="newDescription" type="text" placeholder="Description (optional)" class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30" />
          <input v-model="newTags" type="text" placeholder="Tags: comma, separated" class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30" />
        </div>
        <button type="submit" class="mt-4 rounded-2xl bg-gradient-to-r from-cyan-500 to-blue-600 px-5 py-3 text-sm font-black text-white shadow-lg shadow-cyan-500/25 transition hover:-translate-y-0.5 hover:shadow-xl hover:shadow-cyan-500/30 focus:outline-none focus:ring-4 focus:ring-cyan-200 dark:focus:ring-cyan-900/40">
          Add bookmark
        </button>
      </form>

      <div v-if="activeTagFilter" class="mb-4 flex flex-wrap items-center gap-2 rounded-3xl border border-cyan-200 bg-cyan-50/80 p-3 text-sm text-cyan-800 shadow-sm transition-colors dark:border-cyan-900/30 dark:bg-cyan-900/20 dark:text-cyan-300">
        <span class="font-bold">Filtering by tag:</span>
        <span class="rounded-full bg-white px-3 py-1 font-black text-cyan-700 shadow-sm transition-colors dark:bg-slate-800 dark:text-cyan-400">{{ activeTagFilter }}</span>
        <button type="button" @click="activeTagFilter = null" class="rounded-full bg-cyan-200 px-3 py-1 font-black text-cyan-800 transition hover:bg-cyan-300 dark:bg-cyan-800 dark:text-cyan-200 dark:hover:bg-cyan-700">Clear</button>
      </div>

      <div v-if="allTags.length" class="mb-4 flex flex-wrap gap-2">
        <button type="button" @click="activeTagFilter = null" :class="['rounded-full px-3 py-1 text-xs font-black transition', activeTagFilter === null ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900' : 'bg-white text-slate-600 hover:bg-gray-100 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700']">All</button>
        <button
          v-for="tag in allTags"
          :key="tag"
          type="button"
          @click="filterByTag(tag)"
          :class="['rounded-full px-3 py-1 text-xs font-black transition', activeTagFilter === tag ? 'bg-cyan-500 text-white shadow-md' : 'bg-white text-slate-600 hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400']"
        >#{{ tag }}</button>
      </div>

      <div v-if="bookmarks.length === 0" class="rounded-[2rem] border border-dashed border-gray-300 bg-gray-50 p-10 text-center shadow-sm backdrop-blur-xl transition-colors dark:border-slate-600 dark:bg-slate-800/60">
        <div class="mx-auto grid h-14 w-14 place-items-center rounded-3xl bg-cyan-50 text-cyan-500 transition-colors dark:bg-cyan-900/20 dark:text-cyan-400">
          <svg class="h-7 w-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
          </svg>
        </div>
        <h2 class="mt-4 text-lg font-black text-slate-800 transition-colors dark:text-slate-200">No bookmarks yet</h2>
        <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">Save your first link above.</p>
      </div>

      <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        <article v-for="b in bookmarks" :key="b.id" class="group flex min-h-72 flex-col overflow-hidden rounded-[1.75rem] border border-gray-200/80 bg-white shadow-sm transition hover:-translate-y-1 hover:border-cyan-200 hover:shadow-xl hover:shadow-gray-900/10 dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-cyan-500/30 dark:hover:shadow-slate-950/20">
          <div class="flex items-start justify-between gap-3 border-b border-gray-100 bg-gradient-to-br from-slate-900 to-slate-800 p-4 text-white transition-colors dark:from-slate-950 dark:to-slate-900">
            <div class="grid h-11 w-11 shrink-0 place-items-center rounded-2xl bg-white/10 font-black">{{ getInitial(b.title) }}</div>
            <div class="text-right">
              <h3 class="line-clamp-1 text-base font-black">{{ b.title }}</h3>
              <a :href="b.url" target="_blank" rel="noopener" class="line-clamp-1 text-xs text-cyan-200 hover:text-cyan-50">{{ formatHost(b.url) }}</a>
            </div>
          </div>
          <div class="flex flex-1 flex-col p-4">
            <a :href="b.url" target="_blank" rel="noopener" class="break-words text-sm font-bold text-blue-600 transition-colors hover:underline dark:text-blue-400">{{ b.url }}</a>
            <p v-if="b.description" class="mt-3 line-clamp-3 text-sm leading-6 text-slate-500 transition-colors dark:text-slate-400">{{ b.description }}</p>
            <div v-if="b.tags.length" class="mt-auto flex flex-wrap gap-2 pt-4">
              <button
                v-for="tag in b.tags"
                :key="tag"
                type="button"
                @click="filterByTag(tag)"
                class="rounded-full bg-gray-100 px-3 py-1 text-xs font-black text-slate-600 transition hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400"
              >#{{ tag }}</button>
            </div>
            <div class="mt-4 flex gap-2 border-t border-gray-100 pt-4 transition-colors dark:border-slate-700/50">
              <button type="button" @click="openEdit(b)" class="flex-1 rounded-2xl bg-gray-100 px-4 py-2.5 text-sm font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600">Edit</button>
              <button type="button" @click="removeBookmark(b.id)" class="flex-1 rounded-2xl bg-red-50 px-4 py-2.5 text-sm font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
            </div>
          </div>
        </article>
      </div>
    </template>

    <div v-else class="rounded-[2rem] border border-dashed border-gray-300 bg-gray-50 p-10 text-center shadow-sm backdrop-blur-xl transition-colors dark:border-slate-600 dark:bg-slate-800/60">
      <div class="mx-auto grid h-16 w-16 place-items-center rounded-3xl bg-gray-100 text-slate-400 transition-colors dark:bg-slate-700 dark:text-slate-500">
        <svg class="h-8 w-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </div>
      <h2 class="mt-4 text-lg font-black text-slate-800 transition-colors dark:text-slate-200">Choose a workspace</h2>
      <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">Select a user to manage their bookmarks.</p>
    </div>

    <div v-if="editing" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-4 backdrop-blur-sm">
      <div class="w-full max-w-lg rounded-[2rem] border border-gray-200 bg-white p-5 shadow-2xl shadow-gray-900/30 transition-colors dark:border-white/10 dark:bg-slate-800 dark:shadow-slate-950/30 sm:p-6">
        <div class="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 class="text-xl font-black text-slate-900 transition-colors dark:text-white">Edit bookmark</h2>
            <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">Update the saved link details.</p>
          </div>
          <button type="button" @click="editing = null" class="grid h-9 w-9 place-items-center rounded-2xl bg-gray-100 text-slate-500 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-400 dark:hover:bg-slate-600" aria-label="Close">×</button>
        </div>
        <div class="grid gap-3">
          <input v-model="editForm.title" type="text" placeholder="Title" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
          <input v-model="editForm.url" type="url" placeholder="URL" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
          <input v-model="editForm.description" type="text" placeholder="Description" class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
          <input v-model="editForm.tagsStr" type="text" placeholder="Tags: comma, separated" class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
        </div>
        <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button type="button" @click="editing = null" class="rounded-2xl bg-gray-100 px-5 py-3 text-sm font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600">Cancel</button>
          <button type="button" @click="saveEdit" class="rounded-2xl bg-gradient-to-r from-cyan-500 to-blue-600 px-5 py-3 text-sm font-black text-white shadow-lg shadow-cyan-500/20">Save changes</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getBookmarks, createBookmark, updateBookmark, deleteBookmark } from '../lib/api'
import { useUser } from '../composables/useUser'
import type { BookmarkResponse } from '../types'

const { users, currentUserId, setUser, clearUser } = useUser()

const selectedUserId = ref<number | null>(currentUserId.value)
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

const allTags = computed(() => Array.from(new Set(bookmarks.value.flatMap(b => b.tags))).sort())

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
}

const getInitial = (value: string) => value.trim().charAt(0).toUpperCase() || 'B'

const formatHost = (url: string) => {
  try {
    return new URL(url).host
  } catch {
    return url
  }
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
      user_id: editing.value.user_id,
      title: editForm.value.title,
      url: editForm.value.url,
      description: editForm.value.description,
      tags: tagList as any,
    })
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

watch(selectedUserId, (id) => {
  activeTagFilter.value = null
  if (id) {
    setUser(id)
    loadBookmarks()
  } else {
    clearUser()
    bookmarks.value = []
  }
})

watch(activeTagFilter, () => {
  if (selectedUserId.value) loadBookmarks()
})

if (selectedUserId.value) {
  setUser(selectedUserId.value)
  loadBookmarks()
}
</script>
