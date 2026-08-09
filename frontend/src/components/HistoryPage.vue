<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Browsing log" title="History" color="violet">
      <template #stats>
        <StatCard :value="historyEntries.length" label="Entries" variant="dark" color="violet" />
        <StatCard :value="totalDuration" label="Duration" variant="primary" color="violet" />
      </template>
      <template #controls>
        <UserSelector id="history-user" v-model="selectedUserId" :users="users" color="violet" />
      </template>
    </PageHeader>

    <SelectUserPrompt
      title="Select a user to view their browsing history"
      :users-count="users.length"
      :selected-user-id="selectedUserId"
    />

    <LoadingSpinner v-if="isLoading" message="Loading history..." color="violet" />
    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadHistory" />

    <div v-else-if="selectedUserId">
      <HistoryImport :selected-user-id="selectedUserId" class="mb-4" @imported="loadHistory" />

      <HistoryAddForm @submit="handleAdd" />

      <HistorySearchBar
        v-model="urlFilter"
        :filtered-count="filteredHistory.length"
        :total-count="historyEntries.length"
      />

      <EmptyState
        v-if="filteredHistory.length === 0"
        :title="emptyTitle"
        :description="
          historyEntries.length === 0 ? 'Add a browsing entry above.' : 'Try a different search.'
        "
        icon="clock"
        color="violet"
      />

      <template v-else>
        <!-- Desktop table -->
        <div
          class="hidden overflow-x-auto rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors md:block dark:border-slate-700/80 dark:bg-slate-800/90"
        >
          <table
            class="w-full table-fixed divide-y divide-gray-200 transition-colors dark:divide-slate-700"
          >
            <colgroup>
              <col class="w-[25%]" />
              <col class="w-[35%]" />
              <col class="w-[22%]" />
              <col class="w-[10%]" />
              <col class="w-[8%]" />
            </colgroup>
            <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
              <tr>
                <th
                  v-for="column in columns"
                  :key="column.label"
                  class="px-3 py-3 text-left text-[10px] font-black tracking-wide text-slate-500 uppercase transition-colors dark:text-slate-400"
                  :class="{ 'text-right': column.align === 'right' }"
                >
                  {{ column.label }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
              <HistoryTableRow
                v-for="entry in filteredHistory"
                :key="entry.id"
                :entry="entry"
                @delete="confirmDelete"
              />
            </tbody>
          </table>
        </div>

        <!-- Mobile timeline cards -->
        <div
          class="relative space-y-2 before:absolute before:top-3 before:left-5 before:h-[calc(100%-1.5rem)] before:w-px before:bg-gray-200 md:hidden dark:before:bg-slate-700"
        >
          <HistoryCard
            v-for="entry in filteredHistory"
            :key="entry.id"
            :entry="entry"
            @delete="confirmDelete"
          />
        </div>

        <!-- Scroll sentinel + loading-more indicator -->
        <div ref="scrollSentinel" class="flex items-center justify-center py-6">
          <div
            v-if="isLoadingMore"
            class="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400"
            role="status"
          >
            <LoaderCircle class="h-4 w-4 animate-spin" aria-hidden="true" />
            Loading more…
          </div>
          <span
            v-else-if="!hasMore && filteredHistory.length > 0"
            class="text-xs text-slate-400 dark:text-slate-500"
          >
            All {{ historyEntries.length }} entries loaded
          </span>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { LoaderCircle } from '@lucide/vue';
import { useUser } from '../composables/useUser';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';
import UserSelector from './ui/UserSelector.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import EmptyState from './ui/EmptyState.vue';
import SelectUserPrompt from './ui/SelectUserPrompt.vue';
import { useHistoryPage, type HistoryCreateInput } from './history/composables/useHistoryPage';
import HistoryCard from './history/HistoryCard.vue';
import HistoryImport from './history/HistoryImport.vue';
import HistoryAddForm from './history/HistoryAddForm.vue';
import HistoryTableRow from './history/HistoryTableRow.vue';
import HistorySearchBar from './history/HistorySearchBar.vue';

const { users, currentUserId, setUser, clearUser } = useUser();
const selectedUserId = ref<number | null>(currentUserId.value);

const {
  historyApi: {
    historyEntries,
    isLoading,
    isLoadingMore,
    error,
    urlFilter,
    hasMore,
    totalDuration,
    filteredHistory,
    loadHistory,
    addEntry,
  },
  scrollSentinel,
  confirmDelete,
} = useHistoryPage(selectedUserId);

watch(selectedUserId, (id) => {
  if (id) setUser(id);
  else clearUser();
});

// Data load + observer wiring happen inside useHistoryPage / useHistory.

const columns: { label: string; align?: 'right' }[] = [
  { label: 'Title' },
  { label: 'URL' },
  { label: 'Visited' },
  { label: 'Duration' },
  { label: 'Actions', align: 'right' },
];

const emptyTitle = computed(() =>
  historyEntries.value.length === 0 ? 'No history yet' : 'No matching entries',
);

async function handleAdd(input: HistoryCreateInput) {
  await addEntry(input);
}
</script>
