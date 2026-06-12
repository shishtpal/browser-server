<template>
  <div class="mx-auto max-w-6xl px-4 py-6 sm:px-6 lg:px-8">
    <section class="mb-6 overflow-hidden rounded-[2rem] border border-gray-200/80 bg-white/90 p-5 shadow-2xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-8">
      <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p class="mb-2 inline-flex rounded-full bg-amber-50 px-3 py-1 text-xs font-bold uppercase tracking-[0.2em] text-amber-700 transition-colors dark:bg-amber-900/20 dark:text-amber-400">Workspace</p>
          <h1 class="text-3xl font-black tracking-tight text-slate-900 transition-colors dark:text-white sm:text-5xl">Users</h1>
          <p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600 transition-colors dark:text-slate-400 sm:text-base">Create personal workspaces for todos, bookmarks, history, and wallet entries.</p>
        </div>
        <div class="grid h-24 w-24 shrink-0 place-items-center rounded-[2rem] bg-gradient-to-br from-amber-400 to-orange-500 text-4xl font-black text-white shadow-xl shadow-orange-500/25 sm:h-28 sm:w-28 sm:text-5xl">
          {{ usersList.length }}
        </div>
      </div>
    </section>

    <form @submit.prevent="addUser" class="mb-6 rounded-3xl border border-gray-200 bg-white p-4 shadow-xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-5">
      <div class="mb-4">
        <h2 class="text-lg font-black text-slate-900 transition-colors dark:text-white">Create user</h2>
        <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">Add a workspace that can hold personal data.</p>
      </div>
      <div class="grid gap-3 md:grid-cols-[1fr_1fr_auto]">
        <input v-model="newUsername" type="text" placeholder="Username" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-amber-400 focus:outline-none focus:ring-4 focus:ring-amber-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-amber-900/30" />
        <input v-model="newEmail" type="email" placeholder="Email (optional)" class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-amber-400 focus:outline-none focus:ring-4 focus:ring-amber-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-amber-900/30" />
        <button type="submit" class="rounded-2xl bg-gradient-to-r from-amber-500 to-orange-600 px-5 py-3 text-sm font-black text-white shadow-lg shadow-orange-500/25 transition hover:-translate-y-0.5 hover:shadow-xl hover:shadow-orange-500/30 focus:outline-none focus:ring-4 focus:ring-amber-200 dark:focus:ring-amber-900/40">
          Create
        </button>
      </div>
      <div v-if="successMsg" class="mt-4 rounded-2xl bg-emerald-50 px-4 py-3 text-sm font-bold text-emerald-700 transition-colors dark:bg-emerald-900/20 dark:text-emerald-400">{{ successMsg }}</div>
    </form>

    <div v-if="isLoading" class="flex justify-center py-16">
      <div class="h-10 w-10 animate-spin rounded-full border-4 border-amber-500 border-t-transparent"></div>
      <span class="ml-3 self-center font-semibold text-slate-600 transition-colors dark:text-slate-400">Loading users...</span>
    </div>

    <div v-else-if="error" class="mb-6 rounded-3xl border border-red-200 bg-red-50 p-4 text-sm font-semibold text-red-700 shadow-sm transition-colors dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-400">
      {{ error }}
      <button type="button" @click="loadUsers" class="ml-2 underline decoration-red-300 underline-offset-4 transition-colors dark:decoration-red-800">Retry</button>
    </div>

    <div v-else-if="usersList.length === 0" class="rounded-[2rem] border border-dashed border-gray-300 bg-gray-50 p-10 text-center shadow-sm backdrop-blur-xl transition-colors dark:border-slate-600 dark:bg-slate-800/60">
      <div class="mx-auto grid h-14 w-14 place-items-center rounded-3xl bg-amber-50 text-amber-500 transition-colors dark:bg-amber-900/20 dark:text-amber-400">
        <svg class="h-7 w-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
      </div>
      <h2 class="mt-4 text-lg font-black text-slate-800 transition-colors dark:text-slate-200">No users yet</h2>
      <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">Create your first workspace above.</p>
    </div>

    <div v-else>
      <div class="mb-4 hidden overflow-hidden rounded-[1.75rem] border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90 md:block">
        <table class="min-w-full divide-y divide-gray-200 transition-colors dark:divide-slate-700">
          <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
            <tr>
              <th class="px-5 py-4 text-left text-xs font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">ID</th>
              <th class="px-5 py-4 text-left text-xs font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Username</th>
              <th class="px-5 py-4 text-left text-xs font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Email</th>
              <th class="px-5 py-4 text-right text-xs font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
            <tr v-for="u in usersList" :key="u.id" class="transition hover:bg-amber-50/60 dark:hover:bg-amber-900/20">
              <td class="px-5 py-4 text-sm font-mono font-bold text-slate-400 transition-colors dark:text-slate-500">#{{ u.id }}</td>
              <td class="px-5 py-4 text-sm font-black text-slate-900 transition-colors dark:text-white">{{ u.username }}</td>
              <td class="px-5 py-4 text-sm font-semibold text-slate-600 transition-colors dark:text-slate-400">{{ u.email || '—' }}</td>
              <td class="px-5 py-4 text-right">
                <button type="button" @click="removeUser(u.id)" class="rounded-2xl bg-red-50 px-4 py-2 text-sm font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">
                  Delete
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="grid gap-3 md:hidden">
        <article v-for="u in usersList" :key="u.id" class="rounded-[1.75rem] border border-gray-200/80 bg-white/90 p-4 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90">
          <div class="flex items-start justify-between gap-3">
            <div class="grid h-12 w-12 shrink-0 place-items-center rounded-2xl bg-amber-50 text-lg font-black text-amber-600 transition-colors dark:bg-amber-900/20 dark:text-amber-400">#{{ u.id }}</div>
            <button type="button" @click="removeUser(u.id)" class="rounded-2xl bg-red-50 px-4 py-2 text-sm font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
          </div>
          <h3 class="mt-4 text-base font-black text-slate-900 transition-colors dark:text-white">{{ u.username }}</h3>
          <p class="mt-1 text-sm font-semibold text-slate-600 transition-colors dark:text-slate-400">{{ u.email || 'No email' }}</p>
        </article>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { getUsers, createUser, deleteUser } from '../lib/api'
import type { User } from '../types'

const usersList = ref<User[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const successMsg = ref('')

const newUsername = ref('')
const newEmail = ref('')

const loadUsers = async () => {
  isLoading.value = true
  error.value = null
  try {
    usersList.value = await getUsers()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load users'
  } finally {
    isLoading.value = false
  }
}

const addUser = async () => {
  if (!newUsername.value.trim()) return
  const username = newUsername.value.trim()
  try {
    await createUser({
      username,
      email: newEmail.value.trim() || undefined,
    })
    successMsg.value = `User "${username}" created!`
    newUsername.value = ''
    newEmail.value = ''
    await loadUsers()
    setTimeout(() => { successMsg.value = '' }, 3000)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to create user'
  }
}

const removeUser = async (id: number) => {
  if (!confirm('Delete this user? This will remove all their data (todos, bookmarks, history, wallet entries).')) return
  try {
    await deleteUser(id)
    await loadUsers()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to delete user'
  }
}

loadUsers()
</script>
