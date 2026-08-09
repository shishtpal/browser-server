<template>
  <article
    class="rounded-xl border border-gray-200/80 bg-white/90 p-3.5 shadow-sm transition hover:-translate-y-0.5 hover:border-emerald-200 hover:shadow-md dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-emerald-500/30"
  >
    <div class="flex items-start gap-3">
      <div
        class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-emerald-50 text-sm font-black text-emerald-600 transition-colors dark:bg-emerald-900/20 dark:text-emerald-400"
        aria-hidden="true"
      >
        {{ walletInitial(entry.website) }}
      </div>

      <div class="min-w-0 flex-1">
        <h3
          class="truncate text-sm font-black text-slate-900 dark:text-white"
          :title="entry.website"
        >
          {{ entry.website }}
        </h3>
        <p
          class="mt-0.5 inline-flex items-center gap-1 text-xs font-black text-emerald-700 dark:text-emerald-400"
        >
          <ShieldCheck class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          {{ entry.login_provider }}
        </p>
        <p
          class="mt-0.5 truncate text-xs font-semibold text-slate-600 dark:text-slate-400"
          :title="entry.username"
        >
          {{ entry.username }}
        </p>
      </div>
    </div>

    <div class="mt-3 flex items-center">
      <WalletPasswordField :reveal="reveal" />
    </div>

    <p
      v-if="entry.description"
      class="mt-2 line-clamp-2 text-xs leading-5 text-slate-500 transition-colors dark:text-slate-400"
    >
      {{ entry.description }}
    </p>

    <div class="mt-3 flex gap-1.5 border-t border-gray-100 pt-3 dark:border-slate-700/50">
      <button
        type="button"
        class="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-gray-100 px-3 py-2 text-xs font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600"
        @click="$emit('edit', entry)"
      >
        <Pencil class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        Edit
      </button>
      <button
        type="button"
        class="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-red-50 px-3 py-2 text-xs font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30"
        @click="$emit('delete', entry.id)"
      >
        <Trash2 class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        Delete
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import type { WalletEntry } from '../../../types';
import { Pencil, ShieldCheck, Trash2 } from '@lucide/vue';
import { revealWalletPassword } from '../../../lib/api';
import { walletInitial } from '../walletFormat';
import WalletPasswordField from '../WalletPasswordField.vue';

const props = defineProps<{
  entry: WalletEntry;
  userId: number;
}>();

defineEmits<{
  edit: [entry: WalletEntry];
  delete: [id: number];
}>();

const reveal = () => revealWalletPassword(props.userId, props.entry.id);
</script>
