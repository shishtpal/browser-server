<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <div class="mb-4 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex items-center gap-3">
        <div>
          <p class="mb-1 inline-flex rounded-full bg-cyan-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.2em] text-cyan-700 transition-colors dark:bg-cyan-900/20 dark:text-cyan-400">Saved links</p>
          <h1 class="text-2xl font-black tracking-tight text-slate-900 transition-colors dark:text-white sm:text-3xl">Bookmarks</h1>
        </div>
        <div class="flex items-center gap-2">
          <div class="rounded-xl bg-slate-900 px-3 py-2 text-center text-white shadow-lg shadow-cyan-500/15 transition-colors dark:bg-slate-950">
            <div class="text-sm font-black leading-none">{{ bookmarks.length }}</div>
            <div class="text-[10px] font-semibold text-slate-300 leading-none mt-0.5">Saved</div>
          </div>
          <div class="rounded-xl bg-cyan-500 px-3 py-2 text-center text-white shadow-lg shadow-cyan-500/20">
            <div class="text-sm font-black leading-none">{{ allTags.length }}</div>
            <div class="text-[10px] font-semibold text-cyan-50 leading-none mt-0.5">Tags</div>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <select
          id="bookmark-user"
          v-model="selectedUserId"
          class="rounded-xl border border-gray-300 bg-gray-50 px-3 py-2 text-xs font-semibold text-slate-700 shadow-sm transition focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30"
        >
          <option :value="null">All users</option>
          <option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }}</option>
        </select>
        <div v-if="allTags.length" class="flex flex-wrap gap-1">
          <button type="button" @click="activeTagFilter = null" :class="['rounded-lg px-2.5 py-1.5 text-[10px] font-black transition', activeTagFilter === null ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900' : 'bg-white text-slate-600 hover:bg-gray-100 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700']">All</button>
          <button
            v-for="tag in allTags"
            :key="tag"
            type="button"
            @click="filterByTag(tag)"
            :class="['rounded-lg px-2.5 py-1.5 text-[10px] font-black transition', activeTagFilter === tag ? 'bg-cyan-500 text-white shadow-md' : 'bg-white text-slate-600 hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400']"
          >#{{ tag }}</button>
        </div>
      </div>
    </div>

    <div v-if="users.length > 0 && !selectedUserId" class="mb-4 rounded-xl border border-dashed border-gray-300 bg-gray-50 p-6 text-center shadow-sm transition-colors dark:border-slate-600 dark:bg-slate-800/60">
      <h2 class="text-base font-black text-slate-800 transition-colors dark:text-slate-200">Select a user to manage their bookmarks</h2>
      <p class="mt-1 text-xs text-slate-500 transition-colors dark:text-slate-400">Choose a workspace from the dropdown above.</p>
    </div>

    <div v-if="isLoading" class="flex justify-center py-16">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-cyan-500 border-t-transparent"></div>
      <span class="ml-3 self-center text-sm font-semibold text-slate-600 transition-colors dark:text-slate-400">Loading bookmarks...</span>
    </div>

    <div v-else-if="error" class="mb-4 rounded-xl border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700 shadow-sm transition-colors dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-400">
      {{ error }}
      <button type="button" @click="loadBookmarks" class="ml-2 underline decoration-red-300 underline-offset-4 transition-colors dark:decoration-red-800">Retry</button>
    </div>

    <template v-else-if="selectedUserId">
      <form @submit.prevent="addBookmark" class="mb-4 rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90">
        <div class="flex items-center gap-2">
          <input v-model="newTitle" type="text" placeholder="Title" required class="min-w-0 flex-1 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30" />
          <input v-model="newUrl" type="url" placeholder="URL" required class="min-w-0 flex-[2] rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30" />
          <input v-model="newDescription" type="text" placeholder="Description" class="hidden rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30 lg:block min-w-0 flex-1" />
          <input v-model="newTags" type="text" placeholder="Tags: comma, separated" class="min-w-0 flex-1 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30" />
          <button type="submit" class="shrink-0 rounded-lg bg-gradient-to-r from-cyan-500 to-blue-600 px-4 py-2 text-xs font-black text-white shadow-lg shadow-cyan-500/25 transition hover:-translate-y-0.5 hover:shadow-xl focus:outline-none focus:ring-4 focus:ring-cyan-200 dark:focus:ring-cyan-900/40">
            Add
          </button>
        </div>
      </form>

      <div v-if="activeTagFilter" class="mb-4 flex items-center gap-2 rounded-xl border border-cyan-200 bg-cyan-50/80 p-2 text-xs text-cyan-800 shadow-sm transition-colors dark:border-cyan-900/30 dark:bg-cyan-900/20 dark:text-cyan-300">
        <span class="font-bold">Filtering by tag:</span>
        <span class="rounded-md bg-white px-2 py-0.5 font-black text-cyan-700 shadow-sm transition-colors dark:bg-slate-800 dark:text-cyan-400">{{ activeTagFilter }}</span>
        <button type="button" @click="activeTagFilter = null" class="rounded-md bg-cyan-200 px-2 py-0.5 font-black text-cyan-800 transition hover:bg-cyan-300 dark:bg-cyan-800 dark:text-cyan-200 dark:hover:bg-cyan-700">Clear</button>
      </div>

      <div v-if="bookmarks.length === 0" class="rounded-xl border border-dashed border-gray-300 bg-gray-50 p-8 text-center shadow-sm transition-colors dark:border-slate-600 dark:bg-slate-800/60">
        <div class="mx-auto grid h-10 w-10 place-items-center rounded-xl bg-cyan-50 text-cyan-500 transition-colors dark:bg-cyan-900/20 dark:text-cyan-400">
          <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
          </svg>
        </div>
        <h2 class="mt-3 text-base font-black text-slate-800 transition-colors dark:text-slate-200">No bookmarks yet</h2>
        <p class="mt-1 text-xs text-slate-500 transition-colors dark:text-slate-400">Save your first link above.</p>
      </div>

      <div v-else>
        <div class="hidden overflow-hidden rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90 md:block">
          <table class="min-w-full divide-y divide-gray-200 transition-colors dark:divide-slate-700">
            <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
              <tr>
                <th class="w-10 px-3 py-3"></th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Title</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">URL</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Description</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Tags</th>
                <th class="w-28 px-3 py-3 text-right text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
              <tr v-for="b in bookmarks" :key="b.id" class="transition hover:bg-cyan-50/60 dark:hover:bg-cyan-900/20">
                <td class="px-3 py-3">
                  <div class="grid h-8 w-8 place-items-center rounded-lg bg-gradient-to-br from-slate-900 to-slate-800 text-xs font-black text-white dark:from-slate-950 dark:to-slate-900">{{ getInitial(b.title) }}</div>
                </td>
                <td class="px-3 py-3">
                  <span class="block truncate text-sm font-black text-slate-900 transition-colors dark:text-white" :title="b.title">{{ b.title }}</span>
                  <span class="block truncate text-[10px] text-cyan-600 dark:text-cyan-400">{{ formatHost(b.url) }}</span>
                </td>
                <td class="max-w-xs px-3 py-3">
                  <a :href="b.url" target="_blank" rel="noopener" class="block truncate text-sm font-semibold text-blue-600 transition-colors hover:underline dark:text-blue-400" :title="b.url">{{ b.url }}</a>
                </td>
                <td class="max-w-sm px-3 py-3">
                  <span class="block truncate text-sm text-slate-500 transition-colors dark:text-slate-400" :title="b.description">{{ b.description || '—' }}</span>
                </td>
                <td class="px-3 py-3">
                  <div class="flex flex-wrap gap-1">
                    <button
                      v-for="tag in b.tags"
                      :key="tag"
                      type="button"
                      @click="filterByTag(tag)"
                      class="rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-black text-slate-600 transition hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400"
                    >#{{ tag }}</button>
                    <span v-if="!b.tags.length" class="text-[10px] text-slate-400 dark:text-slate-500">—</span>
                  </div>
                </td>
                <td class="px-3 py-3 text-right">
                  <div class="flex justify-end gap-1">
                    <button type="button" @click="openEdit(b)" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-slate-500 transition hover:bg-cyan-50 hover:text-cyan-700 dark:hover:bg-cyan-900/10 dark:hover:text-cyan-400">Edit</button>
                    <button type="button" @click="removeBookmark(b.id)" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-red-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">Delete</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 md:hidden">
          <article v-for="b in bookmarks" :key="b.id" class="group flex flex-col overflow-hidden rounded-xl border border-gray-200/80 bg-white shadow-sm transition hover:-translate-y-0.5 hover:border-cyan-200 hover:shadow-md dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-cyan-500/30">
            <div class="flex items-start justify-between gap-3 border-b border-gray-100 bg-gradient-to-br from-slate-900 to-slate-800 p-3 text-white transition-colors dark:from-slate-950 dark:to-slate-900">
              <div class="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-white/10 text-sm font-black">{{ getInitial(b.title) }}</div>
              <div class="text-right">
                <h3 class="line-clamp-1 text-sm font-black">{{ b.title }}</h3>
                <a :href="b.url" target="_blank" rel="noopener" class="line-clamp-1 text-[10px] text-cyan-200 hover:text-cyan-50">{{ formatHost(b.url) }}</a>
              </div>
            </div>
            <div class="flex flex-1 flex-col p-3">
              <a :href="b.url" target="_blank" rel="noopener" class="truncate text-xs font-bold text-blue-600 transition-colors hover:underline dark:text-blue-400">{{ b.url }}</a>
              <p v-if="b.description" class="mt-2 line-clamp-2 text-xs leading-5 text-slate-500 transition-colors dark:text-slate-400">{{ b.description }}</p>
              <div v-if="b.tags.length" class="mt-auto flex flex-wrap gap-1 pt-3">
                <button
                  v-for="tag in b.tags"
                  :key="tag"
                  type="button"
                  @click="filterByTag(tag)"
                  class="rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-black text-slate-600 transition hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400"
                >#{{ tag }}</button>
              </div>
              <div class="mt-3 flex gap-2 border-t border-gray-100 pt-3 transition-colors dark:border-slate-700/50">
                <button type="button" @click="openEdit(b)" class="flex-1 rounded-lg bg-gray-100 px-3 py-2 text-xs font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600">Edit</button>
                <button type="button" @click="removeBookmark(b.id)" class="flex-1 rounded-lg bg-red-50 px-3 py-2 text-xs font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
              </div>
            </div>
          </article>
        </div>
      </div>
    </template>

    <div v-else-if="!isLoading && !error" class="rounded-xl border border-dashed border-gray-300 bg-gray-50 p-6 text-center shadow-sm transition-colors dark:border-slate-600 dark:bg-slate-800/60">
      <div class="mx-auto grid h-10 w-10 place-items-center rounded-xl bg-gray-100 text-slate-400 transition-colors dark:bg-slate-700 dark:text-slate-500">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </div>
      <h2 class="mt-3 text-base font-black text-slate-800 transition-colors dark:text-slate-200">Choose a workspace</h2>
      <p class="mt-1 text-xs text-slate-500 transition-colors dark:text-slate-400">Select a user to manage their bookmarks.</p>
    </div>

    <div v-if="editing" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-4 backdrop-blur-sm">
      <div class="w-full max-w-lg rounded-xl border border-gray-200 bg-white p-5 shadow-2xl shadow-gray-900/30 transition-colors dark:border-white/10 dark:bg-slate-800 dark:shadow-slate-950/30 sm:p-6">
        <div class="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 class="text-lg font-black text-slate-900 transition-colors dark:text-white">Edit bookmark</h2>
            <p class="mt-1 text-xs text-slate-500 transition-colors dark:text-slate-400">Update the saved link details.</p>
          </div>
          <button type="button" @click="editing = null" class="grid h-8 w-8 place-items-center rounded-lg bg-gray-100 text-slate-500 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-400 dark:hover:bg-slate-600" aria-label="Close">×</button>
        </div>
        <div class="grid gap-3">
          <input v-model="editForm.title" type="text" placeholder="Title" required class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
          <input v-model="editForm.url" type="url" placeholder="URL" required class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
          <input v-model="editForm.description" type="text" placeholder="Description" class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
          <input v-model="editForm.tagsStr" type="text" placeholder="Tags: comma, separated" class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
        </div>
        <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button type="button" @click="editing = null" class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600">Cancel</button>
          <button type="button" @click="saveEdit" class="rounded-lg bg-gradient-to-r from-cyan-500 to-blue-600 px-4 py-2 text-sm font-black text-white shadow-lg shadow-cyan-500/20">Save changes</button>
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
