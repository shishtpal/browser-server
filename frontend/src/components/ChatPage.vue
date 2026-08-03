<template>
  <div
    class="chat-shell grid h-[calc(100vh-57px)] max-w-full grid-cols-1 overflow-hidden bg-white text-slate-900 dark:bg-slate-950 dark:text-white"
    :class="gridClass" :style="chatFontStyle">
    <!-- Desktop sidebar -->
    <ChatSidebar class="hidden lg:flex" :conversations="filteredConversations"
      :active-id="activeConversation?.id ?? null" :search="search" :status-label="configLabel"
      :disabled="!config?.enabled || isBusy" :archived-conversations="archivedConversations"
      :show-archived="showArchived" @new="startConversation" @select="handleSelectConversation" @rename="openRename"
      @delete="confirmDelete" @archive="confirmArchive" @restore="confirmRestore" @toggle-archived="toggleArchived"
      @update:search="search = $event" />

    <!-- Main panel -->
    <section class="flex h-full min-h-0 flex-col overflow-hidden">
      <ChatTopBar :profiles="profiles" :selected-profile="selectedProfile" :profile-locked="profileLocked"
        :skills="skills" :active-skills="activeSkills" :provider-names="providerNames"
        :selected-provider="selectedProvider" :selected-model="selectedModel" :models="providerModels"
        :supports-tools="selectedModelSupportsTools" :tools-enabled="toolsEnabled" :yolo-mode="yoloMode"
        :disabled="!config?.enabled || isBusy" :title="activeConversation?.title"
        :download-disabled="!activeConversation" :show-tools-panel="showToolsPanel"
        :show-memory-explorer="showMemoryExplorer" :show-prompt-manager="showPromptManager"
        :show-attachment-gallery="showAttachmentGallery"
        @toggle-sidebar="showMobileSidebar = true" @update:selected-profile="selectedProfile = $event"
        @update:selected-provider="selectedProvider = $event" @update:selected-model="selectedModel = $event"
        @update:yolo-mode="yoloMode = $event" @toggle-skill="toggleSkill($event)" @download="downloadConversation"
        @toggle-tools-panel="showToolsPanel = !showToolsPanel"
        @toggle-memory-explorer="showMemoryExplorer = !showMemoryExplorer"
        @toggle-prompt-manager="showPromptManager = !showPromptManager"
        @toggle-attachment-gallery="showAttachmentGallery = !showAttachmentGallery" />

      <!-- Error banner -->
      <ErrorBanner v-if="error" :message="error" :on-retry="() => (error = '')" class="mx-4 mt-3 shrink-0" />

      <!-- AI disabled state -->
      <ChatDisabledState v-if="config && !config.enabled" />

      <!-- Chat area -->
      <template v-else>
        <ChatMessageList ref="messageListRef" :messages="visibleMessages" :loading="isBusy" @suggestion="useSuggestion"
          @copy="copyMessage" @delete="deleteMessage" @branch="handleBranch" @tool-decision="handleToolDecision" />

        <ChatRegenerateButton :visible="canRegenerate" :disabled="isBusy" @regenerate="handleRegenerate" />

        <ChatInput ref="chatInputRef" v-model="draft" :disabled="!config?.enabled" :busy="isBusy"
          :can-append="canAppend" :is-appending="isAppending" :user-id="currentUserId"
          :conversation-id="activeConversation?.id" :attachments-config="attachmentsConfig"
          :supports-vision="selectedModelSupportsVision" :staged-images="stagedAttachments"
          @send="sendMessage" @append="appendContext" @stop="handleStop" @voice="showVoiceModal = true"
          @select-prompt="useSuggestion" @add-images="addImages" @remove-image="removeStagedImage" />
      </template>
    </section>

    <!-- Right tools panel (desktop) -->
    <ChatToolsPanel v-if="showToolsPanel" class="hidden lg:flex" :tools-enabled="userToolsEnabled"
      :model-supports-tools="selectedModelSupportsTools" :yolo-mode="yoloMode"
      :include-all-tool-definitions="includeAllToolDefinitions" :available-tools="availableTools"
      :tools-by-category="toolsByCategory" :disabled-tools="disabledTools" :tool-calls="toolCallEntries" :mcp="mcp"
      :font-family="chatFontFamily" :font-size="chatFontSize" :raw-tool-output="rawToolOutput"
      @close="showToolsPanel = false" @update:tools-enabled="userToolsEnabled = $event"
      @update:yolo-mode="yoloMode = $event" @update:include-all-tool-definitions="includeAllToolDefinitions = $event"
      @update:font-family="chatFontFamily = $event" @update:font-size="chatFontSize = $event"
      @update:raw-tool-output="rawToolOutput = $event" @toggle-tool="toggleTool" />

    <!-- Mobile sidebar drawer -->
    <ChatMobileDrawer :open="showMobileSidebar" :conversations="filteredConversations"
      :active-id="activeConversation?.id ?? null" :disabled="!config?.enabled || isBusy"
      :archived-conversations="archivedConversations" :show-archived="showArchived" @close="showMobileSidebar = false"
      @new="startConversation(); showMobileSidebar = false"
      @select="handleSelectConversation($event); showMobileSidebar = false" @rename="openRename" @delete="confirmDelete"
      @archive="confirmArchive" @restore="confirmRestore" @toggle-archived="toggleArchived" />

    <!-- Voice typing modal -->
    <ChatVoiceTypingModal :open="showVoiceModal" @close="showVoiceModal = false" @use="useVoiceTranscript" />

    <!-- Rename modal -->
    <Modal :open="showRenameModal" title="Rename conversation" @close="showRenameModal = false">
      <form @submit.prevent="handleRename">
        <input v-model="renameTitle"
          class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 dark:border-white/10 dark:bg-slate-900"
          placeholder="Conversation title" autofocus />
        <div class="mt-4 flex justify-end gap-2">
          <button
            class="rounded-lg px-4 py-2 text-sm font-bold text-slate-500 hover:bg-slate-100 dark:hover:bg-white/10"
            type="button" @click="showRenameModal = false">Cancel</button>
          <button
            class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-bold text-white dark:bg-white dark:text-slate-900"
            type="submit">Save</button>
        </div>
      </form>
    </Modal>

    <!-- Archive confirmation modal -->
    <Modal :open="showArchiveModal" title="Archive conversation" @close="showArchiveModal = false">
      <p class="text-sm text-slate-600 dark:text-slate-400">Archive "<strong>{{ archiveTarget?.title }}</strong>"? It
        will
        be moved to the Archived section.</p>
      <div class="mt-4 flex justify-end gap-2">
        <button class="rounded-lg px-4 py-2 text-sm font-bold text-slate-500 hover:bg-slate-100 dark:hover:bg-white/10"
          type="button" @click="showArchiveModal = false">Cancel</button>
        <button class="rounded-lg bg-amber-600 px-4 py-2 text-sm font-bold text-white hover:bg-amber-700" type="button"
          @click="handleArchive">Archive</button>
      </div>
    </Modal>

    <!-- Restore confirmation modal -->
    <Modal :open="showRestoreModal" title="Restore conversation" @close="showRestoreModal = false">
      <p class="text-sm text-slate-600 dark:text-slate-400">Restore "<strong>{{ restoreTarget?.title }}</strong>"? It
        will
        reappear in the main list.</p>
      <div class="mt-4 flex justify-end gap-2">
        <button class="rounded-lg px-4 py-2 text-sm font-bold text-slate-500 hover:bg-slate-100 dark:hover:bg-white/10"
          type="button" @click="showRestoreModal = false">Cancel</button>
        <button class="rounded-lg bg-green-600 px-4 py-2 text-sm font-bold text-white hover:bg-green-700" type="button"
          @click="handleRestore">Restore</button>
      </div>
    </Modal>

    <!-- Delete confirmation modal -->
    <Modal :open="showDeleteModal" title="Delete conversation" @close="showDeleteModal = false">
      <p class="text-sm text-slate-600 dark:text-slate-400">Are you sure you want to delete "<strong>{{
        deleteTarget?.title
          }}</strong>"? This action cannot be undone.</p>
      <div class="mt-4 flex justify-end gap-2">
        <button class="rounded-lg px-4 py-2 text-sm font-bold text-slate-500 hover:bg-slate-100 dark:hover:bg-white/10"
          type="button" @click="showDeleteModal = false">Cancel</button>
        <button class="rounded-lg bg-red-600 px-4 py-2 text-sm font-bold text-white hover:bg-red-700" type="button"
          @click="handleDelete">Delete</button>
      </div>
    </Modal>

    <!-- Copy toast -->
    <ChatCopyToast :visible="showCopyToast" />

    <!-- Branch toast -->
    <Transition name="toast">
      <div v-if="showBranchToast"
        class="pointer-events-none fixed bottom-6 left-1/2 z-50 -translate-x-1/2 rounded-lg border border-indigo-400/40 bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white shadow-lg shadow-indigo-600/30">
        <span class="inline-flex items-center gap-2">
          <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round"
              d="M6 3v12m0 0a3 3 0 103 3M6 15a3 3 0 013-3h6a3 3 0 003-3V6m0 0a3 3 0 10-3-3 3 3 0 003 3z" />
          </svg>
          Branched into a new conversation
        </span>
      </div>
    </Transition>

    <!-- Memory Explorer modal -->
    <ChatMemoryExplorer :open="showMemoryExplorer" :conversation-id="activeConversation?.id ?? ''" :messages="messages"
      @close="showMemoryExplorer = false" @updated="messages = $event" />

    <!-- Prompt Manager modal -->
    <PromptManager :open="showPromptManager" :user-id="currentUserId" @select="applyPromptFromManager"
      @close="showPromptManager = false" />

    <!-- Attachment library modal -->
    <ChatAttachmentGallery :open="showAttachmentGallery" @close="showAttachmentGallery = false"
      @reuse="handleReuseAttachment" />

    <!-- New Conversation modal -->
    <ChatNewConversationModal :open="showNewConversationModal" :profiles="profiles" :provider-names="providerNames"
      :providers="config?.providers ?? {}" :skills="skills" :default-provider="selectedProvider"
      :default-model="selectedModel" :default-profile="profileLocked ? '' : selectedProfile"
      :default-skills="activeSkills" @close="showNewConversationModal = false" @create="handleNewConversationCreate" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useLocalStorage } from '@vueuse/core'
import { deleteAIMessage, getAIConfig, getAIConversation, getAIImageAttachmentBlob } from '../lib/api'
import type { AIAttachmentSummary, AIChatAttachmentsConfig } from '@browser-server/shared-types'
import Modal from './ui/Modal.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import ChatSidebar from './chat/ChatSidebar.vue'
import ChatTopBar from './chat/ChatTopBar.vue'
import ChatMessageList from './chat/ChatMessageList.vue'
import ChatInput from './chat/ChatInput.vue'
import ChatRegenerateButton from './chat/ChatRegenerateButton.vue'
import ChatMobileDrawer from './chat/ChatMobileDrawer.vue'
import ChatDisabledState from './chat/ChatDisabledState.vue'
import ChatCopyToast from './chat/ChatCopyToast.vue'
import ChatToolsPanel from './chat/ChatToolsPanel.vue'
import ChatMemoryExplorer from './chat/ChatMemoryExplorer.vue'
import ChatNewConversationModal from './chat/ChatNewConversationModal.vue'
import ChatAttachmentGallery from './chat/ChatAttachmentGallery.vue'
import ChatVoiceTypingModal from './chat/ChatVoiceTypingModal.vue'
import type { NewConversationResult } from './chat/ChatNewConversationModal.vue'
import type { ToolCallEntry } from './chat/ChatToolsPanel.vue'
import { useChatConfig } from './chat/composables/useChatConfig'
import { useChatConversations } from './chat/composables/useChatConversations'
import { useChatMessaging } from './chat/composables/useChatMessaging'
import { useUser } from '../composables/useUser'
import PromptManager from './prompts/PromptManager.vue'

// ─── Composables ───────────────────────────────────────

const {
  config,
  selectedProvider,
  selectedModel,
  selectedProfile,
  profiles,
  skills,
  mcp,
  activeSkills,
  yoloMode,
  userToolsEnabled,
  includeAllToolDefinitions,
  disabledTools,
  configLabel,
  providerModels,
  selectedModelSupportsTools,
  toolsEnabled,
  availableTools,
  toolsByCategory,
  activeTools,
  selectedModelSupportsVision,
  attachmentsConfig,
  toggleTool,
  toggleSkill,
  setActiveSkills,
  initFromConfig,
} = useChatConfig()

const {
  conversations,
  activeConversation,
  messages,
  search,
  error: convError,
  filteredConversations,
  showRenameModal,
  renameTitle,
  showDeleteModal,
  deleteTarget,
  openRename,
  doRename,
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
  forkConversation,
  selectConversation,
  refreshConversation,
  autoTitle,
} = useChatConversations()

const {
  isBusy,
  canAppend,
  isAppending,
  canRegenerate,
  visibleMessages,
  send,
  append,
  stagedAttachments,
  addImageAttachments,
  removeStagedAttachment,
  decideToolCall,
  regenerate,
  stop,
  cleanup,
} = useChatMessaging(
  () => activeConversation.value,
  () => messages.value,
  (msgs) => { messages.value = msgs },
)

const { currentUserId } = useUser()

// ─── Local state ───────────────────────────────────────

const draft = ref('')
const error = ref('')
const showMobileSidebar = ref(false)
const showCopyToast = ref(false)
const showBranchToast = ref(false)
const showToolsPanel = ref(true)
const showMemoryExplorer = ref(false)
const showNewConversationModal = ref(false)
const showPromptManager = ref(false)
const showAttachmentGallery = ref(false)
const showVoiceModal = ref(false)

// Archive/Restore local state
const chatFontFamily = useLocalStorage(`bs.ai.chatFontFamily`, 'system-ui')
const chatFontSize = useLocalStorage(`bs.ai.chatFontSize`, 14)
const rawToolOutput = useLocalStorage<boolean | null>('bs.ai.rawToolOutput', null, {
  serializer: {
    read: (v) => {
      if (v == null) return null
      if (v === 'true') return true
      if (v === 'false') return false
      return null
    },
    write: (v) => String(v),
  },
})

const messageListRef = ref<InstanceType<typeof ChatMessageList> | null>(null)
const chatInputRef = ref<InstanceType<typeof ChatInput> | null>(null)

const providerNames = computed(() => Object.keys(config.value?.providers ?? {}))

// Profile is locked once the conversation has messages (cannot change mid-conversation)
const profileLocked = computed(() => {
  if (!activeConversation.value) return false
  return messages.value.length > 0
})

// ─── Grid layout ───────────────────────────────────────

const gridClass = computed(() => {
  if (showToolsPanel.value) {
    return 'lg:grid-cols-[300px_minmax(0,1fr)_auto]'
  }
  return 'lg:grid-cols-[300px_minmax(0,1fr)]'
})

const chatFontStyle = computed(() => ({
  fontFamily: chatFontFamily.value,
  fontSize: chatFontSize.value + 'px',
}))

// ─── Tool call entries for the panel ───────────────────

const toolCallEntries = computed<ToolCallEntry[]>(() => {
  return messages.value
    .filter((m) => m.role === 'tool')
    .map((m) => {
      let name = 'Tool call'
      let args: string | undefined
      let result: string | undefined
      let status = m.status === 'pending' ? 'pending' : 'completed'
      try {
        const parsed = JSON.parse(m.content)
        name = parsed.tool || name
        if (parsed.args) args = typeof parsed.args === 'string' ? parsed.args : JSON.stringify(parsed.args, null, 2)
        if (parsed.result !== null && parsed.result !== undefined) {
          result = typeof parsed.result === 'string' ? parsed.result : JSON.stringify(parsed.result, null, 2)
        }
        if (parsed.decision === 'rejected') status = 'rejected'
        else if (parsed.decision === 'commented') status = 'commented'
        else if (parsed.result?.error) status = 'error'
        else if (m.status === 'completed') status = 'completed'
        else if (m.status === 'error') status = 'error'
      } catch { /* use defaults */ }
      return { id: m.tool_call_id || m.id, name, status, args, result }
    })
})

// ─── Lifecycle ─────────────────────────────────────────

onMounted(async () => {
  window.addEventListener('api-token-changed', reload)
  await reload()
})

onUnmounted(() => {
  window.removeEventListener('api-token-changed', reload)
  cleanup()
})

// ─── Core actions ──────────────────────────────────────

async function reload() {
  error.value = ''
  try {
    const cfg = await getAIConfig()
    initFromConfig(cfg)
    if (!cfg.enabled) return
    await loadConversations()
    if (conversations.value.length > 0) {
      await handleSelectConversation(conversations.value[0].id)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load AI chat'
  }
}

async function startConversation() {
  if (!config.value?.enabled) return
  showNewConversationModal.value = true
}

async function handleNewConversationCreate(result: NewConversationResult) {
  showNewConversationModal.value = false
  error.value = ''
  try {
    // Apply the user's choices to the active config state
    selectedProvider.value = result.provider
    selectedModel.value = result.model
    selectedProfile.value = result.profile
    setActiveSkills(result.skills)

    await createConversation(result.provider, result.model, result.profile || undefined)
    nextTick(() => chatInputRef.value?.focus())
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to create conversation'
  }
}

async function handleSelectConversation(id: string) {
  error.value = ''
  try {
    const { provider, model } = await selectConversation(id)
    selectedProvider.value = provider
    selectedModel.value = model
    // Set profile from the conversation (locked once selected)
    selectedProfile.value = activeConversation.value?.profile || ''
    // Restore active skills from conversation state
    setActiveSkills(activeConversation.value?.skills ?? [])
    nextTick(() => messageListRef.value?.scrollToBottom())
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load conversation'
  }
}

async function sendMessage(content?: string) {
  const text = content?.trim() || draft.value.trim()
  if (!config.value?.enabled || !selectedProvider.value || !selectedModel.value || isBusy.value) return
  if (!text && stagedAttachments.value.length === 0) return
  error.value = ''
  draft.value = ''

  try {
    if (!activeConversation.value) {
      await startConversation()
    }
    if (!activeConversation.value) return

    await send(
      text,
      activeConversation.value.id,
      {
        provider: selectedProvider.value,
        model: selectedModel.value,
        toolsEnabled: toolsEnabled.value,
        yoloMode: yoloMode.value,
        includeAllToolDefinitions: includeAllToolDefinitions.value,
        streamEnabled: config.value?.chat?.stream !== false,
        activeTools: activeTools.value,
        skills: activeSkills.value,
        rawToolOutput: rawToolOutput.value ?? undefined,
      },
      async (conversationId, firstMessage) => {
        await refreshConversation(conversationId)
        await autoTitle(firstMessage)
        await loadConversations()
      },
      (msg) => { error.value = msg },
    )
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to send message'
  }
}

async function addImages(files: File[]) {
  if (!activeConversation.value) return
  await addImageAttachments(activeConversation.value.id, files)
}

async function removeStagedImage(id: string) {
  if (!activeConversation.value) return
  await removeStagedAttachment(activeConversation.value.id, id)
}

async function handleReuseAttachment(att: AIAttachmentSummary) {
  if (!activeConversation.value) {
    error.value = 'Select or create a conversation before reusing an attachment.'
    return
  }
  error.value = ''
  try {
    const blob = await getAIImageAttachmentBlob(att.conversation_id, att.id)
    const file = new File([blob], att.filename, { type: att.content_type })
    await addImages([file])
    showAttachmentGallery.value = false
    nextTick(() => chatInputRef.value?.focus())
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to reuse attachment'
  }
}

async function appendContext(content: string) {
  const text = content.trim()
  if (!text || !activeConversation.value) return
  const conversationId = activeConversation.value.id
  const draftAtSubmit = draft.value
  error.value = ''
  const appended = await append(text, conversationId, (msg) => { error.value = msg })
  if (appended && activeConversation.value?.id === conversationId && draft.value === draftAtSubmit) {
    draft.value = ''
  }
}

async function handleToolDecision(callId: string, approved: boolean, comment: string) {
  await decideToolCall(callId, approved, comment, (msg) => { error.value = msg })
}

async function handleRegenerate() {
  if (!activeConversation.value) return
  error.value = ''
  await regenerate(activeConversation.value.id, (msg) => { error.value = msg })
  if (activeConversation.value) {
    try {
      const detail = await getAIConversation(activeConversation.value.id)
      messages.value = detail.messages ?? []
    } catch { /* messages will refresh on next action */ }
  }
}

async function handleStop() {
  if (!activeConversation.value) return
  await stop(activeConversation.value.id)
}

async function handleRename() {
  try {
    await doRename()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to rename'
  }
}

async function handleDelete() {
  try {
    await doDelete()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to delete'
  }
}

async function handleArchive() {
  try {
    await doArchive()
    await loadArchivedConversations()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to archive'
  }
}

async function handleRestore() {
  try {
    await doRestore()
    await loadArchivedConversations()
    await loadConversations()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to restore'
  }
}

// ─── Utilities ─────────────────────────────────────────

function useSuggestion(text: string | { content?: string }) {
  draft.value = typeof text === 'string' ? text : text.content ?? ''
  nextTick(() => chatInputRef.value?.focus())
}

function useVoiceTranscript(text: string) {
  const existing = draft.value.trimEnd()
  draft.value = existing ? `${existing} ${text.trim()}` : text.trim()
  showVoiceModal.value = false
  nextTick(() => chatInputRef.value?.focus())
}

function applyPromptFromManager(prompt: unknown) {
  const raw = prompt as Record<string, unknown> | null | undefined
  const candidate = raw?.content
    ?? raw?.Content
    ?? (raw?.Prompt as Record<string, unknown> | undefined)?.content
    ?? (raw?.prompt as Record<string, unknown> | undefined)?.content

  let text = ''
  if (typeof candidate === 'string') {
    text = candidate
  } else if (candidate != null) {
    try {
      text = JSON.stringify(candidate)
    } catch {
      text = String(candidate)
    }
  }

  if (!text) return
  draft.value = text
  showPromptManager.value = false
  nextTick(() => chatInputRef.value?.focus())
}

async function copyMessage(content: string) {
  try {
    await navigator.clipboard.writeText(content)
    showCopyToast.value = true
    setTimeout(() => { showCopyToast.value = false }, 2000)
  } catch { /* silent */ }
}

async function deleteMessage(messageId: string) {
  if (!activeConversation.value || isBusy.value || messageId.startsWith('temp-')) return
  if (!window.confirm('Delete this message? This action cannot be undone.')) return

  error.value = ''
  try {
    await deleteAIMessage(activeConversation.value.id, messageId)
    messages.value = messages.value.filter((message) => message.id !== messageId)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to delete message'
  }
}

async function handleBranch(messageId: string) {
  if (!activeConversation.value || isBusy.value || messageId.startsWith('temp-')) return
  error.value = ''
  try {
    const source = activeConversation.value
    const forked = await forkConversation(source.id, messageId)
    // Carry the source conversation's runtime selection into the new branch.
    selectedProvider.value = forked.provider
    selectedModel.value = forked.model
    selectedProfile.value = forked.profile || ''
    setActiveSkills(forked.skills ?? [])
    await loadConversations()
    showBranchToast.value = true
    setTimeout(() => { showBranchToast.value = false }, 2200)
    nextTick(() => {
      messageListRef.value?.scrollToBottom()
      chatInputRef.value?.focus()
    })
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to branch conversation'
  }
}

function downloadConversation() {
  const conversation = activeConversation.value
  if (!conversation) return

  const title = conversation.title.replace(/[\r\n]+/g, ' ').trim() || 'AI conversation'
  const sections = visibleMessages.value.map((message) => {
    const role = message.role.charAt(0).toUpperCase() + message.role.slice(1)
    return `## ${role}\n\n${message.content.trim()}`
  })
  const markdown = [
    `# ${title}`,
    `- **Provider:** ${conversation.provider}`,
    `- **Model:** ${conversation.model}`,
    `- **Created:** ${conversation.created_at}`,
    '',
    '---',
    '',
    ...sections.flatMap((section) => [section, '']),
  ].join('\n').trimEnd() + '\n'

  const blobUrl = URL.createObjectURL(new Blob([markdown], { type: 'text/markdown;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = blobUrl
  link.download = `${filenameSafe(title)}.md`
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(blobUrl)
}

function filenameSafe(value: string): string {
  return value
    .replace(/[<>:"/\\|?*\u0000-\u001F]/g, '-')
    .replace(/[. ]+$/g, '')
    .slice(0, 100) || 'ai-conversation'
}
</script>
