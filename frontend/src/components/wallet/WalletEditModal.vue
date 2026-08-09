<template>
  <Modal
    :open="!!entry"
    title="Edit wallet entry"
    description="Update the saved credentials. Leave the password empty to keep the current one."
    @close="$emit('close')"
  >
    <form v-if="entry" :key="entry.id" class="flex flex-col gap-3.5" @submit.prevent="onSave">
      <FormField label="Website" required>
        <div class="relative">
          <Globe
            class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
            aria-hidden="true"
          />
          <input v-model="form.website" type="text" required :class="[inputClass, 'pl-9']" />
        </div>
      </FormField>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <FormField label="Login provider" required>
          <select v-model="form.login_provider" :class="inputClass">
            <option value="Password">Password</option>
            <option>Google</option>
            <option>GitHub</option>
            <option>Microsoft</option>
            <option>Apple</option>
          </select>
        </FormField>
        <FormField label="Username" required>
          <input v-model="form.username" type="text" required :class="inputClass" />
        </FormField>
      </div>

      <FormField
        label="Password"
        :help-text="
          isRevealing ? 'Loading current password…' : 'Fetches the current password on open.'
        "
      >
        <div class="relative">
          <KeyRound
            class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
            aria-hidden="true"
          />
          <input
            v-model="form.password"
            type="text"
            autocomplete="off"
            placeholder="Password (leave empty to keep current)"
            :disabled="isRevealing"
            :class="[inputClass, 'pl-9']"
          />
        </div>
      </FormField>

      <FormField label="Description">
        <input
          v-model="form.description"
          type="text"
          placeholder="Description"
          :class="inputClass"
        />
      </FormField>

      <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <Button type="button" variant="secondary" size="sm" @click="$emit('close')">
          Cancel
        </Button>
        <Button
          type="submit"
          variant="gradient-emerald"
          size="sm"
          :loading="saving"
          loading-text="Saving…"
          :disabled="isRevealing"
        >
          <span class="inline-flex items-center gap-1.5">
            <Check class="h-3.5 w-3.5" :stroke-width="3" aria-hidden="true" />
            Save changes
          </span>
        </Button>
      </div>
    </form>
  </Modal>
</template>

<script setup lang="ts">
import type { WalletEntry } from '../../types';
import { ref, watch } from 'vue';
import { Check, Globe, KeyRound } from '@lucide/vue';
import Button from '../ui/Button.vue';
import FormField from '../ui/FormField.vue';
import Modal from '../ui/Modal.vue';
import type { WalletEntryInput } from './composables/useWallet';

const props = defineProps<{
  entry: WalletEntry | null;
  /** Revealed password of the entry being edited ("" while fetching). */
  revealedPassword: string;
  isRevealing: boolean;
}>();

const emit = defineEmits<{
  close: [];
  save: [input: WalletEntryInput];
}>();

const inputClass =
  'w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-400 focus:ring-4 focus:ring-emerald-100 focus:outline-none disabled:opacity-60 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-emerald-900/30';

const form = ref({
  website: '',
  login_provider: 'Password',
  username: '',
  password: '',
  description: '',
});
const saving = ref(false);

watch(
  () => props.entry,
  (e) => {
    if (!e) return;
    form.value = {
      website: e.website,
      login_provider: e.login_provider || 'Password',
      username: e.username,
      password: '',
      description: e.description ?? '',
    };
  },
  { immediate: true },
);

// Prefill the password once the page reveals it.
watch(
  () => props.revealedPassword,
  (pw) => {
    if (props.entry && pw) form.value.password = pw;
  },
);

const onSave = () => {
  if (!form.value.website.trim() || !form.value.username.trim()) return;
  saving.value = true;
  try {
    emit('save', { ...form.value });
  } finally {
    saving.value = false;
  }
};
</script>
