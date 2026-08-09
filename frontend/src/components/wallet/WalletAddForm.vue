<template>
  <form
    class="mb-4 rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90"
    aria-label="Add a wallet entry"
    @submit.prevent="onSubmit"
  >
    <div class="grid gap-2 sm:grid-cols-2 lg:flex lg:items-center lg:gap-2">
      <div class="relative">
        <Globe
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="website"
          type="text"
          placeholder="Website domain"
          required
          :class="[inputClass, 'pl-9']"
        />
      </div>
      <div class="relative">
        <ShieldCheck
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="loginProvider"
          type="text"
          placeholder="Provider (Google, GitHub, Password...)"
          required
          :class="[inputClass, 'pl-9']"
        />
      </div>
      <div class="relative">
        <User
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="username"
          type="text"
          placeholder="Username"
          required
          :class="[inputClass, 'pl-9']"
        />
      </div>
      <div class="relative">
        <KeyRound
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="password"
          type="password"
          placeholder="Password (optional for provider login)"
          autocomplete="new-password"
          :class="[inputClass, 'pl-9']"
        />
      </div>
      <div class="relative lg:flex-1">
        <AlignLeft
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="description"
          type="text"
          placeholder="Description (optional)"
          :class="[inputClass, 'pl-9']"
        />
      </div>
      <Button
        type="submit"
        variant="gradient-emerald"
        size="sm"
        class="sm:col-span-2 lg:col-span-1"
      >
        <span class="inline-flex items-center gap-1.5">
          <Plus class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          Add
        </span>
      </Button>
    </div>
    <p
      v-if="passwordError"
      class="mt-2 flex items-center gap-1.5 text-xs font-semibold text-rose-600 dark:text-rose-400"
      role="alert"
    >
      <CircleAlert class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
      {{ passwordError }}
    </p>
  </form>
</template>

<script setup lang="ts">
import type { WalletEntryInput } from './composables/useWallet';
import { ref } from 'vue';
import { AlignLeft, CircleAlert, Globe, KeyRound, Plus, ShieldCheck, User } from '@lucide/vue';
import Button from '../ui/Button.vue';
import { isPasswordless } from './walletFormat';

const emit = defineEmits<{ submit: [input: WalletEntryInput] }>();

const inputClass =
  'w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-400 focus:ring-4 focus:ring-emerald-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-emerald-900/30';

const website = ref('');
const loginProvider = ref('Password');
const username = ref('');
const password = ref('');
const description = ref('');
const passwordError = ref('');

const onSubmit = () => {
  passwordError.value = '';
  if (!isPasswordless(loginProvider.value) && !password.value) {
    passwordError.value = 'A password is required for "Password" provider logins.';
    return;
  }
  emit('submit', {
    website: website.value,
    login_provider: loginProvider.value,
    username: username.value,
    password: password.value,
    description: description.value,
  });
  website.value = '';
  loginProvider.value = 'Password';
  username.value = '';
  password.value = '';
  description.value = '';
};
</script>
