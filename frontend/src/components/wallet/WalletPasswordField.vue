<template>
  <div class="inline-flex max-w-full items-center gap-1">
    <span
      class="max-w-44 truncate rounded-md bg-gray-100 px-2 py-1 font-mono text-xs text-slate-600 transition-colors dark:bg-slate-700 dark:text-slate-300"
      :class="{ 'text-slate-400 select-none': !revealed }"
      :aria-label="revealed ? 'Revealed password' : 'Hidden password'"
    >
      {{ revealed ? revealedPassword : '••••••••' }}
    </span>

    <button
      type="button"
      :disabled="loading"
      :title="revealed ? 'Hide password' : 'Reveal password'"
      :aria-label="revealed ? 'Hide password' : 'Reveal password'"
      :aria-pressed="revealed"
      class="grid h-7 w-7 place-items-center rounded-lg text-slate-400 transition hover:bg-white hover:text-emerald-600 disabled:opacity-40 dark:hover:bg-slate-700 dark:hover:text-emerald-400"
      @click="toggleReveal"
    >
      <LoaderCircle v-if="loading" class="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
      <EyeOff v-else-if="revealed" class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
      <Eye v-else class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
    </button>

    <button
      type="button"
      :disabled="loading"
      :title="copied ? 'Copied!' : 'Copy password'"
      :aria-label="copied ? 'Password copied' : 'Copy password'"
      class="grid h-7 w-7 place-items-center rounded-lg transition disabled:opacity-40"
      :class="
        copied
          ? 'text-emerald-600 dark:text-emerald-400'
          : 'text-slate-400 hover:bg-white hover:text-emerald-600 dark:hover:bg-slate-700 dark:hover:text-emerald-400'
      "
      @click="copyPassword"
    >
      <Check v-if="copied" class="h-3.5 w-3.5" :stroke-width="3" aria-hidden="true" />
      <Copy v-else class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Check, Copy, Eye, EyeOff, LoaderCircle } from '@lucide/vue';
import { useWalletPassword } from './composables/useWalletPassword';

const props = defineProps<{ reveal: () => Promise<string> }>();

const { revealed, revealedPassword, loading, copied, toggleReveal, copyPassword } =
  useWalletPassword(props.reveal);
</script>
