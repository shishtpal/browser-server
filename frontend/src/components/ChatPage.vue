<template>
  <div class="grid h-[calc(100vh-57px)] max-w-full grid-cols-1 overflow-hidden bg-white text-slate-900 dark:bg-slate-950 dark:text-white lg:grid-cols-[300px_minmax(0,1fr)]">
    <!-- Desktop sidebar -->
    <ChatSidebar
      class="hidden lg:flex"
      :conversations="filteredConversations"
      :active-id="activeConversation?.id ?? null"
      :search="search"
      :status-label="configLabel"
      :disabled="!config?.enabled || isBusy"
      @new="startConversation"
      @select="handleSelectConversation"
      @rename="openRename"
      @delete="confirmDelete"
      @update:search="search = $event"
    />

    <!-- Main panel -->
    <section class="flex h-full flex-col overflow-hidden">
      <ChatTopBar
        :provider-names="providerNames"
        :selected-provider="selectedProvider"
        :selected-model="selectedModel"
        :models="providerModels"
        :supports-tools="selectedModelSupportsTools"
        :tools-enabled="toolsEnabled"
        :yolo-mode="yoloMode"
        :disabled="!config?.enabled || isBusy"
        :title="activeConversation?.title"
        @toggle-sidebar="showMobileSidebar = true"
        @update:selected-provider="selectedProvider = $event"
        @update:selected-model="selectedModel = $event"
        @update:yolo-mode="yoloMode = $event"
      />

      <!-- Error banner -->
      <ErrorBanner v-if="error" :message="error" :on-retry="() => (error = '')" class="mx-4 mt-3 shrink-0" />

      <!-- AI disabled state -->
      <ChatDisabledState v-if="config && !config.enabled" />

      <!-- Chat area -->
      <template v-else>
        <ChatMessageList
          ref="messageListRef"
          :messages="visibleMessages"
          :loading="isBusy"
          @suggestion="useSuggestion"
          @copy="copyMessage"
          @tool-decision="handleToolDecision"
        />

        <ChatRegenerateButton
          :visible="canRegenerate"
          :disabled="isBusy"
          @regenerate="handleRegenerate"
        />

        <ChatInput
          ref="chatInputRef"
          v-model="draft"
          :disabled="!config?.enabled"
          :busy="isBusy"
          @send="sendMessage"
          @stop="handleStop"
        />
      </template>
    </section>

    <!-- Mobile sidebar drawer -->
    <ChatMobileDrawer
      :open="showMobileSidebar"
      :conversations="filteredConversations"
      :active-id="activeConversation?.id ?? null"
      :disabled="!config?.enabled || isBusy"
      @close="showMobileSidebar = false"
      @new="startConversation(); showMobileSidebar = false"
      @select="handleSelectConversation($event); showMobileSidebar = false"
    />

    <!-- Rename modal -->
    <Modal :open="showRenameModal" title="Rename conversation" @close="showRenameModal = false">
      <form @submit.prevent="handleRename">
        <input
          v-model="renameTitle"
          class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 dark:border-white/10 dark:bg-slate-900"
          placeholder="Conversation title"
          autofocus
        />
        <div class="mt-4 flex justify-end gap-2">
          <button class="rounded-lg px-4 py-2 text-sm font-bold text-slate-500 hover:bg-slate-100 dark:hover:bg-white/10" type="button" @click="showRenameModal = false">Cancel</button>
          <button class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-bold text-white dark:bg-white dark:text-slate-900" type="submit">Save</button>
        </div>
      </form>
    </Modal>

    <!-- Delete confirmation modal -->
    <Modal :open="showDeleteModal" title="Delete conversation" @close="showDeleteModal = false">
      <p class="text-sm text-slate-600 dark:text-slate-400">Are you sure you want to delete "<strong>{{ deleteTarget?.title }}</strong>"? This action cannot be undone.</p>
      <div class="mt-4 flex justify-end gap-2">
        <button class="rounded-lg px-4 py-2 text-sm font-bold text-slate-500 hover:bg-slate-100 dark:hover:bg-white/10" type="button" @click="showDeleteModal = false">Cancel</button>
        <button class="rounded-lg bg-red-600 px-4 py-2 text-sm font-bold text-white hover:bg-red-700" type="button" @click="handleDelete">Delete</button>
      </div>
    </Modal>

    <!-- Copy toast -->
    <ChatCopyToast :visible="showCopyToast" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { getAIConfig, getAIConversation } from '../lib/api'
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
import { useChatConfig } from './chat/composables/useChatConfig'
import { useChatConversations } from './chat/composables/useChatConversations'
import { useChatMessaging } from './chat/composables/useChatMessaging'

// ─── Composables ───────────────────────────────────────

const {
  config,
  selectedProvider,
  selectedModel,
  yoloMode,
  configLabel,
  providerModels,
  selectedModelSupportsTools,
  toolsEnabled,
  initFromConfig,
  loadPersistedSettings,
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
  loadConversations,
  createConversation,
  selectConversation,
  refreshConversation,
  autoTitle,
} = useChatConversations()

const {
  isBusy,
  canRegenerate,
  visibleMessages,
  send,
  decideToolCall,
  regenerate,
  stop,
  cleanup,
} = useChatMessaging(
  () => activeConversation.value,
  () => messages.value,
  (msgs) => { messages.value = msgs },
)

// ─── Local state ───────────────────────────────────────

const draft = ref('')
const error = ref('')
const showMobileSidebar = ref(false)
const showCopyToast = ref(false)

const messageListRef = ref<InstanceType<typeof ChatMessageList> | null>(null)
const chatInputRef = ref<InstanceType<typeof ChatInput> | null>(null)

const providerNames = computed(() => Object.keys(config.value?.providers ?? {}))

// ─── Lifecycle ─────────────────────────────────────────

onMounted(async () => {
  window.addEventListener('api-token-changed', reload)
  loadPersistedSettings()
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
  error.value = ''
  try {
    await createConversation(selectedProvider.value, selectedModel.value)
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
    nextTick(() => messageListRef.value?.scrollToBottom())
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load conversation'
  }
}

async function sendMessage(content?: string) {
  const text = content?.trim() || draft.value.trim()
  if (!text || !config.value?.enabled || !selectedProvider.value || !selectedModel.value || isBusy.value) return
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
        streamEnabled: config.value?.chat?.stream !== false,
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

async function handleToolDecision(callId: string, approved: boolean) {
  await decideToolCall(callId, approved, (msg) => { error.value = msg })
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

// ─── Utilities ─────────────────────────────────────────

function useSuggestion(text: string) {
  draft.value = text
  nextTick(() => chatInputRef.value?.focus())
}

async function copyMessage(content: string) {
  try {
    await navigator.clipboard.writeText(content)
    showCopyToast.value = true
    setTimeout(() => { showCopyToast.value = false }, 2000)
  } catch { /* silent */ }
}
</script>
