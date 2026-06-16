<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { createApiClient, useExtensionSettings, useUserId, useWalletView } from '../composables/composables'
import type { WalletItemView } from '../composables/useWalletView'
import type { PanelStatus } from './types'

const emit = defineEmits<{ (event: 'status', status: PanelStatus): void }>()

const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

const { currentDomain, items, total, isLoading, errorMessage, init, refresh, reveal, update } = useWalletView(client, userId)

interface RowState {
  revealed: boolean
  password: string
  loading: boolean
  copied: '' | 'username' | 'password'
}

const rows = reactive<Record<number, RowState>>({})
const editingId = ref<number | null>(null)
const editUsername = ref('')
const editPassword = ref('')
const isSaving = ref(false)

function rowState(id: number): RowState {
  if (!rows[id]) {
    rows[id] = { revealed: false, password: '', loading: false, copied: '' }
  }
  return rows[id]
}

async function ensurePassword(item: WalletItemView): Promise<string> {
  const state = rowState(item.id)
  if (state.password) return state.password
  state.loading = true
  try {
    state.password = await reveal(item)
    return state.password
  } finally {
    state.loading = false
  }
}

async function toggleReveal(item: WalletItemView) {
  const state = rowState(item.id)
  if (state.revealed) {
    state.revealed = false
    return
  }
  await ensurePassword(item)
  state.revealed = true
}

async function copyUsername(item: WalletItemView) {
  const state = rowState(item.id)
  try {
    await navigator.clipboard.writeText(item.username)
    flashCopied(state, 'username')
  } catch {
    // ignore
  }
}

async function copyPassword(item: WalletItemView) {
  const state = rowState(item.id)
  try {
    const pw = await ensurePassword(item)
    await navigator.clipboard.writeText(pw)
    flashCopied(state, 'password')
  } catch {
    // ignore
  }
}

function flashCopied(state: RowState, which: 'username' | 'password') {
  state.copied = which
  setTimeout(() => {
    if (state.copied === which) state.copied = ''
  }, 1500)
}

async function startEdit(item: WalletItemView) {
  editingId.value = item.id
  editUsername.value = item.username
  editPassword.value = await ensurePassword(item)
}

function cancelEdit() {
  editingId.value = null
  editUsername.value = ''
  editPassword.value = ''
}

async function saveEdit(item: WalletItemView) {
  const changes: Record<string, string> = {}
  if (editUsername.value !== item.username) changes.username = editUsername.value
  const currentPassword = rowState(item.id).password
  if (editPassword.value !== currentPassword) changes.password = editPassword.value

  if (Object.keys(changes).length === 0) {
    cancelEdit()
    return
  }

  isSaving.value = true
  try {
    await update(item.id, changes)
    rowState(item.id).password = editPassword.value
    cancelEdit()
  } finally {
    isSaving.value = false
  }
}

defineExpose({
  refresh: () => void refresh(),
})

const isReady = computed(() => Boolean(client.value) && userId.value > 0)
const showSkeleton = computed(() => isLoading.value && items.value.length === 0)

watch(
  [isReady, isLoading, errorMessage, total],
  () => {
    emit('status', {
      count: total.value,
      state: errorMessage.value ? 'error' : isLoading.value ? 'loading' : 'ready',
    })
  },
  { immediate: true },
)

// Auto-load once the API client is ready (settings load asynchronously).
watch(
  [client, userId],
  () => {
    if (isReady.value) void init()
  },
  { immediate: true },
)
</script>

<template>
  <section class="px-2 py-2">
    <!-- Skeleton -->
    <div v-if="showSkeleton" class="space-y-2 px-1">
      <div v-for="n in 3" :key="n" class="space-y-2 rounded-lg border border-slate-800 p-3">
        <div class="h-3 w-2/3 animate-pulse rounded bg-slate-800" />
        <div class="h-2.5 w-1/2 animate-pulse rounded bg-slate-800/70" />
        <div class="h-2.5 w-1/3 animate-pulse rounded bg-slate-800/70" />
      </div>
    </div>

    <!-- Error -->
    <div v-else-if="errorMessage" class="flex flex-col items-center gap-3 px-4 py-12 text-center">
      <div class="flex h-12 w-12 items-center justify-center rounded-full bg-rose-500/10">
        <svg class="h-6 w-6 text-rose-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 9v4M12 17h.01" />
          <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
        </svg>
      </div>
      <p class="text-sm font-medium text-slate-300">Can't reach the server</p>
      <p class="max-w-[280px] text-xs text-slate-500">{{ errorMessage }}</p>
      <button type="button" class="rounded-lg bg-rose-500 px-4 py-1.5 text-xs font-medium text-white transition hover:bg-rose-400" @click="refresh">
        Try again
      </button>
    </div>

    <!-- No domain -->
    <div v-else-if="!currentDomain" class="flex flex-col items-center gap-2 px-4 py-12 text-center">
      <p class="text-sm font-medium text-slate-300">No active site</p>
      <p class="max-w-[260px] text-xs text-slate-500">Open a regular web page to see saved credentials for it.</p>
    </div>

    <!-- Empty -->
    <div v-else-if="items.length === 0" class="flex flex-col items-center gap-3 px-4 py-12 text-center">
      <div class="flex h-12 w-12 items-center justify-center rounded-full bg-slate-800">
        <svg class="h-6 w-6 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="11" width="18" height="11" rx="2" />
          <path d="M7 11V7a5 5 0 0 1 10 0v4" />
        </svg>
      </div>
      <p class="text-sm font-medium text-slate-300">No saved passwords</p>
      <p class="max-w-[260px] text-xs text-slate-500">Credentials saved for {{ currentDomain }} will appear here.</p>
    </div>

    <!-- List -->
    <ul v-else class="space-y-1.5">
      <li
        v-for="item in items"
        :key="item.id"
        class="rounded-lg border border-slate-800 bg-slate-900/40 p-3 transition hover:border-slate-700"
      >
        <!-- Edit mode -->
        <template v-if="editingId === item.id">
          <div class="min-w-0 space-y-2">
            <p class="truncate text-sm font-medium text-slate-100" :title="item.website">{{ item.website }}</p>
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
              @click="saveEdit(item)"
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
                {{ rowState(item.id).revealed ? rowState(item.id).password : '••••••••••' }}
              </span>
            </div>
          </div>
          <div class="mt-2.5 flex flex-wrap gap-1.5">
            <button
              type="button"
              class="flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium text-slate-300 ring-1 ring-inset ring-slate-700 transition hover:bg-slate-800 hover:text-white disabled:opacity-50"
              :disabled="rowState(item.id).loading"
              @click="toggleReveal(item)"
            >
              <svg v-if="!rowState(item.id).revealed" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
              <svg v-else class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M9.88 9.88a3 3 0 1 0 4.24 4.24" />
                <path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c6.5 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68" />
                <path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3.5 7 10 7a9.74 9.74 0 0 0 5.39-1.61" />
                <path d="m2 2 20 20" />
              </svg>
              {{ rowState(item.id).revealed ? 'Hide' : 'Reveal' }}
            </button>
            <button
              type="button"
              class="rounded-md px-2 py-1 text-[11px] font-medium text-slate-300 ring-1 ring-inset ring-slate-700 transition hover:bg-slate-800 hover:text-white"
              @click="copyUsername(item)"
            >
              {{ rowState(item.id).copied === 'username' ? 'Copied!' : 'Copy user' }}
            </button>
            <button
              type="button"
              class="rounded-md px-2 py-1 text-[11px] font-medium text-rose-300 ring-1 ring-inset ring-rose-800 transition hover:bg-rose-500 hover:text-white disabled:opacity-50"
              :disabled="rowState(item.id).loading"
              @click="copyPassword(item)"
            >
              {{ rowState(item.id).copied === 'password' ? 'Copied!' : 'Copy pass' }}
            </button>
            <button
              type="button"
              class="flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium text-slate-300 ring-1 ring-inset ring-slate-700 transition hover:bg-slate-800 hover:text-white"
              @click="startEdit(item)"
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
    </ul>
  </section>
</template>
