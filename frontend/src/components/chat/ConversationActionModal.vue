<template>
  <Modal :open="open" :title="meta.title" @close="$emit('cancel')">
    <form v-if="isRename" @submit.prevent="$emit('confirm')">
      <input
        v-model="model"
        class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
        placeholder="Conversation title"
        aria-label="Conversation title"
        autofocus
      />
    </form>
    <p v-else class="text-sm text-slate-600 dark:text-slate-400">
      {{ meta.messagePrefix }}<strong>{{ conversationTitle }}</strong
      >{{ meta.messageSuffix }}
    </p>

    <div class="mt-4 flex flex-col-reverse justify-end gap-2 sm:flex-row">
      <Button variant="ghost" size="sm" @click="$emit('cancel')">Cancel</Button>
      <button
        type="button"
        class="inline-flex items-center justify-center gap-1.5 rounded-lg px-4 py-2 text-sm font-bold transition"
        :class="meta.buttonClass"
        @click="$emit('confirm')"
      >
        <component :is="meta.icon" class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        {{ meta.confirmLabel }}
      </button>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Archive, ArchiveRestore, Check, Trash2, type LucideIcon } from '@lucide/vue';
import Button from '../ui/Button.vue';
import Modal from '../ui/Modal.vue';

export type ConversationActionKind = 'rename' | 'archive' | 'restore' | 'delete';

const props = defineProps<{
  open: boolean;
  kind: ConversationActionKind | null;
  /** Displayed conversation title ("rename" also edits it via v-model). */
  conversationTitle?: string;
}>();

const emit = defineEmits<{
  confirm: [];
  cancel: [];
}>();

/** Rename draft (used only when kind === 'rename'). */
const model = defineModel<string>({ default: '' });

const isRename = computed(() => props.kind === 'rename');

const META: Record<
  ConversationActionKind,
  {
    title: string;
    confirmLabel: string;
    icon: LucideIcon;
    buttonClass: string;
    messagePrefix: string;
    messageSuffix: string;
  }
> = {
  rename: {
    title: 'Rename conversation',
    confirmLabel: 'Save',
    icon: Check,
    buttonClass: 'bg-slate-900 text-white hover:bg-slate-800 dark:bg-white dark:text-slate-900',
    messagePrefix: 'Rename',
    messageSuffix: '',
  },
  archive: {
    title: 'Archive conversation',
    confirmLabel: 'Archive',
    icon: Archive,
    buttonClass: 'bg-amber-600 text-white hover:bg-amber-700',
    messagePrefix: 'Archive "',
    messageSuffix: '"? It will be moved to the Archived section.',
  },
  restore: {
    title: 'Restore conversation',
    confirmLabel: 'Restore',
    icon: ArchiveRestore,
    buttonClass: 'bg-emerald-600 text-white hover:bg-emerald-700',
    messagePrefix: 'Restore "',
    messageSuffix: '"? It will reappear in the main list.',
  },
  delete: {
    title: 'Delete conversation',
    confirmLabel: 'Delete',
    icon: Trash2,
    buttonClass: 'bg-red-600 text-white hover:bg-red-700',
    messagePrefix: 'Delete "',
    messageSuffix: '"? This action cannot be undone.',
  },
};

const meta = computed(() => META[props.kind ?? 'rename']);
</script>
