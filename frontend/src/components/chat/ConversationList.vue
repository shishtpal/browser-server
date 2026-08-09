<template>
  <div ref="rootEl" class="space-y-0.5">
    <div
      v-if="conversations.length === 0"
      class="px-2 py-8 text-center text-[0.75rem] text-slate-400 dark:text-slate-500"
    >
      {{ search ? 'No matching conversations' : 'No conversations yet' }}
    </div>

    <div
      v-for="conversation in conversations"
      :key="conversation.id"
      class="group relative rounded-lg border px-2.5 py-2 transition"
      role="button"
      tabindex="0"
      :aria-current="conversation.id === activeId ? 'true' : undefined"
      :class="
        conversation.id === activeId
          ? 'border-indigo-200 bg-white shadow-sm ring-1 ring-indigo-500/10 dark:border-indigo-500/30 dark:bg-white/10 dark:ring-indigo-400/10'
          : 'cursor-pointer border-transparent hover:bg-white dark:hover:bg-white/5'
      "
      @click="$emit('select', conversation.id)"
      @keydown.enter="$emit('select', conversation.id)"
    >
      <span
        class="block truncate pr-6 text-[0.82rem] leading-tight font-semibold"
        :class="
          conversation.id === activeId
            ? 'text-slate-900 dark:text-white'
            : 'text-slate-700 dark:text-slate-200'
        "
      >
        {{ conversation.title }}
      </span>
      <span class="mt-1 block truncate font-mono text-[0.66rem] text-slate-400 dark:text-slate-500">
        {{ conversation.model }} · {{ formatRelativeTime(conversation.updated_at) }}
      </span>
      <span
        v-if="conversation.profile"
        class="mt-1.5 inline-block rounded-full bg-indigo-50 px-1.5 py-0.5 text-[0.6rem] font-semibold tracking-wide text-indigo-600 uppercase dark:bg-indigo-900/30 dark:text-indigo-300"
      >
        {{ conversation.profile }}
      </span>

      <!-- Overflow menu -->
      <div class="absolute top-1.5 right-1.5">
        <button
          class="grid h-7 w-7 place-items-center rounded-md text-slate-400 transition group-hover:opacity-100 hover:bg-slate-200 hover:text-slate-700 sm:opacity-0 dark:hover:bg-white/10 dark:hover:text-white"
          :class="{ 'opacity-100': openMenuId === conversation.id }"
          :aria-label="`Actions for ${conversation.title}`"
          :aria-expanded="openMenuId === conversation.id"
          type="button"
          @click.stop="toggleMenu(conversation.id)"
        >
          <EllipsisVertical class="h-4 w-4" aria-hidden="true" />
        </button>
        <div
          v-if="openMenuId === conversation.id"
          class="absolute top-8 right-0 z-20 w-32 overflow-hidden rounded-lg border border-slate-200 bg-white p-1 text-[0.8rem] shadow-lg dark:border-white/10 dark:bg-slate-900"
          role="menu"
        >
          <button
            class="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left font-medium hover:bg-slate-100 dark:hover:bg-white/10"
            type="button"
            role="menuitem"
            @click.stop="choose('rename', conversation)"
          >
            <Pencil class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
            Rename
          </button>
          <button
            class="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left font-medium text-amber-600 hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/20"
            type="button"
            role="menuitem"
            @click.stop="choose('archive', conversation)"
          >
            <Archive class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
            Archive
          </button>
          <button
            class="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left font-medium text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
            type="button"
            role="menuitem"
            @click.stop="choose('delete', conversation)"
          >
            <Trash2 class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
            Delete
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue';
import { Archive, EllipsisVertical, Pencil, Trash2 } from '@lucide/vue';
import type { AIConversation } from '@browser-server/shared-types';
import { formatRelativeTime } from './chatFormat';

defineProps<{
  conversations: AIConversation[];
  activeId: string | null;
  search: string;
}>();

const emit = defineEmits<{
  select: [id: string];
  rename: [conversation: AIConversation];
  delete: [conversation: AIConversation];
  archive: [conversation: AIConversation];
}>();

const rootEl = ref<HTMLElement | null>(null);
const openMenuId = ref<string | null>(null);

function toggleMenu(id: string) {
  openMenuId.value = openMenuId.value === id ? null : id;
}

function choose(action: 'rename' | 'archive' | 'delete', conversation: AIConversation) {
  openMenuId.value = null;
  (emit as any)(action, conversation);
}

/** The floating action menu closes when clicking anywhere outside it. */
function onDocumentClick(event: MouseEvent) {
  if (!openMenuId.value) return;
  if (rootEl.value?.contains(event.target as Node)) return;
  openMenuId.value = null;
}

if (typeof document !== 'undefined') {
  document.addEventListener('click', onDocumentClick);
  onBeforeUnmount(() => document.removeEventListener('click', onDocumentClick));
}
</script>
