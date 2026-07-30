<template>
  <aside
    class="flex h-full min-h-0 flex-col border-r border-slate-200 bg-slate-50 dark:border-white/10 dark:bg-slate-900/40">
    <div class="flex items-center justify-between gap-3 border-b border-slate-200 px-3.5 py-3 dark:border-white/10">
      <div class="min-w-0">
        <h1 class="flex items-center gap-1.5 text-[0.95rem] font-bold tracking-tight">
          <span
            class="grid h-5 w-5 place-items-center rounded-md bg-gradient-to-br from-indigo-500 to-violet-600 text-white">
            <svg class="h-3 w-3" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round"
                d="M8 10h.01M12 10h.01M16 10h.01M21 12a9 9 0 11-4.5-7.79" />
            </svg>
          </span>
          AI Chat
        </h1>
        <p class="mt-0.5 truncate text-[0.7rem] font-medium text-slate-400 dark:text-slate-500">{{ statusLabel }}</p>
      </div>
      <button
        class="inline-flex items-center gap-1 rounded-lg bg-slate-900 px-2.5 py-1.5 text-[0.72rem] font-semibold text-white shadow-sm transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-white dark:text-slate-900 dark:hover:bg-slate-100"
        :disabled="disabled" type="button" title="New conversation" @click="$emit('new')">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2.2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 5v14m-7-7h14" />
        </svg>
        New
      </button>
    </div>

    <div class="px-3.5 pt-3">
      <div class="relative">
        <svg class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-4.35-4.35M17 11a6 6 0 11-12 0 6 6 0 0112 0z" />
        </svg>
        <input :value="search"
          class="w-full rounded-lg border border-slate-200 bg-white py-1.5 pl-8 pr-3 text-[0.8rem] outline-none placeholder:text-slate-400 focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/15 dark:border-white/10 dark:bg-slate-950 dark:placeholder:text-slate-500"
          placeholder="Search conversations…" type="search"
          @input="$emit('update:search', ($event.target as HTMLInputElement).value)" />
      </div>
    </div>

    <!-- Main conversations -->
    <div class="chat-scroll min-h-0 flex-1 space-y-0.5 overflow-y-auto px-2.5 py-3">
      <div v-if="conversations.length === 0"
        class="px-2 py-8 text-center text-[0.75rem] text-slate-400 dark:text-slate-500">
        {{ search ? 'No matching conversations' : 'No conversations yet' }}
      </div>
      <div v-for="conversation in conversations" :key="conversation.id"
        class="group relative rounded-lg border px-2.5 py-2 transition" :class="conversation.id === activeId
          ? 'border-indigo-200 bg-white shadow-sm ring-1 ring-indigo-500/10 dark:border-indigo-500/30 dark:bg-white/10 dark:ring-indigo-400/10'
          : 'cursor-pointer border-transparent hover:bg-white dark:hover:bg-white/5'"
        @click="$emit('select', conversation.id)">
        <span class="block truncate pr-6 text-[0.82rem] font-semibold leading-tight"
          :class="conversation.id === activeId ? 'text-slate-900 dark:text-white' : 'text-slate-700 dark:text-slate-200'">{{
            conversation.title }}</span>
        <span class="mt-1 block truncate font-mono text-[0.66rem] text-slate-400 dark:text-slate-500">
          {{ conversation.model }} · {{ formatRelativeTime(conversation.updated_at) }}
        </span>
        <span v-if="conversation.profile"
          class="mt-1.5 inline-block rounded-full bg-indigo-50 px-1.5 py-0.5 text-[0.6rem] font-semibold uppercase tracking-wide text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300">{{
            conversation.profile }}</span>
        <div class="absolute right-1.5 top-1.5">
          <button
            class="rounded-md p-1 text-slate-400 opacity-0 transition hover:bg-slate-200 hover:text-slate-700 group-hover:opacity-100 dark:hover:bg-white/10 dark:hover:text-white"
            :class="openMenuId === conversation.id ? 'opacity-100' : ''" aria-label="Conversation actions" type="button"
            @click.stop="toggleMenu(conversation.id)">
            <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
              <path
                d="M10 4a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 7.5a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 7.5a1.5 1.5 0 110-3 1.5 1.5 0 010 3z" />
            </svg>
          </button>
          <div v-if="openMenuId === conversation.id"
            class="absolute right-0 top-8 z-20 w-32 rounded-lg border border-slate-200 bg-white p-1 text-[0.8rem] shadow-lg dark:border-white/10 dark:bg-slate-900">
            <button
              class="block w-full rounded px-2 py-1.5 text-left font-medium hover:bg-slate-100 dark:hover:bg-white/10"
              type="button" @click.stop="chooseAction('rename', conversation)">Rename</button>
            <button
              class="block w-full rounded px-2 py-1.5 text-left font-medium text-amber-600 hover:bg-amber-50 dark:hover:bg-amber-900/20"
              type="button" @click.stop="chooseAction('archive', conversation)">Archive</button>
            <button
              class="block w-full rounded px-2 py-1.5 text-left font-medium text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
              type="button" @click.stop="chooseAction('delete', conversation)">Delete</button>
          </div>
        </div>
      </div>
    </div>

    <ArchivedConversationsList :items="archivedConversations" :open="showArchived" @select="$emit('select', $event)"
      @restore="$emit('restore', $event)" @toggle="$emit('toggle-archived')" />
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
  if (action === 'rename')
    emit('rename', conversation)
  else if (action === 'archive')
    emit('archive', conversation)
  else
    emit('delete', conversation)
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
