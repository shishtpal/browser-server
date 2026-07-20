<template>
  <Modal :open="open" title="Memory Explorer" description="View, edit, or remove messages from this conversation's memory." fullscreen @close="$emit('close')">
    <div class="flex h-full flex-col overflow-hidden">
      <!-- Toolbar -->
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
          <button
            class="rounded bg-red-600 px-2.5 py-1 text-[11px] font-bold text-white hover:bg-red-700"
            type="button"
            :disabled="saving"
            @click="doRemoveAllTools"
          >{{ saving ? '…' : 'Confirm' }}</button>
          <button
            class="rounded px-2.5 py-1 text-[11px] text-slate-500 hover:bg-white dark:hover:bg-white/10"
            type="button"
            @click="confirmRemoveAllTools = false"
          >Cancel</button>
        </div>
      </div>

      <div class="flex-1 space-y-2 overflow-y-auto pr-1">
        <div v-if="messages.length === 0" class="py-8 text-center text-sm text-slate-400">
          No messages in this conversation.
        </div>

        <div
          v-for="(msg, index) in messages"
          :key="msg.id"
          class="group relative rounded-lg border px-3 py-2 transition"
          :class="messageBorderClass(msg)"
        >
          <!-- Role badge + timestamp -->
          <div class="mb-1 flex items-center gap-2">
            <span
              class="rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide"
              :class="roleBadgeClass(msg.role)"
            >{{ msg.role }}</span>
            <span v-if="msg.role === 'tool'" class="text-[10px] font-medium text-slate-600 dark:text-slate-300">{{ getToolName(msg) }}</span>
            <span class="text-[10px] text-slate-400">{{ formatTime(msg.created_at) }}</span>
            <span v-if="msg.status !== 'completed'" class="text-[10px] italic text-slate-400">({{ msg.status }})</span>
          </div>

          <!-- Content display or edit -->
          <div v-if="editingId === msg.id" class="space-y-2">
            <textarea
              ref="editTextarea"
              v-model="editContent"
              class="w-full rounded border border-slate-300 bg-white px-2 py-1.5 font-mono text-xs leading-relaxed outline-none focus:border-slate-500 dark:border-white/20 dark:bg-slate-800 dark:text-white"
              rows="6"
            ></textarea>
            <div class="flex gap-2">
              <button
                class="rounded bg-slate-900 px-3 py-1 text-xs font-bold text-white hover:bg-slate-700 dark:bg-white dark:text-slate-900 dark:hover:bg-slate-200"
                type="button"
                :disabled="saving"
                @click="saveEdit(msg)"
              >{{ saving ? 'Saving…' : 'Save' }}</button>
              <button
                class="rounded px-3 py-1 text-xs text-slate-500 hover:bg-slate-100 dark:hover:bg-white/10"
                type="button"
                @click="cancelEdit"
              >Cancel</button>
            </div>
          </div>

          <!-- Tool message display -->
          <div v-else-if="msg.role === 'tool'" class="space-y-1.5">
            <div v-if="getToolArgs(msg)" class="overflow-hidden rounded border border-slate-200 dark:border-white/10">
              <header class="flex h-6 items-center border-b border-slate-200 bg-slate-50 px-2 dark:border-white/10 dark:bg-slate-800/60">
                <span class="text-[9px] font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Arguments</span>
              </header>
              <pre class="max-h-24 overflow-auto whitespace-pre-wrap break-words px-2 py-1.5 font-mono text-[11px] leading-snug text-slate-700 dark:text-slate-300">{{ getToolArgs(msg) }}</pre>
            </div>
            <div v-if="getToolResult(msg)" class="overflow-hidden rounded border border-slate-200 dark:border-white/10">
              <header class="flex h-6 items-center border-b border-slate-200 bg-slate-50 px-2 dark:border-white/10 dark:bg-slate-800/60">
                <span class="text-[9px] font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Result</span>
              </header>
              <pre class="max-h-32 overflow-auto whitespace-pre-wrap break-words px-2 py-1.5 font-mono text-[11px] leading-snug text-slate-700 dark:text-slate-300">{{ getToolResult(msg) }}</pre>
            </div>
            <div v-if="getToolDecision(msg)" class="text-[10px] text-slate-500 dark:text-slate-400">
              Decision: <span class="font-medium" :class="getToolDecision(msg) === 'approved' ? 'text-emerald-600 dark:text-emerald-400' : getToolDecision(msg) === 'rejected' ? 'text-red-600 dark:text-red-400' : 'text-amber-600 dark:text-amber-400'">{{ getToolDecision(msg) }}</span>
            </div>
          </div>

          <!-- Regular message display -->
          <div v-else>
            <pre class="max-h-40 overflow-auto whitespace-pre-wrap break-words text-xs leading-relaxed text-slate-700 dark:text-slate-300">{{ truncateContent(msg.content) }}</pre>
          </div>

          <!-- Action buttons (shown on hover) -->
          <div class="absolute right-2 top-2 hidden gap-1 group-hover:flex">
            <button
              class="rounded p-1 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-white"
              title="Edit message"
              type="button"
              @click="startEdit(msg)"
            >
              <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
            </button>
            <button
              v-if="deleteTargetId !== msg.id"
              class="rounded p-1 text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/30 dark:hover:text-red-400"
              title="Delete message"
              type="button"
              @click="confirmDeleteMessage(msg)"
            >
              <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
            </button>
          </div>

          <!-- Inline delete confirmation (appears within the same message card) -->
          <div v-if="deleteTargetId === msg.id" class="mt-2 flex items-center gap-2 rounded border border-red-200 bg-red-50 px-2.5 py-1.5 dark:border-red-900/40 dark:bg-red-950/20">
            <span class="flex-1 text-[11px] text-red-700 dark:text-red-300">Delete this {{ msg.role }} message?</span>
            <button
              class="rounded bg-red-600 px-2.5 py-1 text-[11px] font-bold text-white hover:bg-red-700"
              type="button"
              :disabled="saving"
              @click="doDelete(msg)"
            >{{ saving ? '…' : 'Delete' }}</button>
            <button
              class="rounded px-2.5 py-1 text-[11px] text-slate-500 hover:bg-white dark:hover:bg-white/10"
              type="button"
              @click="deleteTargetId = null"
            >Cancel</button>
          </div>
        </div>
      </div>

      <!-- Error banner -->
      <p v-if="errorMsg" class="mt-3 shrink-0 text-xs text-red-500">{{ errorMsg }}</p>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import type { AIMessage } from '@browser-server/shared-types'
import Modal from '../ui/Modal.vue'
import { updateAIMessage, deleteAIMessage } from '../../lib/api'

const props = defineProps<{
  open: boolean
  conversationId: string
  messages: AIMessage[]
}>()

const emit = defineEmits<{
  close: []
  updated: [messages: AIMessage[]]
}>()

const editingId = ref<string | null>(null)
const editContent = ref('')
const deleteTargetId = ref<string | null>(null)
const confirmRemoveAllTools = ref(false)
const saving = ref(false)
const errorMsg = ref('')

const toolMessageCount = computed(() => props.messages.filter((m) => m.role === 'tool').length)

function roleBadgeClass(role: string) {
  switch (role) {
    case 'user': return 'bg-slate-900 text-white dark:bg-white dark:text-slate-900'
    case 'assistant': return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
    case 'tool': return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
    case 'system': return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
    default: return 'bg-slate-100 text-slate-600'
  }
}

function messageBorderClass(msg: AIMessage) {
  if (deleteTargetId.value === msg.id) return 'border-red-300 bg-red-50/80 dark:border-red-800/60 dark:bg-red-950/20'
  if (msg.status === 'error') return 'border-red-200 bg-red-50/50 dark:border-red-900/30 dark:bg-red-950/10'
  if (msg.status === 'superseded') return 'border-slate-200 bg-slate-50/50 opacity-50 dark:border-white/5 dark:bg-slate-900/30'
  return 'border-slate-200 bg-white dark:border-white/10 dark:bg-slate-900'
}

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  } catch { return iso }
}

function truncateContent(content: string) {
  if (content.length <= 500) return content
  return content.slice(0, 500) + '…'
}

// ─── Tool message helpers ──────────────────────────────

function parseToolContent(msg: AIMessage): { tool?: string; args?: unknown; result?: unknown; decision?: string } {
  try {
    return JSON.parse(msg.content)
  } catch {
    return {}
  }
}

function getToolName(msg: AIMessage): string {
  const parsed = parseToolContent(msg)
  if (!parsed.tool) return ''
  return parsed.tool.split('_').filter(Boolean).map((w) => w[0].toUpperCase() + w.slice(1)).join(' ')
}

function getToolArgs(msg: AIMessage): string {
  const parsed = parseToolContent(msg)
  if (parsed.args === null || parsed.args === undefined) return ''
  if (typeof parsed.args === 'string') return parsed.args
  try { return JSON.stringify(parsed.args, null, 2) } catch { return String(parsed.args) }
}

function getToolResult(msg: AIMessage): string {
  const parsed = parseToolContent(msg)
  if (parsed.result === null || parsed.result === undefined) return ''
  if (typeof parsed.result === 'string') return parsed.result
  try { return JSON.stringify(parsed.result, null, 2) } catch { return String(parsed.result) }
}

function getToolDecision(msg: AIMessage): string {
  return parseToolContent(msg).decision || ''
}

// ─── Edit actions ──────────────────────────────────────

function startEdit(msg: AIMessage) {
  editingId.value = msg.id
  editContent.value = msg.content
  errorMsg.value = ''
  deleteTargetId.value = null
  nextTick(() => {
    // Textarea will auto-focus via v-model reactivity
  })
}

function cancelEdit() {
  editingId.value = null
  editContent.value = ''
}

async function saveEdit(msg: AIMessage) {
  if (!props.conversationId) return
  saving.value = true
  errorMsg.value = ''
  try {
    const updated = await updateAIMessage(props.conversationId, msg.id, { content: editContent.value })
    const newMessages = props.messages.map((m) => m.id === updated.id ? updated : m)
    emit('updated', newMessages)
    editingId.value = null
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : 'Failed to update message'
  } finally {
    saving.value = false
  }
}

// ─── Delete actions ────────────────────────────────────

function confirmDeleteMessage(msg: AIMessage) {
  deleteTargetId.value = msg.id
  errorMsg.value = ''
  editingId.value = null
}

async function doDelete(msg: AIMessage) {
  if (!props.conversationId) return
  saving.value = true
  errorMsg.value = ''
  try {
    await deleteAIMessage(props.conversationId, msg.id)
    const newMessages = props.messages.filter((m) => m.id !== msg.id)
    emit('updated', newMessages)
    deleteTargetId.value = null
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : 'Failed to delete message'
  } finally {
    saving.value = false
  }
}

async function doRemoveAllTools() {
  if (!props.conversationId) return
  saving.value = true
  errorMsg.value = ''
  const toolMessages = props.messages.filter((m) => m.role === 'tool')
  try {
    for (const msg of toolMessages) {
      await deleteAIMessage(props.conversationId, msg.id)
    }
    const newMessages = props.messages.filter((m) => m.role !== 'tool')
    emit('updated', newMessages)
    confirmRemoveAllTools.value = false
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : 'Failed to remove tool messages'
  } finally {
    saving.value = false
  }
}
</script>
