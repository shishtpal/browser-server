import type { AIConversation, AIMessage } from '@browser-server/shared-types'
import { computed, ref } from 'vue'
import {
  archiveAIConversation,
  createAIConversation,
  deleteAIConversation,
  getAIConversation,
  listAIConversations,
  listArchivedAIConversations,
  restoreAIConversation,
  updateAIConversation,
} from '../../../lib/api'

export function useChatConversations() {
  const conversations = ref<AIConversation[]>([])
  const activeConversation = ref<AIConversation | null>(null)
  const messages = ref<AIMessage[]>([])
  const search = ref('')
  const error = ref('')

  // Rename state
  const showRenameModal = ref(false)
  const renameTarget = ref<AIConversation | null>(null)
  const renameTitle = ref('')

  // Delete state
  const showDeleteModal = ref(false)
  const deleteTarget = ref<AIConversation | null>(null)

  // Archive state
  const showArchiveModal = ref(false)
  const archiveTarget = ref<AIConversation | null>(null)

  // Restore state
  const showRestoreModal = ref(false)
  const restoreTarget = ref<AIConversation | null>(null)

  // Archived conversations
  const archivedConversations = ref<AIConversation[]>([])
  const showArchived = ref(false)

  const filteredConversations = computed(() => {
    const query = search.value.trim().toLowerCase()
    if (!query) return conversations.value
    return conversations.value.filter((c) =>
      c.title.toLowerCase().includes(query) || c.model.toLowerCase().includes(query) || c.preview?.toLowerCase().includes(query)
    )
  })

  async function loadConversations() {
    conversations.value = await listAIConversations(undefined, 50)
  }

  async function createConversation(provider: string, model: string, profile?: string): Promise<AIConversation> {
    const conversation = await createAIConversation({ provider, model, profile })
    conversations.value = [conversation, ...conversations.value]
    activeConversation.value = conversation
    messages.value = []
    return conversation
  }

  async function selectConversation(id: string): Promise<{ provider: string; model: string }> {
    const detail = await getAIConversation(id)
    activeConversation.value = detail.conversation
    messages.value = detail.messages ?? []
    return { provider: detail.conversation.provider, model: detail.conversation.model }
  }

  async function refreshConversation(id: string) {
    const detail = await getAIConversation(id)
    if (activeConversation.value?.id === id) {
      messages.value = detail.messages ?? []
    }
  }

  // Rename
  function openRename(conversation: AIConversation) {
    renameTarget.value = conversation
    renameTitle.value = conversation.title
    showRenameModal.value = true
  }

  async function doRename() {
    if (!renameTarget.value || !renameTitle.value.trim()) return
    const updated = await updateAIConversation(renameTarget.value.id, { title: renameTitle.value.trim() })
    const idx = conversations.value.findIndex((c) => c.id === updated.id)
    if (idx >= 0) conversations.value[idx] = updated
    if (activeConversation.value?.id === updated.id) activeConversation.value = updated
    showRenameModal.value = false
  }

  // Delete
  function confirmDelete(conversation: AIConversation) {
    deleteTarget.value = conversation
    showDeleteModal.value = true
  }

  async function doDelete() {
    if (!deleteTarget.value) return
    const id = deleteTarget.value.id
    await deleteAIConversation(id)
    conversations.value = conversations.value.filter((c) => c.id !== id)
    if (activeConversation.value?.id === id) {
      activeConversation.value = null
      messages.value = []
    }
    showDeleteModal.value = false
  }

  // Auto-title after first message
  async function autoTitle(firstMessage: string) {
    if (!activeConversation.value) return
    if (messages.value.filter((m) => m.role === 'user').length === 1 && activeConversation.value.title === 'New chat') {
      const title = firstMessage.slice(0, 60)
      activeConversation.value = await updateAIConversation(activeConversation.value.id, { title })
    }
  }

  // Archive
  function confirmArchive(conversation: AIConversation) {
    archiveTarget.value = conversation
    showArchiveModal.value = true
  }

  async function doArchive() {
    if (!archiveTarget.value) return
    const id = archiveTarget.value.id
    await archiveAIConversation(id)
    conversations.value = conversations.value.filter((c) => c.id !== id)
    if (activeConversation.value?.id === id) {
      activeConversation.value = null
      messages.value = []
    }
    showArchiveModal.value = false
    archiveTarget.value = null
  }

  // Restore
  function confirmRestore(conversation: AIConversation) {
    restoreTarget.value = conversation
    showRestoreModal.value = true
  }

  async function doRestore() {
    if (!restoreTarget.value) return
    const id = restoreTarget.value.id
    await restoreAIConversation(id)
    archivedConversations.value = archivedConversations.value.filter((c) => c.id !== id)
    showRestoreModal.value = false
    restoreTarget.value = null
  }

  async function loadArchivedConversations() {
    archivedConversations.value = await listArchivedAIConversations()
  }

  function toggleArchived() {
    showArchived.value = !showArchived.value
    if (showArchived.value) {
      loadArchivedConversations()
    }
  }

  return {
    conversations,
    activeConversation,
    messages,
    search,
    error,
    filteredConversations,
    // Rename
    showRenameModal,
    renameTarget,
    renameTitle,
    openRename,
    doRename,
    // Delete
    showDeleteModal,
    deleteTarget,
    confirmDelete,
    doDelete,
    // Archive
    showArchiveModal,
    archiveTarget,
    confirmArchive,
    doArchive,
    // Restore
    showRestoreModal,
    restoreTarget,
    confirmRestore,
    doRestore,
    // Archived
    archivedConversations,
    showArchived,
    toggleArchived,
    loadArchivedConversations,
    // Actions
    loadConversations,
    createConversation,
    selectConversation,
    refreshConversation,
    autoTitle,
  }
}
