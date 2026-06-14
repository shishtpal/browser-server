<script setup lang="ts">
import { computed, onMounted, reactive, watch } from 'vue'
import { createApiClient, useExtensionSettings, useUserId, useWalletView } from '../composables/composables'
import type { WalletItemView } from '../composables/useWalletView'

const emit = defineEmits<{ (event: 'stats', label: string): void }>()

const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

const { domainDisplay, currentDomain, items, stats, errorMessage, init, reveal } = useWalletView(client, userId)

interface RowState {
  revealed: boolean
  password: string
  loading: boolean
  copied: '' | 'username' | 'password'
}

const rows = reactive<Record<number, RowState>>({})

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

defineExpose({
  refresh: () => void init(),
})

onMounted(() => {
  void init()
})

watch(stats, (label) => emit('stats', label))
watch(errorMessage, () => emit('stats', stats.value))
</script>

<template>
  <section class="max-h-[392px] overflow-y-auto">
    <p class="border-b border-slate-800 px-4 py-2 text-center text-xs text-slate-400">
      {{ domainDisplay }}
    </p>

    <p v-if="errorMessage" class="px-4 py-6 text-center text-xs text-rose-300">
      {{ errorMessage }}
    </p>
    <p
      v-else-if="!currentDomain"
      class="px-4 py-10 text-center text-sm text-slate-500"
    >
      No active domain detected.
    </p>
    <p
      v-else-if="items.length === 0"
      class="px-4 py-10 text-center text-sm text-slate-500"
    >
      No saved passwords for this site.
    </p>
    <ul v-else class="divide-y divide-slate-800">
      <li
        v-for="item in items"
        :key="item.id"
        class="px-4 py-3 hover:bg-slate-800/40"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-medium text-slate-100" :title="item.website">{{ item.website }}</p>
            <p class="mt-0.5 truncate text-xs text-slate-400" :title="item.username">{{ item.username }}</p>
            <p class="mt-1 font-mono text-xs text-slate-300">
              {{ rowState(item.id).revealed ? rowState(item.id).password : '••••••••' }}
            </p>
          </div>
        </div>
        <div class="mt-2 flex flex-wrap gap-1.5">
          <button
            type="button"
            class="rounded px-2 py-1 text-[11px] font-medium text-slate-300 ring-1 ring-slate-700 hover:bg-slate-700 hover:text-white"
            :disabled="rowState(item.id).loading"
            @click="toggleReveal(item)"
          >
            {{ rowState(item.id).revealed ? 'Hide' : 'Reveal' }}
          </button>
          <button
            type="button"
            class="rounded px-2 py-1 text-[11px] font-medium text-slate-300 ring-1 ring-slate-700 hover:bg-slate-700 hover:text-white"
            @click="copyUsername(item)"
          >
            {{ rowState(item.id).copied === 'username' ? 'Copied!' : 'Copy user' }}
          </button>
          <button
            type="button"
            class="rounded px-2 py-1 text-[11px] font-medium text-rose-300 ring-1 ring-rose-800 hover:bg-rose-500 hover:text-white"
            :disabled="rowState(item.id).loading"
            @click="copyPassword(item)"
          >
            {{ rowState(item.id).copied === 'password' ? 'Copied!' : 'Copy pass' }}
          </button>
        </div>
      </li>
    </ul>
  </section>
</template>
