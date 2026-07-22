import type { AIMessage } from '@browser-server/shared-types'
import { computed, ref, watch, type Ref } from 'vue'
import { getAIConversation, updateAIMessage, deleteAIMessage } from '../../../lib/api'

export function useMemoryExplorer(
  open: Ref<boolean>,
  conversationId: Ref<string>,
  messages: Ref<AIMessage[]>,
  onUpdated: (msgs: AIMessage[]) => void,
) {
  const editingId = ref<string | null>(null)
  const editContent = ref('')
  const deleteTargetId = ref<string | null>(null)
  const confirmRemoveAllTools = ref(false)
  const saving = ref(false)
  const errorMsg = ref('')

  const toolMessageCount = computed(() => messages.value.filter((m) => m.role === 'tool').length)

  // When the modal opens, re-fetch messages from the backend so we always
  // have real persisted IDs (streaming creates temp frontend-only IDs that
  // the backend won't recognise until a full conversation reload).
  watch(open, async (isOpen) => {
    if (isOpen && conversationId.value) {
      errorMsg.value = ''
      editingId.value = null
      deleteTargetId.value = null
      confirmRemoveAllTools.value = false
      try {
        const detail = await getAIConversation(conversationId.value)
        if (detail.messages) {
          onUpdated(detail.messages)
        }
      } catch { /* fall through with existing props */ }
    }
  })

  // ─── Edit actions ──────────────────────────────────────

  function startEdit(msg: AIMessage) {
    editingId.value = msg.id
    editContent.value = msg.content
    errorMsg.value = ''
    deleteTargetId.value = null
  }

  function cancelEdit() {
    editingId.value = null
    editContent.value = ''
  }

  async function saveEdit(msg: AIMessage) {
    if (!conversationId.value) return
    saving.value = true
    errorMsg.value = ''
    try {
      const updated = await updateAIMessage(conversationId.value, msg.id, { content: editContent.value })
      const newMessages = messages.value.map((m) => m.id === updated.id ? updated : m)
      onUpdated(newMessages)
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

  function cancelDelete() {
    deleteTargetId.value = null
  }

  async function doDelete(msg: AIMessage) {
    if (!conversationId.value) return
    saving.value = true
    errorMsg.value = ''
    try {
      await deleteAIMessage(conversationId.value, msg.id)
      const newMessages = messages.value.filter((m) => m.id !== msg.id)
      onUpdated(newMessages)
      deleteTargetId.value = null
    } catch (err) {
      errorMsg.value = err instanceof Error ? err.message : 'Failed to delete message'
    } finally {
      saving.value = false
    }
  }

  async function doRemoveAllTools() {
    if (!conversationId.value) return
    saving.value = true
    errorMsg.value = ''
    const toolMessages = messages.value.filter((m) => m.role === 'tool')
    try {
      for (const msg of toolMessages) {
        await deleteAIMessage(conversationId.value, msg.id)
      }
      const newMessages = messages.value.filter((m) => m.role !== 'tool')
      onUpdated(newMessages)
      confirmRemoveAllTools.value = false
    } catch (err) {
      errorMsg.value = err instanceof Error ? err.message : 'Failed to remove tool messages'
    } finally {
      saving.value = false
    }
  }

  return {
    // State
    editingId,
    editContent,
    deleteTargetId,
    confirmRemoveAllTools,
    saving,
    errorMsg,
    toolMessageCount,
    // Edit
    startEdit,
    cancelEdit,
    saveEdit,
    // Delete
    confirmDeleteMessage,
    cancelDelete,
    doDelete,
    doRemoveAllTools,
  }
}
