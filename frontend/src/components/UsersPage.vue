<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Workspace" title="Users" color="amber">
      <template #stats>
        <div class="grid h-12 w-12 place-items-center rounded-xl bg-gradient-to-br from-amber-400 to-orange-500 text-lg font-black text-white shadow-xl shadow-orange-500/25">
          {{ usersList.length }}
        </div>
      </template>
    </PageHeader>

    <form @submit.prevent="addUser" class="mb-4 rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90">
      <div class="flex items-center gap-2">
        <InputField v-model="newUsername" type="text" placeholder="Username" required flex color="amber" />
        <InputField v-model="newEmail" type="email" placeholder="Email (optional)" flex color="amber" />
        <Button type="submit" variant="gradient-amber" size="sm">Create</Button>
      </div>
      <div v-if="successMsg" class="mt-3 rounded-lg bg-emerald-50 px-3 py-2 text-xs font-bold text-emerald-700 transition-colors dark:bg-emerald-900/20 dark:text-emerald-400">{{ successMsg }}</div>
    </form>

    <LoadingSpinner v-if="isLoading" message="Loading users..." color="amber" />

    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadUsers" />

    <EmptyState
      v-else-if="usersList.length === 0"
      title="No users yet"
      description="Create your first workspace above."
      icon="users"
      color="amber"
    />

    <div v-else>
      <div class="hidden overflow-hidden rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90 md:block">
        <table class="min-w-full divide-y divide-gray-200 transition-colors dark:divide-slate-700">
          <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
            <tr>
              <th class="w-16 px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">ID</th>
              <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Username</th>
              <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Email</th>
              <th class="w-20 px-3 py-3 text-right text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
            <tr v-for="u in usersList" :key="u.id" class="transition hover:bg-amber-50/60 dark:hover:bg-amber-900/20">
              <td class="px-3 py-3 text-sm font-mono font-bold text-slate-400 transition-colors dark:text-slate-500">#{{ u.id }}</td>
              <td class="px-3 py-3 text-sm font-black text-slate-900 transition-colors dark:text-white">{{ u.username }}</td>
              <td class="px-3 py-3 text-sm font-semibold text-slate-600 transition-colors dark:text-slate-400">{{ u.email || '—' }}</td>
              <td class="px-3 py-3 text-right">
                <button type="button" @click="removeUser(u.id)" class="rounded-lg bg-red-50 px-3 py-1.5 text-xs font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="grid gap-3 md:hidden">
        <article v-for="u in usersList" :key="u.id" class="rounded-xl border border-gray-200/80 bg-white/90 p-3 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90">
          <div class="flex items-start justify-between gap-3">
            <div class="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-amber-50 text-sm font-black text-amber-600 transition-colors dark:bg-amber-900/20 dark:text-amber-400">#{{ u.id }}</div>
            <button type="button" @click="removeUser(u.id)" class="rounded-lg bg-red-50 px-3 py-1.5 text-xs font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
          </div>
          <h3 class="mt-3 text-sm font-black text-slate-900 transition-colors dark:text-white">{{ u.username }}</h3>
          <p class="mt-0.5 text-xs font-semibold text-slate-600 transition-colors dark:text-slate-400">{{ u.email || 'No email' }}</p>
        </article>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useUsers } from '../composables/useUsers'
import PageHeader from './ui/PageHeader.vue'
import LoadingSpinner from './ui/LoadingSpinner.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import EmptyState from './ui/EmptyState.vue'
import InputField from './ui/InputField.vue'
import Button from './ui/Button.vue'

const {
  usersList,
  isLoading,
  error,
  successMsg,
  newUsername,
  newEmail,
  loadUsers,
  addUser,
  removeUser,
} = useUsers()

loadUsers()
</script>
