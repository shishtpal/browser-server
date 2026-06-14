<template>
  <article class="rounded-xl border border-gray-200/80 bg-white/90 p-3 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90">
    <div class="flex items-start justify-between gap-3">
      <div class="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-emerald-50 text-sm font-black text-emerald-600 transition-colors dark:bg-emerald-900/20 dark:text-emerald-400">{{ getInitial(entry.website) }}</div>
      <div class="flex gap-1">
        <button type="button" @click="emit('edit', entry)" class="rounded-lg bg-gray-100 px-3 py-1.5 text-xs font-black text-slate-700 transition-colors dark:bg-slate-700 dark:text-slate-200">Edit</button>
        <button type="button" @click="emit('delete', entry.id)" class="rounded-lg bg-red-50 px-3 py-1.5 text-xs font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
      </div>
    </div>
    <h3 class="mt-3 text-sm font-black text-slate-900 transition-colors dark:text-white">{{ entry.website }}</h3>
    <p class="mt-0.5 text-xs font-semibold text-slate-600 transition-colors dark:text-slate-400">{{ entry.username }}</p>
    <div class="mt-2 flex flex-wrap items-center gap-2">
      <span class="rounded-md bg-gray-100 px-2 py-0.5 font-mono text-xs text-slate-600 transition-colors dark:bg-slate-700 dark:text-slate-300">{{ revealed ? revealedPassword : '••••••••' }}</span>
      <button type="button" @click="toggleReveal" :disabled="loading" class="text-xs font-black text-emerald-700 transition-colors disabled:opacity-40 dark:text-emerald-400">{{ revealed ? 'Hide' : 'Reveal' }}</button>
      <button type="button" @click="copyPassword" :disabled="loading" class="text-xs font-black text-emerald-700 transition-colors disabled:opacity-40 dark:text-emerald-400">{{ copied ? 'Copied!' : 'Copy' }}</button>
    </div>
    <p v-if="entry.description" class="mt-2 text-xs leading-5 text-slate-500 transition-colors dark:text-slate-400">{{ entry.description }}</p>
  </article>
</template>

<script setup lang="ts">
import { ref } from 'vue'
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
const copied = ref(false)

const getInitial = (value: string) => value.trim().charAt(0).toUpperCase() || 'W'

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
