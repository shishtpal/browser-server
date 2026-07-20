<template>
  <aside class="flex flex-col border-r border-slate-200 bg-slate-50/80 dark:border-white/10 dark:bg-slate-900/60">
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

    <div class="flex-1 space-y-1 overflow-y-auto px-3 py-3">
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
        <div class="absolute right-2 top-2 hidden gap-1 group-hover:flex">
          <button
            class="rounded p-1 text-slate-400 hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-white"
            title="Rename"
            type="button"
            @click.stop="$emit('rename', conversation)"
          >
            <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
          </button>
          <button
            class="rounded p-1 text-slate-400 hover:bg-red-100 hover:text-red-600 dark:hover:bg-red-900/30 dark:hover:text-red-400"
            title="Delete"
            type="button"
            @click.stop="$emit('delete', conversation)"
          >
            <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
          </button>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import type { AIConversation } from '@browser-server/shared-types'

defineProps<{
  conversations: AIConversation[]
  activeId: string | null
  search: string
  statusLabel: string
  disabled: boolean
}>()

defineEmits<{
  new: []
  select: [id: string]
  rename: [conversation: AIConversation]
  delete: [conversation: AIConversation]
  'update:search': [value: string]
}>()

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
