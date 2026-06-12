<template>
  <div class="mx-auto max-w-6xl px-4 py-6 sm:px-6 lg:px-8">
    <section class="mb-6 overflow-hidden rounded-[2rem] border border-gray-200/80 bg-white/90 p-5 shadow-2xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-8">
      <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p class="mb-2 inline-flex rounded-full bg-violet-50 px-3 py-1 text-xs font-bold uppercase tracking-[0.2em] text-violet-700 transition-colors dark:bg-violet-900/20 dark:text-violet-400">Browsing log</p>
          <h1 class="text-3xl font-black tracking-tight text-slate-900 transition-colors dark:text-white sm:text-5xl">History</h1>
          <p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600 transition-colors dark:text-slate-400 sm:text-base">Review visited pages with a clean timeline and quick search.</p>
        </div>
        <div class="grid grid-cols-2 gap-3 sm:max-w-sm">
          <div class="rounded-3xl bg-slate-900 p-4 text-center text-white shadow-lg shadow-violet-500/15 transition-colors dark:bg-slate-950">
            <div class="text-2xl font-black">{{ historyEntries.length }}</div>
            <div class="text-xs font-semibold text-slate-300">Entries</div>
          </div>
          <div class="rounded-3xl bg-violet-600 p-4 text-center text-white shadow-lg shadow-violet-500/20">
            <div class="text-2xl font-black">{{ totalDuration }}</div>
            <div class="text-xs font-semibold text-violet-100">Time</div>
          </div>
        </div>
      </div>
    </section>

    <section class="mb-6 rounded-3xl border border-gray-200 bg-white/90 p-4 shadow-xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-5">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div class="min-w-0 flex-1">
          <label class="text-sm font-bold text-slate-700 transition-colors dark:text-slate-300" for="history-user">Select user</label>
          <select
            id="history-user"
            v-model="selectedUserId"
            class="mt-2 w-full rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30"
          >
            <option :value="null">Choose a user</option>
            <option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }}</option>
          </select>
          <p v-if="users.length === 0" class="mt-2 text-sm text-slate-500 transition-colors dark:text-slate-400">No users yet. <a href="/users" class="font-bold text-violet-700 transition-colors hover:text-violet-800 dark:text-violet-400 dark:hover:text-violet-300">Create one</a> to start.</p>
        </div>
      </div>
    </section>

    <div v-if="isLoading" class="flex justify-center py-16">
      <div class="h-10 w-10 animate-spin rounded-full border-4 border-violet-500 border-t-transparent"></div>
      <span class="ml-3 self-center font-semibold text-slate-600 transition-colors dark:text-slate-400">Loading history...</span>
    </div>

    <div v-else-if="error" class="mb-6 rounded-3xl border border-red-200 bg-red-50 p-4 text-sm font-semibold text-red-700 shadow-sm transition-colors dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-400">
      {{ error }}
      <button type="button" @click="loadHistory" class="ml-2 underline decoration-red-300 underline-offset-4 transition-colors dark:decoration-red-800">Retry</button>
    </div>

    <template v-else-if="selectedUserId">
      <form @submit.prevent="addEntry" class="mb-6 rounded-3xl border border-gray-200 bg-white p-4 shadow-xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-5">
        <div class="grid gap-3 md:grid-cols-[1fr_1fr_10rem_auto]">
          <input v-model="newUrl" type="url" placeholder="URL (https://...)" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30" />
          <input v-model="newTitle" type="text" placeholder="Title" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30" />
          <input v-model="newDuration" type="number" min="0" placeholder="Seconds" class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30" />
          <button type="submit" class="rounded-2xl bg-gradient-to-r from-violet-600 to-fuchsia-600 px-5 py-3 text-sm font-black text-white shadow-lg shadow-violet-500/25 transition hover:-translate-y-0.5 hover:shadow-xl hover:shadow-violet-500/30 focus:outline-none focus:ring-4 focus:ring-violet-200 dark:focus:ring-violet-900/40">
            Add
          </button>
        </div>
      </form>

      <div class="mb-5">
        <input
          v-model="urlFilter"
          type="text"
          placeholder="Search visited URLs..."
          class="w-full rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm backdrop-blur-xl transition placeholder:text-slate-400 focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30"
        />
      </div>

      <div v-if="filteredHistory.length === 0" class="rounded-[2rem] border border-dashed border-gray-300 bg-gray-50 p-10 text-center shadow-sm backdrop-blur-xl transition-colors dark:border-slate-600 dark:bg-slate-800/60">
        <div class="mx-auto grid h-14 w-14 place-items-center rounded-3xl bg-violet-50 text-violet-500 transition-colors dark:bg-violet-900/20 dark:text-violet-400">
          <svg class="h-7 w-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <h2 class="mt-4 text-lg font-black text-slate-800 transition-colors dark:text-slate-200">{{ historyEntries.length === 0 ? 'No history yet' : 'No matching entries' }}</h2>
        <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">{{ historyEntries.length === 0 ? 'Add a browsing entry above.' : 'Try a different URL search.' }}</p>
      </div>

      <div v-else class="relative space-y-4 before:absolute before:left-6 before:top-3 before:h-[calc(100%-1.5rem)] before:w-px before:bg-gray-200 dark:before:bg-slate-700 sm:before:left-7">
        <article v-for="h in filteredHistory" :key="h.id" class="relative rounded-[1.75rem] border border-gray-200/80 bg-white p-4 shadow-sm transition hover:-translate-y-0.5 hover:border-violet-200 hover:shadow-xl hover:shadow-gray-900/10 dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-violet-500/30 dark:hover:shadow-slate-950/20 sm:p-5">
          <div class="absolute left-4 top-6 h-3.5 w-3.5 rounded-full border-4 border-white bg-violet-500 shadow-sm dark:border-slate-800 sm:-left-[22px] sm:top-7"></div>
          <div class="pl-8 sm:pl-0">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0">
                <a :href="h.url" target="_blank" rel="noopener" class="group flex items-start gap-3">
                  <div class="grid h-10 w-10 shrink-0 place-items-center rounded-2xl bg-violet-50 text-violet-600 transition group-hover:bg-violet-100 dark:bg-violet-900/20 dark:text-violet-400 dark:group-hover:bg-violet-900/30">
                    <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064" />
                    </svg>
                  </div>
                  <span class="min-w-0">
                    <span class="block truncate text-base font-black text-slate-900 transition-colors sm:text-lg dark:text-white">{{ h.title }}</span>
                    <span class="mt-1 block break-words text-sm font-semibold text-blue-600 transition-colors hover:underline dark:text-blue-400">{{ h.url }}</span>
                  </span>
                </a>
                <div class="mt-3 flex flex-wrap gap-2 pl-[3.25rem] sm:pl-0">
                  <span class="rounded-full bg-gray-100 px-3 py-1 text-xs font-black text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400">{{ formatDate(h.visited_at) }}</span>
                  <span v-if="h.duration > 0" class="rounded-full bg-violet-50 px-3 py-1 text-xs font-black text-violet-700 transition-colors dark:bg-violet-900/20 dark:text-violet-400">{{ formatDuration(h.duration) }}</span>
                </div>
              </div>
              <button type="button" @click="removeEntry(h.id)" class="self-start rounded-2xl bg-red-50 px-4 py-2 text-sm font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30 sm:self-center">Delete</button>
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
      <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">Select a user to view their browsing history.</p>
    </div>
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
