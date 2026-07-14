<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  domain: string
  hasItems: boolean
}>()

const emit = defineEmits<{
  (event: 'add', payload: { login_provider: string; username: string; password: string; description?: string }): void
}>()

const showForm = ref(!props.hasItems)
const loginProvider = ref('Password')
const username = ref('')
const password = ref('')
const description = ref('')
const isAdding = ref(false)
const error = ref<string | null>(null)

function reset() {
  loginProvider.value = 'Password'
  username.value = ''
  password.value = ''
  description.value = ''
  error.value = null
}

async function submit() {
  if (!loginProvider.value.trim() || !username.value.trim()) return
  if (loginProvider.value.trim().toLowerCase() === 'password' && !password.value) {
    error.value = 'A password is required for Password logins.'
    return
  }

  isAdding.value = true
  error.value = null
  try {
    emit('add', {
      login_provider: loginProvider.value.trim(),
      username: username.value.trim(),
      password: password.value,
      description: description.value.trim() || undefined,
    })
    reset()
    showForm.value = false
  } finally {
    isAdding.value = false
  }
}

defineExpose({ showForm })
</script>

<template>
  <div class="mb-2 rounded-lg border border-slate-800 bg-slate-900/40 p-3">
    <button
      v-if="hasItems && !showForm"
      type="button"
      class="w-full rounded-md bg-emerald-500 px-3 py-1.5 text-xs font-semibold text-slate-950 transition hover:bg-emerald-400"
      @click="showForm = true"
    >
      Add another account
    </button>
    <form v-else class="space-y-2" @submit.prevent="submit">
      <div class="flex items-center justify-between gap-2">
        <p class="truncate text-xs font-semibold text-slate-200">Add account for {{ domain }}</p>
        <button v-if="hasItems" type="button" class="text-xs text-slate-500 hover:text-white" @click="showForm = false">Cancel</button>
      </div>
      <input v-model="loginProvider" type="text" required placeholder="Provider (Password, Google, GitHub...)" class="w-full rounded border border-slate-700 bg-slate-950 px-2 py-1.5 text-xs text-slate-100 outline-none focus:border-emerald-400" />
      <input v-model="username" type="text" required placeholder="Username or email" class="w-full rounded border border-slate-700 bg-slate-950 px-2 py-1.5 text-xs text-slate-100 outline-none focus:border-emerald-400" />
      <input v-model="password" type="password" placeholder="Password (optional for provider login)" class="w-full rounded border border-slate-700 bg-slate-950 px-2 py-1.5 text-xs text-slate-100 outline-none focus:border-emerald-400" />
      <input v-model="description" type="text" placeholder="Description (optional)" class="w-full rounded border border-slate-700 bg-slate-950 px-2 py-1.5 text-xs text-slate-100 outline-none focus:border-emerald-400" />
      <p v-if="error" class="text-xs text-rose-400">{{ error }}</p>
      <button type="submit" :disabled="isAdding" class="w-full rounded-md bg-emerald-500 px-3 py-1.5 text-xs font-semibold text-slate-950 transition hover:bg-emerald-400 disabled:opacity-50">
        {{ isAdding ? 'Adding…' : 'Save account' }}
      </button>
    </form>
  </div>
</template>
