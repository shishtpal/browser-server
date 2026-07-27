<template>
  <aside class="flex h-full min-h-0 flex-col border-r border-slate-200 bg-slate-50/80 dark:border-white/10 dark:bg-slate-900/60">
    <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4 dark:border-white/10">
      <div>
        <h1 class="text-lg font-black">AI Chat</h1>
        <p class="text-xs text-slate-500 dark:text-slate-400">{{ statusLabel }}</p>
      </div>
      <button
        class="rounded-lg bg-slate-900 px-3 py-2 text-xs font-bold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-white dark:text-slate-900 dark:hover:bg-gray-100"
        :disabled="disabled"
        type="button"
        title="New conversation"
        @click="$emit('new')"
      >
        + New
      </button>
    </div>

    <div class="px-4 pt-3">
      <input
        :value="search"
        class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm outline-none placeholder:text-slate-400 focus:border-slate-400 dark:border-white/10 dark:bg-slate-950 dark:placeholder:text-slate-500"
        placeholder="Search conversations…"
        type="search"
        @input="$emit('update:search', ($event.target as HTMLInputElement).value)"
      />
    </div>

    <!-- Main conversations -->
    <div class="min-h-0 flex-1 space-y-1 overflow-y-auto px-3 py-3">
      <div v-if="conversations.length === 0" class="px-2 py-6 text-center text-xs text-slate-400 dark:text-slate-500">
        {{ search ? 'No matching conversations' : 'No conversations yet' }}
      </div>
      <div
        v-for="conversation in conversations"
        :key="conversation.id"
        class="group relative rounded-lg border p-3 transition"
        :class="conversation.id === activeId
          ? 'border-slate-900 bg-white shadow-sm dark:border-white/20 dark:bg-white/10'
          : 'cursor-pointer border-transparent hover:border-slate-200 hover:bg-white dark:hover:border-white/10 dark:hover:bg-white/5'"
        @click="$emit('select', conversation.id)"
      >
        <span class="block truncate text-sm font-semibold">{{ conversation.title }}</span>
        <span class="mt-0.5 block truncate text-xs text-slate-500 dark:text-slate-400">
          {{ conversation.model }} · {{ formatRelativeTime(conversation.updated_at) }}
        </span>
        <span
          v-if="conversation.profile"
          class="mt-1 inline-block rounded-full bg-indigo-100 px-2 py-0.5 text-[10px] font-semibold text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300"
        >{{ conversation.profile }}</span>
        <div class="absolute right-2 top-2">
          <button
            class="rounded p-1 text-slate-400 hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-white"
            aria-label="Conversation actions"
            type="button"
            @click.stop="toggleMenu(conversation.id)"
          >
            <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20"><path d="M10 4a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 7.5a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 7.5a1.5 1.5 0 110-3 1.5 1.5 0 010 3z"/></svg>
          </button>
          <div v-if="openMenuId === conversation.id" class="absolute right-0 top-8 z-20 w-32 rounded-lg border border-slate-200 bg-white p-1 text-sm shadow-lg dark:border-white/10 dark:bg-slate-900">
            <button class="block w-full rounded px-2 py-1.5 text-left hover:bg-slate-100 dark:hover:bg-white/10" type="button" @click.stop="chooseAction('rename', conversation)">Edit</button>
            <button class="block w-full rounded px-2 py-1.5 text-left text-amber-600 hover:bg-amber-50 dark:hover:bg-amber-900/20" type="button" @click.stop="chooseAction('archive', conversation)">Archive</button>
            <button class="block w-full rounded px-2 py-1.5 text-left text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20" type="button" @click.stop="chooseAction('delete', conversation)">Delete</button>
          </div>
        </div>
      </div>
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
import { ref } from 'vue'
import type { AIConversation } from '@browser-server/shared-types'
import ArchivedConversationsList from './ArchivedConversationsList.vue'

defineProps<{
  conversations: AIConversation[]
  activeId: string | null
  search: string
  statusLabel: string
  disabled: boolean
  archivedConversations: AIConversation[]
  showArchived: boolean
}>()

const emit = defineEmits<{
  new: []
  select: [id: string]
  rename: [conversation: AIConversation]
  delete: [conversation: AIConversation]
  archive: [conversation: AIConversation]
  restore: [conversation: AIConversation]
  'toggle-archived': []
  'update:search': [value: string]
}>()

const openMenuId = ref<string | null>(null)

function toggleMenu(id: string) {
  openMenuId.value = openMenuId.value === id ? null : id
}

function chooseAction(action: 'rename' | 'archive' | 'delete', conversation: AIConversation) {
  openMenuId.value = null
  emit(action, conversation)
}

function formatRelativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) return 'just now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  return new Date(iso).toLocaleDateString()
}
</script>
