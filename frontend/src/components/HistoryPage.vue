<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Browsing log" title="History" color="violet">
      <template #stats>
        <StatCard :value="historyEntries.length" label="Entries" variant="dark" color="violet" />
        <StatCard :value="totalDuration" label="Duration" variant="primary" color="violet" />
      </template>
      <template #actions>
        <UserSelector id="history-user" v-model="selectedUserId" :users="users" color="violet" />
      </template>
    </PageHeader>

    <SelectUserPrompt title="Select a user to view their browsing history" :users-count="users.length" :selected-user-id="selectedUserId" />

    <LoadingSpinner v-if="isLoading" message="Loading history..." color="violet" />

    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadHistory" />

    <div v-else-if="selectedUserId">
      <form @submit.prevent="addEntry" class="mb-4 rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90">
        <div class="flex items-center gap-2">
          <InputField v-model="newUrl" type="url" placeholder="URL" required flex color="violet" />
          <InputField v-model="newTitle" type="text" placeholder="Title" required flex color="violet" />
          <InputField v-model="newDuration" type="number" placeholder="Seconds" class="w-24" color="violet" />
          <Button type="submit" variant="gradient-violet" size="sm">Add</Button>
        </div>
      </form>

      <div class="mb-4">
        <InputField v-model="urlFilter" placeholder="Search by URL or title..." color="violet" />
      </div>

      <EmptyState
        v-if="filteredHistory.length === 0"
        :title="historyEntries.length === 0 ? 'No history yet' : 'No matching entries'"
        :description="historyEntries.length === 0 ? 'Add a browsing entry above.' : 'Try a different search.'"
        icon="clock"
        color="violet"
      />

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
              <HistoryTableRow
                v-for="h in filteredHistory"
                :key="h.id"
                :entry="h"
                @delete="removeEntry"
              />
            </tbody>
          </table>
        </div>

        <div class="relative space-y-2 before:absolute before:left-5 before:top-3 before:h-[calc(100%-1.5rem)] before:w-px before:bg-gray-200 dark:before:bg-slate-700 md:hidden">
          <HistoryCard
            v-for="h in filteredHistory"
            :key="h.id"
            :entry="h"
            @delete="removeEntry"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getHistory, createHistory, deleteHistory } from '../lib/api'
import { formatDuration } from '../lib/utils'
import { useUser } from '../composables/useUser'
import type { History } from '../types'
import PageHeader from './ui/PageHeader.vue'
import StatCard from './ui/StatCard.vue'
import UserSelector from './ui/UserSelector.vue'
import LoadingSpinner from './ui/LoadingSpinner.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import EmptyState from './ui/EmptyState.vue'
import InputField from './ui/InputField.vue'
import Button from './ui/Button.vue'
import SelectUserPrompt from './ui/SelectUserPrompt.vue'
import HistoryTableRow from './history/HistoryTableRow.vue'
import HistoryCard from './history/HistoryCard.vue'

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
