<template>
  <tr class="transition hover:bg-emerald-50/60 dark:hover:bg-emerald-900/20">
    <td class="px-3 py-3 text-sm font-black text-slate-900 transition-colors dark:text-white">{{ entry.website }}</td>
    <td class="px-3 py-3 text-sm font-semibold text-slate-600 transition-colors dark:text-slate-400">{{ entry.username }}</td>
    <td class="px-3 py-3 text-sm font-mono text-slate-600 transition-colors dark:text-slate-400">
      <span class="rounded-md bg-gray-100 px-2 py-1 transition-colors dark:bg-slate-700">{{ revealed ? revealedPassword : '••••••••' }}</span>
      <button type="button" @click="toggleReveal" :disabled="loading" class="ml-1 inline-grid h-7 w-7 place-items-center rounded-lg text-slate-400 transition hover:bg-white hover:text-emerald-600 disabled:opacity-40 dark:hover:bg-slate-700 dark:hover:text-emerald-400">
        <svg v-if="!revealed" class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
        </svg>
        <svg v-else class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
        </svg>
      </button>
      <button type="button" @click="copyPassword" :disabled="loading" :title="copied ? 'Copied!' : 'Copy password'" class="ml-0.5 inline-grid h-7 w-7 place-items-center rounded-lg transition hover:bg-white disabled:opacity-40 dark:hover:bg-slate-700" :class="copied ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400'">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
        </svg>
      </button>
    </td>
    <td class="max-w-56 px-3 py-3">
      <span class="block truncate text-sm text-slate-500 transition-colors dark:text-slate-400" :title="entry.description">{{ entry.description || '—' }}</span>
    </td>
    <td class="px-3 py-3">
      <span class="whitespace-nowrap rounded-md bg-gray-100 px-2 py-1 text-[10px] font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400">{{ formatDateShort(entry.updated_at || '') }}</span>
    </td>
    <td class="px-3 py-3 text-right">
      <div class="flex justify-end gap-1">
        <button type="button" @click="emit('edit', entry)" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-slate-500 transition hover:bg-emerald-50 hover:text-emerald-700 dark:hover:bg-emerald-900/10 dark:hover:text-emerald-400">Edit</button>
        <button type="button" @click="emit('delete', entry.id)" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-red-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">Delete</button>
      </div>
    </td>
  </tr>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { formatDate as formatDateShort } from '../../lib/utils'
import { revealWalletPassword } from '../../lib/api'
import type { WalletEntry } from '../../types'

interface Props {
  entry: WalletEntry
  userId: number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  edit: [entry: WalletEntry]
  delete: [id: number]
}>()

const revealed = ref(false)
const revealedPassword = ref('')
const loading = ref(false)

const fetchPassword = async () => {
  if (revealedPassword.value) return revealedPassword.value
  loading.value = true
  try {
    revealedPassword.value = await revealWalletPassword(props.userId, props.entry.id)
    return revealedPassword.value
  } finally {
    loading.value = false
  }
}

const toggleReveal = async () => {
  if (revealed.value) {
    revealed.value = false
    return
  }
  await fetchPassword()
  revealed.value = true
}

const copied = ref(false)

const copyPassword = async () => {
  try {
    const pw = await fetchPassword()
    await navigator.clipboard.writeText(pw)
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  } catch {
    // clipboard or fetch failed; ignore
  }
}
</script>
