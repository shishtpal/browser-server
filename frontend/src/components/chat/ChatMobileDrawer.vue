<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex lg:hidden">
      <div class="absolute inset-0 bg-slate-950/50 backdrop-blur-sm" @click="$emit('close')"></div>
      <aside
        class="relative z-10 flex h-full w-80 max-w-[85vw] flex-col bg-white dark:bg-slate-900"
      >
        <div
          class="flex items-center justify-between border-b border-slate-200 p-4 dark:border-white/10"
        >
          <h2 class="font-black">Conversations</h2>
          <button
            class="rounded-lg p-2 hover:bg-slate-100 dark:hover:bg-white/10"
            type="button"
            @click="$emit('close')"
          >
            ✕
          </button>
        </div>
        <div class="flex-1 space-y-1 overflow-y-auto p-3">
          <button
            class="mb-3 w-full rounded-lg bg-slate-900 px-3 py-2 text-sm font-bold text-white dark:bg-white dark:text-slate-900"
            :disabled="disabled"
            type="button"
            @click="$emit('new')"
          >
            + New Chat
          </button>

          <!-- Main conversations -->
          <div
            v-if="conversations.length === 0"
            class="px-2 py-4 text-center text-xs text-slate-400 dark:text-slate-500"
          >
            No conversations
          </div>
          <div
            v-for="conversation in conversations"
            :key="'m-' + conversation.id"
            class="cursor-pointer rounded-lg p-3 transition"
            :class="
              conversation.id === activeId
                ? 'bg-slate-100 dark:bg-white/10'
                : 'hover:bg-slate-50 dark:hover:bg-white/5'
            "
            @click="$emit('select', conversation.id)"
          >
            <span class="block truncate text-sm font-semibold">{{ conversation.title }}</span>
            <span class="block truncate text-xs text-slate-500">{{ conversation.model }}</span>
            <div class="relative mt-1 flex justify-end">
              <button
                class="rounded p-1 text-slate-400 hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-white"
                aria-label="Conversation actions"
                type="button"
                @click.stop="toggleMenu(conversation.id)"
              >
                <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    d="M10 4a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 7.5a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 7.5a1.5 1.5 0 110-3 1.5 1.5 0 010 3z"
                  />
                </svg>
              </button>
              <div
                v-if="openMenuId === conversation.id"
                class="absolute top-8 right-0 z-20 w-32 rounded-lg border border-slate-200 bg-white p-1 text-sm shadow-lg dark:border-white/10 dark:bg-slate-900"
              >
                <button
                  class="block w-full rounded px-2 py-1.5 text-left hover:bg-slate-100 dark:hover:bg-white/10"
                  type="button"
                  @click.stop="chooseAction('rename', conversation)"
                >
                  Edit
                </button>
                <button
                  class="block w-full rounded px-2 py-1.5 text-left text-amber-600 hover:bg-amber-50 dark:hover:bg-amber-900/20"
                  type="button"
                  @click.stop="chooseAction('archive', conversation)"
                >
                  Archive
                </button>
                <button
                  class="block w-full rounded px-2 py-1.5 text-left text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
                  type="button"
                  @click.stop="chooseAction('delete', conversation)"
                >
                  Delete
                </button>
              </div>
            </div>
          </div>

          <!-- Archived toggle -->
          <div class="mt-2 border-t border-slate-200 pt-2 dark:border-white/10">
            <button
              class="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-xs font-semibold text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-white"
              type="button"
              @click="$emit('toggle-archived')"
            >
              <svg
                class="h-3.5 w-3.5 transition-transform"
                :class="{ 'rotate-90': showArchived }"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M9 5l7 7-7 7"
                />
              </svg>
              Archived
              <span
                class="ml-auto rounded-full bg-slate-200 px-1.5 py-0.5 text-[10px] font-bold text-slate-600 dark:bg-white/10 dark:text-slate-300"
                >{{ archivedConversations.length }}</span
              >
            </button>
          </div>

          <!-- Archived list (expanded) -->
          <div v-if="showArchived && archivedConversations.length > 0" class="space-y-1">
            <div
              v-for="conversation in archivedConversations"
              :key="'m-archived-' + conversation.id"
              class="cursor-pointer rounded-lg border border-slate-200 bg-slate-50/50 p-3 transition dark:border-white/5 dark:bg-white/5"
              @click="$emit('select', conversation.id)"
            >
              <span
                class="block truncate text-sm font-semibold text-slate-400 dark:text-slate-500"
                >{{ conversation.title }}</span
              >
              <span class="block truncate text-xs text-slate-400 dark:text-slate-500">{{
                conversation.model
              }}</span>
              <div class="mt-1 flex gap-1">
                <button
                  class="rounded p-1 text-slate-400 hover:bg-green-100 hover:text-green-600 dark:hover:bg-green-900/30 dark:hover:text-green-400"
                  title="Restore"
                  type="button"
                  @click.stop="$emit('restore', conversation)"
                >
                  <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                    />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </aside>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { AIConversation } from '@browser-server/shared-types';
import { ref } from 'vue';

defineProps<{
  open: boolean;
  conversations: AIConversation[];
  activeId: string | null;
  disabled: boolean;
  archivedConversations: AIConversation[];
  showArchived: boolean;
}>();

const emit = defineEmits<{
  close: [];
  new: [];
  select: [id: string];
  rename: [conversation: AIConversation];
  delete: [conversation: AIConversation];
  archive: [conversation: AIConversation];
  restore: [conversation: AIConversation];
  'toggle-archived': [];
}>();

const openMenuId = ref<string | null>(null);

function toggleMenu(id: string) {
  openMenuId.value = openMenuId.value === id ? null : id;
}

function chooseAction(action: 'rename' | 'archive' | 'delete', conversation: AIConversation) {
  openMenuId.value = null;
  if (action === 'rename') emit('rename', conversation);
  else if (action === 'archive') emit('archive', conversation);
  else emit('delete', conversation);
}
</script>
