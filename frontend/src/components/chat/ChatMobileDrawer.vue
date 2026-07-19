<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex lg:hidden">
      <div class="absolute inset-0 bg-slate-950/50 backdrop-blur-sm" @click="$emit('close')"></div>
      <aside class="relative z-10 flex h-full w-80 max-w-[85vw] flex-col bg-white dark:bg-slate-900">
        <div class="flex items-center justify-between border-b border-slate-200 p-4 dark:border-white/10">
          <h2 class="font-black">Conversations</h2>
          <button class="rounded-lg p-2 hover:bg-slate-100 dark:hover:bg-white/10" type="button" @click="$emit('close')">✕</button>
        </div>
        <div class="flex-1 space-y-1 overflow-y-auto p-3">
          <button
            class="mb-3 w-full rounded-lg bg-slate-900 px-3 py-2 text-sm font-bold text-white dark:bg-white dark:text-slate-900"
            :disabled="disabled"
            type="button"
            @click="$emit('new')"
          >+ New Chat</button>
          <div
            v-for="conversation in conversations"
            :key="'m-' + conversation.id"
            class="cursor-pointer rounded-lg p-3 transition"
            :class="conversation.id === activeId ? 'bg-slate-100 dark:bg-white/10' : 'hover:bg-slate-50 dark:hover:bg-white/5'"
            @click="$emit('select', conversation.id)"
          >
            <span class="block truncate text-sm font-semibold">{{ conversation.title }}</span>
            <span class="block truncate text-xs text-slate-500">{{ conversation.model }}</span>
          </div>
        </div>
      </aside>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { AIConversation } from '@browser-server/shared-types'

defineProps<{
  open: boolean
  conversations: AIConversation[]
  activeId: string | null
  disabled: boolean
}>()

defineEmits<{
  close: []
  new: []
  select: [id: string]
}>()
</script>
