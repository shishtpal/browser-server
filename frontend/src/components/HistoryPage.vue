<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <div class="mb-4 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex items-center gap-3">
        <div>
          <p class="mb-1 inline-flex rounded-full bg-violet-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.2em] text-violet-700 transition-colors dark:bg-violet-900/20 dark:text-violet-400">Browsing log</p>
          <h1 class="text-2xl font-black tracking-tight text-slate-900 transition-colors dark:text-white sm:text-3xl">History</h1>
        </div>
        <div class="flex items-center gap-2">
          <div class="rounded-xl bg-slate-900 px-3 py-2 text-center text-white shadow-lg shadow-violet-500/15 transition-colors dark:bg-slate-950">
            <div class="text-sm font-black leading-none">{{ historyEntries.length }}</div>
            <div class="text-[10px] font-semibold text-slate-300 leading-none mt-0.5">Entries</div>
          </div>
          <div class="rounded-xl bg-violet-600 px-3 py-2 text-center text-white shadow-lg shadow-violet-500/20">
            <div class="text-sm font-black leading-none">{{ totalDuration }}</div>
            <div class="text-[10px] font-semibold text-violet-100 leading-none mt-0.5">Duration</div>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <select
          id="history-user"
          v-model="selectedUserId"
          class="rounded-xl border border-gray-300 bg-gray-50 px-3 py-2 text-xs font-semibold text-slate-700 shadow-sm transition focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-violet-900/30"
        >
          <option :value="null">All users</option>
          <option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }}</option>
        </select>
      </div>
    </div>

    <div v-if="users.length > 0 && !selectedUserId" class="mb-4 rounded-xl border border-dashed border-gray-300 bg-gray-50 p-6 text-center shadow-sm transition-colors dark:border-slate-600 dark:bg-slate-800/60">
      <h2 class="text-base font-black text-slate-800 transition-colors dark:text-slate-200">Select a user to view their browsing history</h2>
      <p class="mt-1 text-xs text-slate-500 transition-colors dark:text-slate-400">Choose a workspace from the dropdown above.</p>
    </div>

    <div v-if="isLoading" class="flex justify-center py-16">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-violet-500 border-t-transparent"></div>
      <span class="ml-3 self-center text-sm font-semibold text-slate-600 transition-colors dark:text-slate-400">Loading history...</span>
    </div>

    <div v-else-if="error" class="mb-4 rounded-xl border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700 shadow-sm transition-colors dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-400">
      {{ error }}
      <button type="button" @click="loadHistory" class="ml-2 underline decoration-red-300 underline-offset-4 transition-colors dark:decoration-red-800">Retry</button>
    </div>

    <template v-else-if="selectedUserId">
      <form @submit.prevent="addEntry" class="mb-4 rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90">
        <div class="flex items-center gap-2">
          <input v-model="newUrl" type="url" placeholder="URL" required class="min-w-0 flex-[2] rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30" />
          <input v-model="newTitle" type="text" placeholder="Title" required class="min-w-0 flex-1 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30" />
          <input v-model="newDuration" type="number" min="0" placeholder="Seconds" class="w-24 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30" />
          <button type="submit" class="shrink-0 rounded-lg bg-gradient-to-r from-violet-600 to-fuchsia-600 px-4 py-2 text-xs font-black text-white shadow-lg shadow-violet-500/25 transition hover:-translate-y-0.5 hover:shadow-xl focus:outline-none focus:ring-4 focus:ring-violet-200 dark:focus:ring-violet-900/40">
            Add
          </button>
        </div>
      </form>

      <div class="mb-4">
        <input
          v-model="urlFilter"
          type="text"
          placeholder="Search by URL or title..."
          class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30"
        />
      </div>

      <div v-if="filteredHistory.length === 0" class="rounded-xl border border-dashed border-gray-300 bg-gray-50 p-8 text-center shadow-sm transition-colors dark:border-slate-600 dark:bg-slate-800/60">
        <div class="mx-auto grid h-10 w-10 place-items-center rounded-xl bg-violet-50 text-violet-500 transition-colors dark:bg-violet-900/20 dark:text-violet-400">
          <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <h2 class="mt-3 text-base font-black text-slate-800 transition-colors dark:text-slate-200">{{ historyEntries.length === 0 ? 'No history yet' : 'No matching entries' }}</h2>
        <p class="mt-1 text-xs text-slate-500 transition-colors dark:text-slate-400">{{ historyEntries.length === 0 ? 'Add a browsing entry above.' : 'Try a different search.' }}</p>
      </div>

      <div v-else>
        <div class="hidden overflow-hidden rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90 md:block">
          <table class="min-w-full divide-y divide-gray-200 transition-colors dark:divide-slate-700">
            <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
              <tr>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Title</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">URL</th>
                <th class="w-28 px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Visited</th>
                <th class="w-20 px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Duration</th>
                <th class="w-20 px-3 py-3 text-right text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
              <tr v-for="h in filteredHistory" :key="h.id" class="transition hover:bg-violet-50/60 dark:hover:bg-violet-900/20">
                <td class="px-3 py-3">
                  <div class="flex items-center gap-3">
                    <div class="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-violet-50 text-violet-600 dark:bg-violet-900/20 dark:text-violet-400">
                      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064" />
                      </svg>
                    </div>
                    <span class="block truncate text-sm font-black text-slate-900 transition-colors dark:text-white" :title="h.title">{{ h.title }}</span>
                  </div>
                </td>
                <td class="max-w-md px-3 py-3">
                  <a :href="h.url" target="_blank" rel="noopener" class="block truncate text-sm font-semibold text-blue-600 transition-colors hover:underline dark:text-blue-400" :title="h.url">{{ h.url }}</a>
                </td>
                <td class="px-3 py-3">
                  <span class="whitespace-nowrap rounded-md bg-gray-100 px-2 py-1 text-[10px] font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400">{{ formatDate(h.visited_at) }}</span>
                </td>
                <td class="px-3 py-3">
                  <span v-if="h.duration > 0" class="whitespace-nowrap rounded-md bg-violet-50 px-2 py-1 text-[10px] font-bold text-violet-700 transition-colors dark:bg-violet-900/20 dark:text-violet-400">{{ formatDuration(h.duration) }}</span>
                  <span v-else class="text-[10px] text-slate-400 dark:text-slate-500">—</span>
                </td>
                <td class="px-3 py-3 text-right">
                  <button type="button" @click="removeEntry(h.id)" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-red-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">Delete</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="relative space-y-2 before:absolute before:left-5 before:top-3 before:h-[calc(100%-1.5rem)] before:w-px before:bg-gray-200 dark:before:bg-slate-700 md:hidden">
          <article v-for="h in filteredHistory" :key="h.id" class="relative rounded-xl border border-gray-200/80 bg-white p-3 shadow-sm transition hover:-translate-y-0.5 hover:border-violet-200 hover:shadow-md dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-violet-500/30">
            <div class="absolute left-3 top-4 h-3 w-3 rounded-full border-3 border-white bg-violet-500 shadow-sm dark:border-slate-800"></div>
            <div class="pl-7">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <span class="block truncate text-sm font-black text-slate-900 transition-colors dark:text-white">{{ h.title }}</span>
                  <span class="mt-0.5 block truncate text-xs font-semibold text-blue-600 transition-colors hover:underline dark:text-blue-400">{{ h.url }}</span>
                  <div class="mt-2 flex flex-wrap gap-2">
                    <span class="rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400">{{ formatDate(h.visited_at) }}</span>
                    <span v-if="h.duration > 0" class="rounded-md bg-violet-50 px-2 py-0.5 text-[10px] font-bold text-violet-700 transition-colors dark:bg-violet-900/20 dark:text-violet-400">{{ formatDuration(h.duration) }}</span>
                  </div>
                </div>
                <button type="button" @click="removeEntry(h.id)" class="shrink-0 rounded-lg bg-red-50 px-3 py-1.5 text-xs font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
              </div>
            </div>
          </article>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getHistory, createHistory, deleteHistory } from '../lib/api'
import { useUser } from '../composables/useUser'
import type { History } from '../types'

const { users, currentUserId, setUser, clearUser } = useUser()

const selectedUserId = ref<number | null>(currentUserId.value)
const historyEntries = ref<History[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const urlFilter = ref('')

const newUrl = ref('')
const newTitle = ref('')
const newDuration = ref('')

const totalDuration = computed(() => formatDuration(historyEntries.value.reduce((sum, h) => sum + h.duration, 0)))

const filteredHistory = computed(() => {
  if (!urlFilter.value.trim()) return historyEntries.value
  const q = urlFilter.value.toLowerCase()
  return historyEntries.value.filter(h => h.url.toLowerCase().includes(q) || h.title.toLowerCase().includes(q))
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
  if (!sec) return '0s'
  if (sec < 60) return `${sec}s`
  const m = Math.floor(sec / 60)
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  return `${h}h ${m % 60}m`
}

watch(selectedUserId, (id) => {
  if (id) {
    setUser(id)
    loadHistory()
  } else {
    clearUser()
    historyEntries.value = []
    urlFilter.value = ''
  }
})

if (selectedUserId.value) {
  setUser(selectedUserId.value)
  loadHistory()
}
</script>
