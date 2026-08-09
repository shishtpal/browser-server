<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex lg:hidden"
        role="dialog"
        aria-modal="true"
        aria-label="Conversations"
      >
        <div
          class="absolute inset-0 bg-slate-950/50 backdrop-blur-sm"
          @click="$emit('close')"
        ></div>
        <aside
          class="relative z-10 flex h-full w-80 max-w-[85vw] flex-col bg-white shadow-2xl dark:bg-slate-900"
        >
          <div
            class="flex items-center justify-between border-b border-slate-200 p-4 dark:border-white/10"
          >
            <h2 class="text-sm font-black">Conversations</h2>
            <button
              class="grid h-9 w-9 place-items-center rounded-lg text-slate-400 transition hover:bg-slate-100 dark:hover:bg-white/10"
              type="button"
              aria-label="Close conversation list"
              @click="$emit('close')"
            >
              <X class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
            </button>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain p-3">
            <Button
              variant="primary"
              class="mb-3 w-full"
              :disabled="disabled"
              @click="$emit('new')"
            >
              <span class="inline-flex items-center gap-1.5">
                <MessageSquarePlus class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
                New chat
              </span>
            </Button>

            <ConversationList
              :conversations="conversations"
              :active-id="activeId"
              :search="''"
              @select="$emit('select', $event)"
              @rename="$emit('rename', $event)"
              @delete="$emit('delete', $event)"
              @archive="$emit('archive', $event)"
            />

            <ArchivedConversationsList
              :items="archivedConversations"
              :open="showArchived"
              @select="$emit('select', $event)"
              @restore="$emit('restore', $event)"
              @toggle="$emit('toggle-archived')"
            />
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { MessageSquarePlus, X } from '@lucide/vue';
import type { AIConversation } from '@browser-server/shared-types';
import Button from '../ui/Button.vue';
import ArchivedConversationsList from './ArchivedConversationsList.vue';
import ConversationList from './ConversationList.vue';

defineProps<{
  open: boolean;
  conversations: AIConversation[];
  activeId: string | null;
  disabled: boolean;
  archivedConversations: AIConversation[];
  showArchived: boolean;
}>();

defineEmits<{
  close: [];
  new: [];
  select: [id: string];
  rename: [conversation: AIConversation];
  delete: [conversation: AIConversation];
  archive: [conversation: AIConversation];
  restore: [conversation: AIConversation];
  'toggle-archived': [];
}>();
</script>

<style scoped>
.drawer-enter-active,
.drawer-leave-active {
  transition: opacity 0.2s ease;
}
.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}
.drawer-enter-active aside,
.drawer-leave-active aside {
  transition: transform 0.2s ease;
}
.drawer-enter-from aside,
.drawer-leave-to aside {
  transform: translateX(-100%);
}
</style>
