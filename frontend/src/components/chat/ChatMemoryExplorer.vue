<template>
  <Modal :open="open" title="Memory Explorer" description="View, edit, or remove messages from this conversation's memory." fullscreen @close="$emit('close')">
    <div class="flex h-full flex-col overflow-hidden">
      <!-- Toolbar: bulk remove tool calls -->
      <div v-if="toolMessageCount > 0" class="mb-3 flex shrink-0 items-center gap-3">
        <button
          v-if="!confirmRemoveAllTools"
          class="flex items-center gap-1.5 rounded-lg border border-amber-200 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700 transition hover:bg-amber-100 dark:border-amber-800/50 dark:bg-amber-950/30 dark:text-amber-300 dark:hover:bg-amber-950/50"
          type="button"
          :disabled="saving"
          @click="confirmRemoveAllTools = true"
        >
          <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
          Remove all tool calls ({{ toolMessageCount }})
        </button>
        <div v-else class="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-1.5 dark:border-red-900/40 dark:bg-red-950/20">
          <span class="text-xs text-red-700 dark:text-red-300">Remove all {{ toolMessageCount }} tool messages?</span>
          <Button variant="danger" size="sm" :loading="saving" loading-text="…" @click="doRemoveAllTools">Confirm</Button>
          <Button variant="ghost" size="sm" @click="confirmRemoveAllTools = false">Cancel</Button>
        </div>
      </div>

      <!-- Message list -->
      <div class="flex-1 space-y-2 overflow-y-auto pr-1">
        <EmptyState
          v-if="messages.length === 0"
          title="No messages"
          description="This conversation has no messages yet."
          icon="default"
          color="indigo"
        />

        <MemoryMessageCard
          v-for="msg in messages"
          :key="msg.id"
          :message="msg"
          :editing="editingId === msg.id"
          :edit-content="editContent"
          :is-delete-target="deleteTargetId === msg.id"
          :saving="saving"
          @edit="startEdit(msg)"
          @edit-input="editContent = $event"
          @save="saveEdit(msg)"
          @cancel-edit="cancelEdit"
          @confirm-delete="confirmDeleteMessage(msg)"
          @delete="doDelete(msg)"
          @cancel-delete="cancelDelete"
        />
      </div>

      <!-- Error banner -->
      <ErrorBanner v-if="errorMsg" :message="errorMsg" class="mt-3 shrink-0" />
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { computed, toRef } from 'vue'
import type { AIMessage } from '@browser-server/shared-types'
import Modal from '../ui/Modal.vue'
import Button from '../ui/Button.vue'
import EmptyState from '../ui/EmptyState.vue'
import ErrorBanner from '../ui/ErrorBanner.vue'
import MemoryMessageCard from './memory/MemoryMessageCard.vue'
import { useMemoryExplorer } from './composables/useMemoryExplorer'

const props = defineProps<{
  open: boolean
  conversationId: string
  messages: AIMessage[]
}>()

const emit = defineEmits<{
  close: []
  updated: [messages: AIMessage[]]
}>()

const {
  editingId,
  editContent,
  deleteTargetId,
  confirmRemoveAllTools,
  saving,
  errorMsg,
  toolMessageCount,
  startEdit,
  cancelEdit,
  saveEdit,
  confirmDeleteMessage,
  cancelDelete,
  doDelete,
  doRemoveAllTools,
} = useMemoryExplorer(
  toRef(props, 'open'),
  toRef(props, 'conversationId'),
  toRef(props, 'messages'),
  (msgs) => emit('updated', msgs),
)
</script>
