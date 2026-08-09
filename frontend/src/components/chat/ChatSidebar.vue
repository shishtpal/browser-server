<template>
  <aside
    class="flex h-full min-h-0 shrink-0 flex-col overflow-hidden border-r border-slate-200 bg-slate-50 dark:border-white/10 dark:bg-slate-900/40"
  >
    <div
      class="flex items-center justify-between gap-3 border-b border-slate-200 px-3.5 py-3 dark:border-white/10"
    >
      <div class="min-w-0">
        <h1 class="flex items-center gap-1.5 text-[0.95rem] font-bold tracking-tight">
          <span
            class="grid h-5 w-5 place-items-center rounded-md bg-gradient-to-br from-indigo-500 to-violet-600 text-white"
          >
            <MessageCircle class="h-3 w-3" :stroke-width="2.25" aria-hidden="true" />
          </span>
          AI Chat
        </h1>
        <p class="mt-0.5 truncate text-[0.7rem] font-medium text-slate-400 dark:text-slate-500">
          {{ statusLabel }}
        </p>
      </div>
      <Button
        variant="primary"
        size="sm"
        class="!px-2.5 !py-1.5 text-[0.72rem]"
        :disabled="disabled"
        title="New conversation"
        @click="$emit('new')"
      >
        <span class="inline-flex items-center gap-1">
          <Plus class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          New
        </span>
      </Button>
    </div>

    <div class="px-3.5 pt-3">
      <div class="relative">
        <Search
          class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          :value="search"
          class="w-full rounded-lg border border-slate-200 bg-white py-1.5 pr-3 pl-8 text-[0.8rem] outline-none placeholder:text-slate-400 focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/15 dark:border-white/10 dark:bg-slate-950 dark:placeholder:text-slate-500"
          placeholder="Search conversations…"
          type="search"
          @input="$emit('update:search', ($event.target as HTMLInputElement).value)"
        />
      </div>
    </div>

    <div class="chat-scroll min-h-0 flex-1 overflow-y-auto px-2.5 py-3">
      <ConversationList
        :conversations="conversations"
        :active-id="activeId"
        :search="search"
        @select="$emit('select', $event)"
        @rename="$emit('rename', $event)"
        @delete="$emit('delete', $event)"
        @archive="$emit('archive', $event)"
      />
    </div>

    <ArchivedConversationsList
      :items="archivedConversations"
      :open="showArchived"
      @select="$emit('select', $event)"
      @restore="$emit('restore', $event)"
      @toggle="$emit('toggle-archived')"
    />
  </aside>
</template>

<script setup lang="ts">
import type { AIConversation } from '@browser-server/shared-types';
import { MessageCircle, Plus, Search } from '@lucide/vue';
import Button from '../ui/Button.vue';
import ArchivedConversationsList from './ArchivedConversationsList.vue';
import ConversationList from './ConversationList.vue';

defineProps<{
  conversations: AIConversation[];
  activeId: string | null;
  search: string;
  statusLabel: string;
  disabled: boolean;
  archivedConversations: AIConversation[];
  showArchived: boolean;
}>();

defineEmits<{
  new: [];
  select: [id: string];
  rename: [conversation: AIConversation];
  delete: [conversation: AIConversation];
  archive: [conversation: AIConversation];
  restore: [conversation: AIConversation];
  'toggle-archived': [];
  'update:search': [value: string];
}>();
</script>
