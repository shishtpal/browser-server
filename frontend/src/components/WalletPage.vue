<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Password vault" title="Wallet" color="emerald">
      <template #stats>
        <StatCard :value="walletEntries.length" label="Entries" variant="dark" color="emerald" />
        <StatCard
          :value="filteredEntries.length"
          label="Visible"
          variant="primary"
          color="emerald"
        />
      </template>
      <template #controls>
        <UserSelector id="wallet-user" v-model="selectedUserId" :users="users" color="emerald" />
      </template>
    </PageHeader>

    <SelectUserPrompt
      title="Select a user to manage their wallet"
      :users-count="users.length"
      :selected-user-id="selectedUserId"
    />

    <LoadingSpinner v-if="isLoading" message="Loading wallet..." color="emerald" />
    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadWallet" />

    <div v-else-if="selectedUserId">
      <div>
        <WalletAddForm @submit="addEntry" />
      </div>

      <WalletImport :selected-user-id="selectedUserId" class="mb-4" @imported="loadWallet" />

      <WalletSearchBar
        v-model:search-query="searchQuery"
        v-model:search-column="searchColumn"
        :placeholder="searchPlaceholder"
        :filtered-count="filteredEntries.length"
        :total-count="walletEntries.length"
      />

      <EmptyState
        v-if="filteredEntries.length === 0"
        :title="walletEntries.length === 0 ? 'No saved credentials' : 'No matching entries'"
        :description="
          walletEntries.length === 0 ? 'Add your first entry above.' : 'Try a different search.'
        "
        icon="lock"
        color="emerald"
      />

      <template v-else>
        <!-- Desktop table -->
        <div
          class="hidden overflow-hidden rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors md:block dark:border-slate-700/80 dark:bg-slate-800/90"
        >
          <table
            class="min-w-full divide-y divide-gray-200 transition-colors dark:divide-slate-700"
          >
            <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
              <tr>
                <th
                  v-for="column in columns"
                  :key="column.label"
                  class="px-3 py-3 text-left text-[10px] font-black tracking-wide text-slate-500 uppercase transition-colors dark:text-slate-400"
                  :class="[column.align === 'right' ? 'text-right' : '', column.width ?? '']"
                >
                  {{ column.label }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
              <WalletTableRow
                v-for="entry in filteredEntries"
                :key="entry.id"
                :entry="entry"
                :user-id="selectedUserId"
                @edit="openEdit"
                @delete="confirmDelete"
              />
            </tbody>
          </table>
        </div>

        <!-- Mobile cards -->
        <div class="grid gap-3 sm:grid-cols-2 md:hidden">
          <WalletCard
            v-for="entry in filteredEntries"
            :key="entry.id"
            :entry="entry"
            :user-id="selectedUserId"
            @edit="openEdit"
            @delete="confirmDelete"
          />
        </div>
      </template>
    </div>

    <WalletEditModal
      :entry="editing"
      :revealed-password="editingPassword"
      :is-revealing="isRevealingPassword"
      @close="closeEdit"
      @save="saveEdit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { Plus } from '@lucide/vue';
import { useUser } from '../composables/useUser';
import { useWalletPage } from './wallet/composables/useWalletPage';
import Button from './ui/Button.vue';
import UserSelector from './ui/UserSelector.vue';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import EmptyState from './ui/EmptyState.vue';
import SelectUserPrompt from './ui/SelectUserPrompt.vue';
import WalletAddForm from './wallet/WalletAddForm.vue';
import WalletImport from './wallet/WalletImport.vue';
import WalletSearchBar from './wallet/WalletSearchBar.vue';
import WalletTableRow from './wallet/views/WalletTableRow.vue';
import WalletCard from './wallet/views/WalletCard.vue';
import WalletEditModal from './wallet/WalletEditModal.vue';

const { users, currentUserId, setUser, clearUser } = useUser();
const selectedUserId = ref<number | null>(currentUserId.value);

const {
  walletApi: {
    walletEntries,
    isLoading,
    error,
    searchQuery,
    searchColumn,
    filteredEntries,
    loadWallet,
    addEntry,
  },
  editing,
  editingPassword,
  isRevealingPassword,
  openEdit,
  closeEdit,
  saveEdit,
  confirmDelete,
  searchPlaceholder,
} = useWalletPage(selectedUserId);

watch(selectedUserId, (id) => {
  if (id) setUser(id);
  else clearUser();
});

// Data loads automatically via the immediate watcher inside useWallet.

const columns: { label: string; align?: 'right'; width?: string }[] = [
  { label: 'Website' },
  { label: 'Provider' },
  { label: 'Username' },
  { label: 'Password' },
  { label: 'Description' },
  { label: 'Updated', width: 'w-28' },
  { label: 'Actions', align: 'right', width: 'w-24' },
];
</script>
