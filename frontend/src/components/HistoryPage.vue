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
      <form @submit.prevent="addEntry" class="mb-4 flex flex-col gap-2 rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors sm:flex-row sm:items-center sm:gap-3 dark:border-white/10 dark:bg-slate-800/90">
        <InputField v-model="newUrl" type="url" placeholder="https://example.com" required flex color="violet" />
        <InputField v-model="newTitle" type="text" placeholder="Page title" required flex color="violet" />
        <InputField v-model="newDuration" type="number" placeholder="Duration (s)" class="w-28 shrink-0 sm:w-24" color="violet" />
        <Button type="submit" variant="gradient-violet" size="sm">Add</Button>
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
import { ref, watch } from 'vue'
import { useUser } from '../composables/useUser'
import { useHistory } from '../composables/useHistory'
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

const {
  historyEntries,
  isLoading,
  error,
  urlFilter,
  newUrl,
  newTitle,
  newDuration,
  totalDuration,
  filteredHistory,
  loadHistory,
  addEntry,
  removeEntry,
} = useHistory(selectedUserId)

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
