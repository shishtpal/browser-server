<script setup lang="ts">
import { reactive, ref } from 'vue'
import type { WalletItemView } from '../../composables/useWalletView'

const props = defineProps<{
  item: WalletItemView
  revealFn: (item: WalletItemView) => Promise<string>
  updateFn: (id: number, changes: Record<string, string>) => Promise<void>
}>()

interface RowState {
  revealed: boolean
  password: string
  loading: boolean
  copied: '' | 'username' | 'password'
}

const state = reactive<RowState>({ revealed: false, password: '', loading: false, copied: '' })
const editing = ref(false)
const editLoginProvider = ref('')
const editUsername = ref('')
const editPassword = ref('')
const isSaving = ref(false)

async function ensurePassword(): Promise<string> {
  if (state.password) return state.password
  state.loading = true
  try {
    state.password = await props.revealFn(props.item)
    return state.password
  } finally {
    state.loading = false
  }
}

async function toggleReveal() {
  if (state.revealed) {
    state.revealed = false
    return
  }
  await ensurePassword()
  state.revealed = true
}

async function copyUsername() {
  try {
    await navigator.clipboard.writeText(props.item.username)
    flashCopied('username')
  } catch {
    // ignore
  }
}

async function copyPassword() {
  try {
    const pw = await ensurePassword()
    await navigator.clipboard.writeText(pw)
    flashCopied('password')
  } catch {
    // ignore
  }
}

function flashCopied(which: 'username' | 'password') {
  state.copied = which
  setTimeout(() => {
    if (state.copied === which) state.copied = ''
  }, 1500)
}

async function startEdit() {
  editLoginProvider.value = props.item.loginProvider
  editUsername.value = props.item.username
  editPassword.value = await ensurePassword()
  editing.value = true
}

function cancelEdit() {
  editing.value = false
}

async function saveEdit() {
  const changes: Record<string, string> = {}
  if (editLoginProvider.value !== props.item.loginProvider) changes.login_provider = editLoginProvider.value
  if (editUsername.value !== props.item.username) changes.username = editUsername.value
  if (editPassword.value !== state.password) changes.password = editPassword.value

  if (Object.keys(changes).length === 0) {
    cancelEdit()
    return
  }

  isSaving.value = true
  try {
    await props.updateFn(props.item.id, changes)
    state.password = editPassword.value
    cancelEdit()
  } finally {
    isSaving.value = false
  }
}
</script>

<template>
  <li class="rounded-lg border border-slate-800 bg-slate-900/40 p-3 transition hover:border-slate-700">
    <!-- Edit mode -->
    <template v-if="editing">
      <div class="min-w-0 space-y-2">
        <p class="truncate text-sm font-medium text-slate-100" :title="item.website">{{ item.website }}</p>
        <input
          v-model="editLoginProvider"
          type="text"
          class="w-full rounded border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100 outline-none focus:border-rose-400 focus:ring-1 focus:ring-rose-500/20"
          placeholder="Login provider"
        />
        <div class="flex items-center gap-2 text-xs">
          <svg class="h-3.5 w-3.5 shrink-0 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
            <circle cx="12" cy="7" r="4" />
          </svg>
          <input
            v-model="editUsername"
            type="text"
            class="min-w-0 flex-1 rounded border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100 outline-none focus:border-rose-400 focus:ring-1 focus:ring-rose-500/20"
            placeholder="Username"
          />
        </div>
        <div class="flex items-center gap-2 text-xs">
          <svg class="h-3.5 w-3.5 shrink-0 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="11" width="18" height="11" rx="2" />
            <path d="M7 11V7a5 5 0 0 1 10 0v4" />
          </svg>
          <input
            v-model="editPassword"
            type="text"
            class="min-w-0 flex-1 rounded border border-slate-700 bg-slate-950 px-2 py-1 font-mono text-xs text-slate-100 outline-none focus:border-rose-400 focus:ring-1 focus:ring-rose-500/20"
            placeholder="Password"
          />
        </div>
      </div>
      <div class="mt-2.5 flex gap-1.5">
        <button
          type="button"
          class="rounded-md bg-rose-500 px-3 py-1 text-[11px] font-medium text-white transition hover:bg-rose-400 disabled:opacity-50"
          :disabled="isSaving"
          @click="saveEdit"
        >
          {{ isSaving ? 'Saving…' : 'Save' }}
        </button>
        <button
          type="button"
          class="rounded-md px-3 py-1 text-[11px] font-medium text-slate-400 ring-1 ring-inset ring-slate-700 transition hover:bg-slate-800 hover:text-white"
          @click="cancelEdit"
        >
          Cancel
        </button>
      </div>
    </template>

    <!-- Read-only mode -->
    <template v-else>
      <div class="min-w-0">
        <p class="truncate text-sm font-medium text-slate-100" :title="item.website">{{ item.website }}</p>
        <p class="mt-1 truncate text-xs font-medium text-emerald-400" :title="item.loginProvider">{{ item.loginProvider }}</p>
        <div class="mt-1 flex items-center gap-2 text-xs">
          <svg class="h-3.5 w-3.5 shrink-0 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
            <circle cx="12" cy="7" r="4" />
          </svg>
          <span class="truncate text-slate-400" :title="item.username">{{ item.username }}</span>
        </div>
        <div class="mt-1 flex items-center gap-2 text-xs">
          <svg class="h-3.5 w-3.5 shrink-0 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="11" width="18" height="11" rx="2" />
            <path d="M7 11V7a5 5 0 0 1 10 0v4" />
          </svg>
          <span class="font-mono text-slate-300">
            {{ state.revealed ? state.password : '••••••••••' }}
          </span>
        </div>
      </div>
      <div class="mt-2.5 flex flex-wrap gap-1.5">
        <button
          type="button"
          class="flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium text-slate-300 ring-1 ring-inset ring-slate-700 transition hover:bg-slate-800 hover:text-white disabled:opacity-50"
          :disabled="state.loading"
          @click="toggleReveal"
        >
          <svg v-if="!state.revealed" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7z" />
            <circle cx="12" cy="12" r="3" />
          </svg>
          <svg v-else class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9.88 9.88a3 3 0 1 0 4.24 4.24" />
            <path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c6.5 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68" />
            <path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3.5 7 10 7a9.74 9.74 0 0 0 5.39-1.61" />
            <path d="m2 2 20 20" />
          </svg>
          {{ state.revealed ? 'Hide' : 'Reveal' }}
        </button>
        <button
          type="button"
          class="rounded-md px-2 py-1 text-[11px] font-medium text-slate-300 ring-1 ring-inset ring-slate-700 transition hover:bg-slate-800 hover:text-white"
          @click="copyUsername"
        >
          {{ state.copied === 'username' ? 'Copied!' : 'Copy user' }}
        </button>
        <button
          type="button"
          class="rounded-md px-2 py-1 text-[11px] font-medium text-rose-300 ring-1 ring-inset ring-rose-800 transition hover:bg-rose-500 hover:text-white disabled:opacity-50"
          :disabled="state.loading"
          @click="copyPassword"
        >
          {{ state.copied === 'password' ? 'Copied!' : 'Copy pass' }}
        </button>
        <button
          type="button"
          class="flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium text-slate-300 ring-1 ring-inset ring-slate-700 transition hover:bg-slate-800 hover:text-white"
          @click="startEdit"
        >
          <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
          </svg>
          Edit
        </button>
      </div>
    </template>
  </li>
</template>
