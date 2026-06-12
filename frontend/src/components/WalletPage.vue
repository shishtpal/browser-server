<template>
  <div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
    <section class="mb-6 overflow-hidden rounded-[2rem] border border-gray-200/80 bg-white/90 p-5 shadow-2xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-8">
      <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p class="mb-2 inline-flex rounded-full bg-emerald-50 px-3 py-1 text-xs font-bold uppercase tracking-[0.2em] text-emerald-700 transition-colors dark:bg-emerald-900/20 dark:text-emerald-400">Password vault</p>
          <h1 class="text-3xl font-black tracking-tight text-slate-900 transition-colors dark:text-white sm:text-5xl">Wallet</h1>
          <p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600 transition-colors dark:text-slate-400 sm:text-base">Keep credentials organized with masked passwords and quick reveal.</p>
        </div>
        <div class="grid grid-cols-2 gap-3 sm:max-w-sm">
          <div class="rounded-3xl bg-slate-900 p-4 text-center text-white shadow-lg shadow-emerald-500/15 transition-colors dark:bg-slate-950">
            <div class="text-2xl font-black">{{ walletEntries.length }}</div>
            <div class="text-xs font-semibold text-slate-300">Entries</div>
          </div>
          <div class="rounded-3xl bg-emerald-500 p-4 text-center text-white shadow-lg shadow-emerald-500/20">
            <div class="text-2xl font-black">{{ filteredEntries.length }}</div>
            <div class="text-xs font-semibold text-emerald-50">Visible</div>
          </div>
        </div>
      </div>
    </section>

    <section class="mb-6 rounded-3xl border border-gray-200 bg-white/90 p-4 shadow-xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-5">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div class="min-w-0 flex-1">
          <label class="text-sm font-bold text-slate-700 transition-colors dark:text-slate-300" for="wallet-user">Select user</label>
          <select
            id="wallet-user"
            v-model="selectedUserId"
            class="mt-2 w-full rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-emerald-900/30"
          >
            <option :value="null">Choose a user</option>
            <option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }}</option>
          </select>
          <p v-if="users.length === 0" class="mt-2 text-sm text-slate-500 transition-colors dark:text-slate-400">No users yet. <a href="/users" class="font-bold text-emerald-700 transition-colors hover:text-emerald-800 dark:text-emerald-400 dark:hover:text-emerald-300">Create one</a> to start.</p>
        </div>
      </div>
    </section>

    <div v-if="isLoading" class="flex justify-center py-16">
      <div class="h-10 w-10 animate-spin rounded-full border-4 border-emerald-500 border-t-transparent"></div>
      <span class="ml-3 self-center font-semibold text-slate-600 transition-colors dark:text-slate-400">Loading wallet...</span>
    </div>

    <div v-else-if="error" class="mb-6 rounded-3xl border border-red-200 bg-red-50 p-4 text-sm font-semibold text-red-700 shadow-sm transition-colors dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-400">
      {{ error }}
      <button type="button" @click="loadWallet" class="ml-2 underline decoration-red-300 underline-offset-4 transition-colors dark:decoration-red-800">Retry</button>
    </div>

    <template v-else-if="selectedUserId">
      <form @submit.prevent="addEntry" class="mb-6 rounded-3xl border border-gray-200 bg-white p-4 shadow-xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-5">
        <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
          <input v-model="newWebsite" type="text" placeholder="Website" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-emerald-900/30" />
          <input v-model="newUsername" type="text" placeholder="Username" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-emerald-900/30" />
          <input v-model="newPassword" type="password" placeholder="Password" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-emerald-900/30" />
          <input v-model="newDescription" type="text" placeholder="Description (optional)" class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-emerald-900/30" />
        </div>
        <button type="submit" class="mt-4 rounded-2xl bg-gradient-to-r from-emerald-500 to-teal-600 px-5 py-3 text-sm font-black text-white shadow-lg shadow-emerald-500/25 transition hover:-translate-y-0.5 hover:shadow-xl hover:shadow-emerald-500/30 focus:outline-none focus:ring-4 focus:ring-emerald-200 dark:focus:ring-emerald-900/40">
          Add entry
        </button>
      </form>

      <div class="mb-5">
        <input
          v-model="websiteFilter"
          type="text"
          placeholder="Search website, username, or description..."
          class="w-full rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 shadow-sm backdrop-blur-xl transition placeholder:text-slate-400 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-emerald-900/30"
        />
      </div>

      <div v-if="filteredEntries.length === 0" class="rounded-[2rem] border border-dashed border-gray-300 bg-gray-50 p-10 text-center shadow-sm backdrop-blur-xl transition-colors dark:border-slate-600 dark:bg-slate-800/60">
        <div class="mx-auto grid h-14 w-14 place-items-center rounded-3xl bg-emerald-50 text-emerald-500 transition-colors dark:bg-emerald-900/20 dark:text-emerald-400">
          <svg class="h-7 w-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
        </div>
        <h2 class="mt-4 text-lg font-black text-slate-800 transition-colors dark:text-slate-200">{{ walletEntries.length === 0 ? 'No saved credentials' : 'No matching entries' }}</h2>
        <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">{{ walletEntries.length === 0 ? 'Add your first entry above.' : 'Try a different search.' }}</p>
      </div>

      <div v-else>
        <div class="hidden overflow-hidden rounded-[1.75rem] border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90 md:block">
          <table class="min-w-full divide-y divide-gray-200 transition-colors dark:divide-slate-700">
            <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
              <tr>
                <th class="px-5 py-4 text-left text-xs font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Website</th>
                <th class="px-5 py-4 text-left text-xs font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Username</th>
                <th class="px-5 py-4 text-left text-xs font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Password</th>
                <th class="px-5 py-4 text-left text-xs font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Description</th>
                <th class="px-5 py-4 text-right text-xs font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
              <tr v-for="e in filteredEntries" :key="e.id" class="transition hover:bg-emerald-50/60 dark:hover:bg-emerald-900/20">
                <td class="px-5 py-4 text-sm font-black text-slate-900 transition-colors dark:text-white">{{ e.website }}</td>
                <td class="px-5 py-4 text-sm font-semibold text-slate-600 transition-colors dark:text-slate-400">{{ e.username }}</td>
                <td class="px-5 py-4 text-sm font-mono text-slate-600 transition-colors dark:text-slate-400">
                  <span class="rounded-full bg-gray-100 px-3 py-1 transition-colors dark:bg-slate-700">{{ revealedPasswords[e.id] ? e.password : maskPassword(e.password) }}</span>
                  <button type="button" @click="toggleReveal(e.id)" class="ml-2 inline-grid h-8 w-8 place-items-center rounded-full text-slate-400 transition hover:bg-white hover:text-emerald-600 dark:hover:bg-slate-700 dark:hover:text-emerald-400">
                    <svg v-if="!revealedPasswords[e.id]" class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    </svg>
                    <svg v-else class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                    </svg>
                  </button>
                </td>
                <td class="max-w-56 px-5 py-4 text-sm text-slate-500 transition-colors dark:text-slate-400">{{ e.description || '—' }}</td>
                <td class="px-5 py-4 text-right">
                  <div class="flex justify-end gap-2">
                    <button type="button" @click="openEdit(e)" class="rounded-2xl bg-gray-100 px-4 py-2 text-sm font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600">Edit</button>
                    <button type="button" @click="removeEntry(e.id)" class="rounded-2xl bg-red-50 px-4 py-2 text-sm font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="grid gap-3 md:hidden">
          <article v-for="e in filteredEntries" :key="e.id" class="rounded-[1.75rem] border border-gray-200/80 bg-white/90 p-4 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90">
            <div class="flex items-start justify-between gap-3">
              <div class="grid h-11 w-11 shrink-0 place-items-center rounded-2xl bg-emerald-50 font-black text-emerald-600 transition-colors dark:bg-emerald-900/20 dark:text-emerald-400">{{ getInitial(e.website) }}</div>
              <div class="flex gap-2">
                <button type="button" @click="openEdit(e)" class="rounded-2xl bg-gray-100 px-4 py-2 text-sm font-black text-slate-700 transition-colors dark:bg-slate-700 dark:text-slate-200">Edit</button>
                <button type="button" @click="removeEntry(e.id)" class="rounded-2xl bg-red-50 px-4 py-2 text-sm font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
              </div>
            </div>
            <h3 class="mt-4 text-base font-black text-slate-900 transition-colors dark:text-white">{{ e.website }}</h3>
            <p class="mt-1 text-sm font-semibold text-slate-600 transition-colors dark:text-slate-400">{{ e.username }}</p>
            <div class="mt-3 flex flex-wrap items-center gap-2">
              <span class="rounded-full bg-gray-100 px-3 py-1 font-mono text-sm text-slate-600 transition-colors dark:bg-slate-700 dark:text-slate-300">{{ revealedPasswords[e.id] ? e.password : maskPassword(e.password) }}</span>
              <button type="button" @click="toggleReveal(e.id)" class="text-sm font-black text-emerald-700 transition-colors dark:text-emerald-400">{{ revealedPasswords[e.id] ? 'Hide' : 'Reveal' }}</button>
            </div>
            <p v-if="e.description" class="mt-3 text-sm leading-6 text-slate-500 transition-colors dark:text-slate-400">{{ e.description }}</p>
          </article>
        </div>
      </div>
    </template>

    <div v-else class="rounded-[2rem] border border-dashed border-gray-300 bg-gray-50 p-10 text-center shadow-sm backdrop-blur-xl transition-colors dark:border-slate-600 dark:bg-slate-800/60">
      <div class="mx-auto grid h-16 w-16 place-items-center rounded-3xl bg-gray-100 text-slate-400 transition-colors dark:bg-slate-700 dark:text-slate-500">
        <svg class="h-8 w-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </div>
      <h2 class="mt-4 text-lg font-black text-slate-800 transition-colors dark:text-slate-200">Choose a workspace</h2>
      <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">Select a user to manage their wallet.</p>
    </div>

    <div v-if="editing" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-4 backdrop-blur-sm">
      <div class="w-full max-w-lg rounded-[2rem] border border-gray-200 bg-white p-5 shadow-2xl shadow-gray-900/30 transition-colors dark:border-white/10 dark:bg-slate-800 dark:shadow-slate-950/30 sm:p-6">
        <div class="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 class="text-xl font-black text-slate-900 transition-colors dark:text-white">Edit wallet entry</h2>
            <p class="mt-1 text-sm text-slate-500 transition-colors dark:text-slate-400">Update saved credentials.</p>
          </div>
          <button type="button" @click="editing = null" class="grid h-9 w-9 place-items-center rounded-2xl bg-gray-100 text-slate-500 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-400 dark:hover:bg-slate-600" aria-label="Close">×</button>
        </div>
        <div class="grid gap-3">
          <input v-model="editForm.website" type="text" placeholder="Website" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-emerald-900/30" />
          <input v-model="editForm.username" type="text" placeholder="Username" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-emerald-900/30" />
          <input v-model="editForm.password" type="text" placeholder="Password" required class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-emerald-900/30" />
          <input v-model="editForm.description" type="text" placeholder="Description" class="rounded-2xl border border-gray-300 bg-gray-50 px-4 py-3 text-sm font-semibold text-slate-700 focus:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-emerald-900/30" />
        </div>
        <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button type="button" @click="editing = null" class="rounded-2xl bg-gray-100 px-5 py-3 text-sm font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600">Cancel</button>
          <button type="button" @click="saveEdit" class="rounded-2xl bg-gradient-to-r from-emerald-500 to-teal-600 px-5 py-3 text-sm font-black text-white shadow-lg shadow-emerald-500/20">Save changes</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { getWallet, createWalletEntry, updateWalletEntry, deleteWalletEntry } from '../lib/api'
import { useUser } from '../composables/useUser'
import type { WalletEntry } from '../types'

const { users, currentUserId, setUser, clearUser } = useUser()

const selectedUserId = ref<number | null>(currentUserId.value)
const walletEntries = ref<WalletEntry[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const websiteFilter = ref('')

const newWebsite = ref('')
const newUsername = ref('')
const newPassword = ref('')
const newDescription = ref('')

const revealedPasswords = reactive<Record<number, boolean>>({})

const editing = ref<WalletEntry | null>(null)
const editForm = ref({ website: '', username: '', password: '', description: '' })

const filteredEntries = computed(() => {
  if (!websiteFilter.value.trim()) return walletEntries.value
  const q = websiteFilter.value.toLowerCase()
  return walletEntries.value.filter(e =>
    e.website.toLowerCase().includes(q) ||
    e.username.toLowerCase().includes(q) ||
    e.description.toLowerCase().includes(q)
  )
})

const loadWallet = async () => {
  if (!selectedUserId.value) return
  isLoading.value = true
  error.value = null
  Object.keys(revealedPasswords).forEach(key => { delete revealedPasswords[Number(key)] })
  try {
    walletEntries.value = await getWallet(selectedUserId.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load wallet'
  } finally {
    isLoading.value = false
  }
}

const addEntry = async () => {
  if (!selectedUserId.value || !newWebsite.value.trim() || !newUsername.value.trim() || !newPassword.value) return
  try {
    await createWalletEntry({
      user_id: selectedUserId.value,
      website: newWebsite.value.trim(),
      username: newUsername.value.trim(),
      password: newPassword.value,
      description: newDescription.value.trim() || undefined,
    })
    newWebsite.value = ''
    newUsername.value = ''
    newPassword.value = ''
    newDescription.value = ''
    await loadWallet()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to add entry'
  }
}

const toggleReveal = (id: number) => {
  revealedPasswords[id] = !revealedPasswords[id]
}

const maskPassword = (password: string) => {
  if (!password) return '••••'
  if (password.length <= 4) return '••••'
  return `${password.slice(0, 2)}${'•'.repeat(Math.min(password.length - 2, 8))}`
}

const getInitial = (value: string) => value.trim().charAt(0).toUpperCase() || 'W'

const openEdit = (e: WalletEntry) => {
  editing.value = e
  editForm.value = {
    website: e.website,
    username: e.username,
    password: e.password,
    description: e.description,
  }
}

const saveEdit = async () => {
  if (!editing.value) return
  try {
    await updateWalletEntry(editing.value.id, editForm.value)
    editing.value = null
    await loadWallet()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to update entry'
  }
}

const removeEntry = async (id: number) => {
  if (!confirm('Delete this wallet entry?')) return
  try {
    await deleteWalletEntry(id)
    await loadWallet()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to delete entry'
  }
}

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
