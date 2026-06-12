<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Password vault" title="Wallet" color="emerald">
      <template #stats>
        <StatCard :value="walletEntries.length" label="Entries" variant="dark" color="emerald" />
        <StatCard :value="filteredEntries.length" label="Visible" variant="primary" color="emerald" />
      </template>
      <template #actions>
        <UserSelector id="wallet-user" v-model="selectedUserId" :users="users" color="emerald" />
      </template>
    </PageHeader>

    <SelectUserPrompt title="Select a user to manage their wallet" :users-count="users.length" :selected-user-id="selectedUserId" />

    <LoadingSpinner v-if="isLoading" message="Loading wallet..." color="emerald" />

    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadWallet" />

    <div v-else-if="selectedUserId">
      <form @submit.prevent="addEntry" class="mb-4 rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90">
        <div class="flex items-center gap-2">
          <InputField v-model="newWebsite" type="text" placeholder="Website" required flex color="emerald" />
          <InputField v-model="newUsername" type="text" placeholder="Username" required flex color="emerald" />
          <InputField v-model="newPassword" type="password" placeholder="Password" required flex color="emerald" />
          <InputField v-model="newDescription" type="text" placeholder="Description" class="hidden lg:block" flex color="emerald" />
          <Button type="submit" variant="gradient-emerald" size="sm">Add</Button>
        </div>
      </form>

      <div class="mb-4">
        <InputField v-model="websiteFilter" placeholder="Search website, username, or description..." color="emerald" />
      </div>

      <EmptyState
        v-if="filteredEntries.length === 0"
        :title="walletEntries.length === 0 ? 'No saved credentials' : 'No matching entries'"
        :description="walletEntries.length === 0 ? 'Add your first entry above.' : 'Try a different search.'"
        icon="lock"
        color="emerald"
      />

      <div v-else>
        <div class="hidden overflow-hidden rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90 md:block">
          <table class="min-w-full divide-y divide-gray-200 transition-colors dark:divide-slate-700">
            <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
              <tr>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Website</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Username</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Password</th>
                <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Description</th>
                <th class="w-28 px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Updated</th>
                <th class="w-24 px-3 py-3 text-right text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
              <WalletTableRow
                v-for="e in filteredEntries"
                :key="e.id"
                :entry="e"
                @edit="openEdit"
                @delete="removeEntry"
              />
            </tbody>
          </table>
        </div>

        <div class="grid gap-3 md:hidden">
          <WalletCard
            v-for="e in filteredEntries"
            :key="e.id"
            :entry="e"
            @edit="openEdit"
            @delete="removeEntry"
          />
        </div>
      </div>
    </div>

    <Modal :open="!!editing" title="Edit wallet entry" description="Update saved credentials." @close="editing = null">
      <div v-if="editing" class="grid gap-3">
        <input v-model="editForm.website" type="text" placeholder="Website" required class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-emerald-900/30" />
        <input v-model="editForm.username" type="text" placeholder="Username" required class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-emerald-900/30" />
        <input v-model="editForm.password" type="text" placeholder="Password" required class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-emerald-900/30" />
        <input v-model="editForm.description" type="text" placeholder="Description" class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-emerald-900/30" />
      </div>
      <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <button type="button" @click="editing = null" class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600">Cancel</button>
        <button type="button" @click="saveEdit" class="rounded-lg bg-gradient-to-r from-emerald-500 to-teal-600 px-4 py-2 text-sm font-black text-white shadow-lg shadow-emerald-500/20">Save changes</button>
      </div>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useUser } from '../composables/useUser'
import { useWallet } from '../composables/useWallet'
import PageHeader from './ui/PageHeader.vue'
import StatCard from './ui/StatCard.vue'
import UserSelector from './ui/UserSelector.vue'
import LoadingSpinner from './ui/LoadingSpinner.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import EmptyState from './ui/EmptyState.vue'
import InputField from './ui/InputField.vue'
import Button from './ui/Button.vue'
import SelectUserPrompt from './ui/SelectUserPrompt.vue'
import Modal from './ui/Modal.vue'
import WalletTableRow from './wallet/WalletTableRow.vue'
import WalletCard from './wallet/WalletCard.vue'

const { users, currentUserId, setUser, clearUser } = useUser()

const selectedUserId = ref<number | null>(currentUserId.value)

const {
  walletEntries,
  isLoading,
  error,
  websiteFilter,
  newWebsite,
  newUsername,
  newPassword,
  newDescription,
  editing,
  editForm,
  filteredEntries,
  loadWallet,
  addEntry,
  openEdit,
  saveEdit,
  removeEntry,
} = useWallet(selectedUserId)

watch(selectedUserId, (id) => {
  if (id) {
    setUser(id)
    loadWallet()
  } else {
    clearUser()
    walletEntries.value = []
    websiteFilter.value = ''
  }
})

if (selectedUserId.value) {
  setUser(selectedUserId.value)
  loadWallet()
}
</script>
