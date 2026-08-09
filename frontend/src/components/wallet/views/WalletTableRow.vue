<template>
  <tr class="transition hover:bg-emerald-50/60 dark:hover:bg-emerald-900/20">
    <td class="px-3 py-3">
      <div class="flex min-w-0 items-center gap-2.5">
        <div
          class="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-emerald-50 text-sm font-black text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-400"
          aria-hidden="true"
        >
          {{ walletInitial(entry.website) }}
        </div>
        <span
          class="truncate text-sm font-black text-slate-900 dark:text-white"
          :title="entry.website"
        >
          {{ entry.website }}
        </span>
      </div>
    </td>

    <td class="px-3 py-3">
      <span
        class="inline-flex items-center gap-1 rounded-md bg-emerald-50 px-1.5 py-0.5 text-[11px] font-black text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400"
      >
        <ShieldCheck class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
        {{ entry.login_provider }}
      </span>
    </td>

    <td class="truncate px-3 py-3">
      <span
        class="text-sm font-semibold text-slate-600 dark:text-slate-400"
        :title="entry.username"
      >
        {{ entry.username }}
      </span>
    </td>

    <td class="px-3 py-3">
      <WalletPasswordField :reveal="reveal" />
    </td>

    <td class="max-w-56 px-3 py-3">
      <span
        class="block truncate text-sm text-slate-500 transition-colors dark:text-slate-400"
        :title="entry.description"
      >
        {{ entry.description || '—' }}
      </span>
    </td>

    <td class="px-3 py-3">
      <span
        class="rounded-md bg-gray-100 px-2 py-1 text-[10px] font-bold whitespace-nowrap text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400"
      >
        {{ updatedLabel }}
      </span>
    </td>

    <td class="px-3 py-3 text-right">
      <div class="flex justify-end gap-0.5">
        <button
          type="button"
          title="Edit entry"
          aria-label="Edit wallet entry"
          class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-emerald-50 hover:text-emerald-700 dark:hover:bg-emerald-900/10 dark:hover:text-emerald-400"
          @click="$emit('edit', entry)"
        >
          <Pencil class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
        <button
          type="button"
          title="Delete entry"
          aria-label="Delete wallet entry"
          class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
          @click="$emit('delete', entry.id)"
        >
          <Trash2 class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
      </div>
    </td>
  </tr>
</template>

<script setup lang="ts">
import type { WalletEntry } from '../../../types';
import { computed } from 'vue';
import { Pencil, ShieldCheck, Trash2 } from '@lucide/vue';
import { formatDate } from '../../../lib/utils';
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

const updatedLabel = computed(() => formatDate(props.entry.updated_at || ''));
</script>
