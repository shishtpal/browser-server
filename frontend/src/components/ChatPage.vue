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
      @select="selectConversation"
      @rename="openRename"
      @delete="confirmDelete"
      @update:search="search = $event"
    />

    <!-- Main panel -->
    <section class="flex h-full flex-col overflow-hidden">
      <!-- Top bar -->
      <header class="flex shrink-0 flex-wrap items-center gap-3 border-b border-slate-200 px-4 py-3 dark:border-white/10">
        <button
          class="rounded-lg border border-slate-200 p-2 lg:hidden dark:border-white/10"
          type="button"
          aria-label="Toggle sidebar"
          @click="showMobileSidebar = true"
        >
          <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/></svg>
        </button>

        <select
          v-model="selectedProvider"
          class="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-white/10 dark:bg-slate-900"
          :disabled="!config?.enabled || isBusy"
        >
          <option v-for="(_, name) in config?.providers" :key="name" :value="name">{{ name }}</option>
        </select>
        <select
          v-model="selectedModel"
          class="min-w-0 flex-1 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm sm:flex-none dark:border-white/10 dark:bg-slate-900"
          :disabled="!config?.enabled || isBusy"
        >
          <option v-for="model in providerModels" :key="model.id" :value="model.id">{{ model.label || model.id }}{{ model.supports_tools ? ' 🔧' : '' }}</option>
        </select>
        <span v-if="selectedModelSupportsTools" class="hidden items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-semibold text-amber-700 sm:flex dark:bg-amber-900/30 dark:text-amber-300">
          🔧 Tools
        </span>
        <span v-if="activeConversation" class="ml-auto hidden truncate text-xs text-slate-500 sm:block dark:text-slate-400">
          {{ activeConversation.title }}
        </span>
      </header>

      <!-- Error banner -->
      <div v-if="error" class="mx-4 mt-3 flex shrink-0 items-start gap-3 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300">
        <svg class="mt-0.5 h-4 w-4 shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/></svg>
        <span class="flex-1">{{ error }}</span>
        <button class="shrink-0 text-red-400 hover:text-red-600" type="button" @click="error = ''">✕</button>
      </div>

      <!-- AI disabled state -->
      <div v-if="config && !config.enabled" class="grid flex-1 place-items-center p-6 text-center">
        <div class="max-w-sm">
          <div class="mx-auto mb-4 grid h-16 w-16 place-items-center rounded-2xl bg-slate-100 dark:bg-white/5">
            <svg class="h-8 w-8 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.75 3.104v5.714a2.25 2.25 0 01-.659 1.591L5 14.5M9.75 3.104c-.251.023-.501.05-.75.082m.75-.082a24.301 24.301 0 014.5 0m0 0v5.714c0 .597.237 1.17.659 1.591L19.8 15.3M14.25 3.104c.251.023.501.05.75.082M19.8 15.3l-1.57.393A9.065 9.065 0 0112 15a9.065 9.065 0 00-6.23.693L5 14.5m14.8.8l1.402 1.402c1.232 1.232.65 3.318-1.067 3.611A48.309 48.309 0 0112 21c-2.773 0-5.491-.235-8.135-.687-1.718-.293-2.3-2.379-1.067-3.61L5 14.5"/></svg>
          </div>
          <h2 class="text-xl font-black">AI is disabled</h2>
          <p class="mt-2 text-sm text-slate-500 dark:text-slate-400">Create <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs font-mono dark:bg-white/10">bs-ai-config.json</code> with provider credentials, then restart the server.</p>
        </div>
      </div>

      <!-- Chat area: messages + input pinned to viewport bottom -->
      <template v-else>
        <ChatMessageList
          ref="messageListRef"
          :messages="visibleMessages"
          :loading="isBusy"
          @suggestion="useSuggestion"
          @copy="copyMessage"
        />

        <!-- Regenerate button -->
        <div v-if="canRegenerate" class="flex shrink-0 justify-center border-t border-slate-100 px-4 py-2 dark:border-white/5">
          <button
            class="flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-600 transition hover:border-slate-300 hover:bg-slate-50 dark:border-white/10 dark:text-slate-400 dark:hover:border-white/20 dark:hover:bg-white/5"
            type="button"
            :disabled="isBusy"
            @click="regenerate"
          >
            <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
            Regenerate
          </button>
        </div>

        <ChatInput
          ref="chatInputRef"
          v-model="draft"
          :disabled="!config?.enabled"
          :busy="isBusy"
          @send="sendMessage"
          @stop="stop"
        />
      </template>
    </section>

    <!-- Mobile sidebar drawer -->
    <Teleport to="body">
      <div v-if="showMobileSidebar" class="fixed inset-0 z-50 flex lg:hidden">
        <div class="absolute inset-0 bg-slate-950/50 backdrop-blur-sm" @click="showMobileSidebar = false"></div>
        <aside class="relative z-10 flex h-full w-80 max-w-[85vw] flex-col bg-white dark:bg-slate-900">
          <div class="flex items-center justify-between border-b border-slate-200 p-4 dark:border-white/10">
            <h2 class="font-black">Conversations</h2>
            <button class="rounded-lg p-2 hover:bg-slate-100 dark:hover:bg-white/10" type="button" @click="showMobileSidebar = false">✕</button>
          </div>
          <div class="flex-1 space-y-1 overflow-y-auto p-3">
            <button
              class="mb-3 w-full rounded-lg bg-slate-900 px-3 py-2 text-sm font-bold text-white dark:bg-white dark:text-slate-900"
              :disabled="!config?.enabled || isBusy"
              type="button"
              @click="startConversation(); showMobileSidebar = false"
            >+ New Chat</button>
            <div
              v-for="conversation in filteredConversations"
              :key="'m-' + conversation.id"
              class="cursor-pointer rounded-lg p-3 transition"
              :class="conversation.id === activeConversation?.id ? 'bg-slate-100 dark:bg-white/10' : 'hover:bg-slate-50 dark:hover:bg-white/5'"
              @click="selectConversation(conversation.id); showMobileSidebar = false"
            >
              <span class="block truncate text-sm font-semibold">{{ conversation.title }}</span>
              <span class="block truncate text-xs text-slate-500">{{ conversation.model }}</span>
            </div>
          </div>
        </aside>
      </div>
    </Teleport>

    <!-- Rename modal -->
    <Modal :open="showRenameModal" title="Rename conversation" @close="showRenameModal = false">
      <form @submit.prevent="doRename">
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
        <button class="rounded-lg bg-red-600 px-4 py-2 text-sm font-bold text-white hover:bg-red-700" type="button" @click="doDelete">Delete</button>
      </div>
    </Modal>

    <!-- Copy toast -->
    <Teleport to="body">
      <Transition name="toast">
        <div v-if="showCopyToast" class="fixed bottom-6 left-1/2 z-50 -translate-x-1/2 rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white shadow-lg dark:bg-white dark:text-slate-900">
          Copied to clipboard
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import type { AIConfig, AIConversation, AIMessage, AIStreamEvent } from '@browser-server/shared-types'
import {
  createAIConversation,
  deleteAIConversation,
  getAIConfig,
  getAIConversation,
  listAIConversations,
  regenerateAIMessage,
  sendAIMessage,
  sendAIMessageStream,
  stopAIGeneration,
  updateAIConversation,
} from '../lib/api'
import Modal from './ui/Modal.vue'
import ChatSidebar from './chat/ChatSidebar.vue'
import ChatMessageList from './chat/ChatMessageList.vue'
import ChatInput from './chat/ChatInput.vue'

// ─── State ─────────────────────────────────────────────

const config = ref<AIConfig | null>(null)
const conversations = ref<AIConversation[]>([])
const activeConversation = ref<AIConversation | null>(null)
const messages = ref<AIMessage[]>([])
const draft = ref('')
const search = ref('')
const error = ref('')
const isBusy = ref(false)
const selectedProvider = ref('')
const selectedModel = ref('')
const showMobileSidebar = ref(false)
const showCopyToast = ref(false)

const messageListRef = ref<InstanceType<typeof ChatMessageList> | null>(null)
const chatInputRef = ref<InstanceType<typeof ChatInput> | null>(null)
let streamController: AbortController | null = null

// Rename
const showRenameModal = ref(false)
const renameTarget = ref<AIConversation | null>(null)
const renameTitle = ref('')

// Delete
const showDeleteModal = ref(false)
const deleteTarget = ref<AIConversation | null>(null)

// ─── Computed ──────────────────────────────────────────

const configLabel = computed(() => {
  if (!config.value) return 'Loading…'
  return config.value.enabled ? `Ready · ${selectedModel.value.split('/').pop() || 'select model'}` : 'Disabled'
})

const providerModels = computed(() => {
  if (!config.value || !selectedProvider.value) return []
  return config.value.providers[selectedProvider.value]?.models ?? []
})

const filteredConversations = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return conversations.value
  return conversations.value.filter((c) =>
    c.title.toLowerCase().includes(query) || c.model.toLowerCase().includes(query) || c.preview?.toLowerCase().includes(query)
  )
})

const canRegenerate = computed(() => {
  if (!activeConversation.value || isBusy.value) return false
  const assistantMessages = messages.value.filter((m) => m.role === 'assistant' && m.status !== 'superseded')
  return assistantMessages.length > 0
})

const selectedModelSupportsTools = computed(() => {
  const models = providerModels.value
  const current = models.find((m) => m.id === selectedModel.value)
  return current?.supports_tools ?? false
})

const visibleMessages = computed(() => {
  return messages.value.filter((m) => m.status !== 'superseded')
})

// ─── Watchers ──────────────────────────────────────────

watch(selectedProvider, () => {
  const models = providerModels.value
  if (models.length > 0 && !models.some((m) => m.id === selectedModel.value)) {
    selectedModel.value = models.find((m) => m.default)?.id || models[0].id
  }
})

// ─── Lifecycle ─────────────────────────────────────────

onMounted(async () => {
  window.addEventListener('api-token-changed', reload)
  await reload()
})

onUnmounted(() => {
  window.removeEventListener('api-token-changed', reload)
})

// ─── Core actions ──────────────────────────────────────

async function reload() {
  error.value = ''
  try {
    config.value = await getAIConfig()
    if (!config.value.enabled) return
    selectedProvider.value = config.value.default_provider || Object.keys(config.value.providers ?? {})[0] || ''
    const provider = config.value.providers?.[selectedProvider.value]
    const models = provider?.models ?? []
    selectedModel.value = provider?.default_model || models.find((m) => m.default)?.id || models[0]?.id || ''
    conversations.value = await listAIConversations(undefined, 50)
    if (conversations.value.length > 0) {
      await selectConversation(conversations.value[0].id)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load AI chat'
  }
}

async function startConversation() {
  if (!config.value?.enabled) return
  error.value = ''
  try {
    const conversation = await createAIConversation({ provider: selectedProvider.value, model: selectedModel.value })
    conversations.value = [conversation, ...conversations.value]
    activeConversation.value = conversation
    messages.value = []
    nextTick(() => chatInputRef.value?.focus())
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to create conversation'
  }
}

async function selectConversation(id: string) {
  error.value = ''
  try {
    const detail = await getAIConversation(id)
    activeConversation.value = detail.conversation
    messages.value = detail.messages ?? []
    selectedProvider.value = detail.conversation.provider
    selectedModel.value = detail.conversation.model
    nextTick(() => messageListRef.value?.scrollToBottom())
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load conversation'
  }
}

async function sendMessage(content?: string) {
  const text = content?.trim() || draft.value.trim()
  if (!text || !config.value?.enabled || !selectedProvider.value || !selectedModel.value || isBusy.value) return
  error.value = ''
  isBusy.value = true
  draft.value = ''

  try {
    if (!activeConversation.value) {
      await startConversation()
    }
    if (!activeConversation.value) return

    // Optimistic user message displayed immediately
    const tempUserId = 'temp-user-' + Date.now()
    const tempUserMsg: AIMessage = {
      id: tempUserId,
      conversation_id: activeConversation.value.id,
      role: 'user',
      content: text,
      status: 'completed',
      created_at: new Date().toISOString(),
    }
    messages.value = [...messages.value, tempUserMsg]

    // Pending assistant message for streaming
    const tempAssistantId = 'temp-assistant-' + Date.now()
    const tempAssistantMsg: AIMessage = {
      id: tempAssistantId,
      conversation_id: activeConversation.value.id,
      role: 'assistant',
      content: '',
      status: 'pending',
      created_at: new Date().toISOString(),
    }
    messages.value = [...messages.value, tempAssistantMsg]

    const useStream = config.value?.chat?.stream !== false
    const convId = activeConversation.value.id

    if (useStream) {
      // SSE streaming
      streamController = sendAIMessageStream(
        convId,
        {
          content: text,
          provider: selectedProvider.value,
          model: selectedModel.value,
          stream: true,
          tools_enabled: config.value?.tools?.enabled ?? false,
        },
        (event: AIStreamEvent) => {
          switch (event.type) {
            case 'delta': {
              const idx = messages.value.findIndex((m) => m.id === tempAssistantId)
              if (idx >= 0) {
                const msg = { ...messages.value[idx], content: messages.value[idx].content + event.content }
                messages.value = [...messages.value.slice(0, idx), msg, ...messages.value.slice(idx + 1)]
              }
              break
            }
            case 'tool_call': {
              // Display tool call as a pending tool message
              const toolMsg: AIMessage = {
                id: 'temp-tool-' + Date.now() + Math.random(),
                conversation_id: convId,
                role: 'tool',
                content: JSON.stringify({ tool: event.tool_call?.name || '', args: event.tool_call?.arguments ? JSON.parse(event.tool_call.arguments) : null, result: null }),
                status: 'pending',
                created_at: new Date().toISOString(),
              }
              // Insert before the assistant message
              const assistIdx = messages.value.findIndex((m) => m.id === tempAssistantId)
              if (assistIdx >= 0) {
                messages.value = [...messages.value.slice(0, assistIdx), toolMsg, ...messages.value.slice(assistIdx)]
              } else {
                messages.value = [...messages.value, toolMsg]
              }
              break
            }
            case 'tool_result': {
              // Update the last pending tool message with the result
              const lastPendingTool = [...messages.value].reverse().find((m) => m.role === 'tool' && m.status === 'pending')
              if (lastPendingTool) {
                try {
                  const parsed = JSON.parse(event.content)
                  const idx = messages.value.findIndex((m) => m.id === lastPendingTool.id)
                  if (idx >= 0) {
                    const updated = { ...messages.value[idx], content: event.content, status: parsed.result?.error ? 'error' as const : 'completed' as const }
                    messages.value = [...messages.value.slice(0, idx), updated, ...messages.value.slice(idx + 1)]
                  }
                } catch {
                  // skip
                }
              }
              break
            }
            case 'done': {
              const idx = messages.value.findIndex((m) => m.id === tempAssistantId)
              if (idx >= 0) {
                const msg = { ...messages.value[idx], status: 'completed' as const, id: event.message_id || messages.value[idx].id }
                messages.value = [...messages.value.slice(0, idx), msg, ...messages.value.slice(idx + 1)]
              }
              break
            }
            case 'error': {
              const idx = messages.value.findIndex((m) => m.id === tempAssistantId)
              if (idx >= 0) {
                const msg = { ...messages.value[idx], status: 'error' as const }
                messages.value = [...messages.value.slice(0, idx), msg, ...messages.value.slice(idx + 1)]
              }
              error.value = event.message || 'AI generation failed'
              break
            }
          }
        },
        (err) => {
          error.value = err.message || 'Stream connection failed'
          const idx = messages.value.findIndex((m) => m.id === tempAssistantId)
          if (idx >= 0) {
            const msg = { ...messages.value[idx], status: 'error' as const }
            messages.value = [...messages.value.slice(0, idx), msg, ...messages.value.slice(idx + 1)]
          }
          isBusy.value = false
          streamController = null
        },
      )

      // Wait briefly then consider it done after no more events
      // The actual completion is handled by the 'done' event
      const checkDone = setInterval(() => {
        const assistant = messages.value.find((m) => m.id === tempAssistantId)
        if (assistant && (assistant.status === 'completed' || assistant.status === 'error' || assistant.status === 'cancelled')) {
          clearInterval(checkDone)
          isBusy.value = false
          streamController = null
          // Auto-title on first exchange
          autoTitle(text)
          refreshConversations()
        }
      }, 200)
    } else {
      // Non-streaming fallback
      const result = await sendAIMessage(convId, {
        content: text,
        provider: selectedProvider.value,
        model: selectedModel.value,
        stream: false,
        tools_enabled: config.value?.tools?.enabled ?? false,
      })

      // Replace temp messages with real ones
      messages.value = messages.value.filter((m) => m.id !== tempUserId && m.id !== tempAssistantId)
      const newMessages = [result.user_message]
      if (result.tool_messages && result.tool_messages.length > 0) {
        newMessages.push(...result.tool_messages)
      }
      newMessages.push(result.assistant_message)
      messages.value = [...messages.value, ...newMessages]

      await autoTitle(text)
      await refreshConversations()
      isBusy.value = false
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to send message'
    isBusy.value = false
  }
}

async function autoTitle(firstMessage: string) {
  if (!activeConversation.value) return
  if (messages.value.filter((m) => m.role === 'user').length === 1 && activeConversation.value.title === 'New chat') {
    const title = firstMessage.slice(0, 60)
    activeConversation.value = await updateAIConversation(activeConversation.value.id, { title })
  }
}

async function refreshConversations() {
  conversations.value = await listAIConversations(undefined, 50)
}

async function stop() {
  if (!activeConversation.value) return
  // Abort the client-side stream
  if (streamController) {
    streamController.abort()
    streamController = null
  }
  try { await stopAIGeneration(activeConversation.value.id) } catch { /* best-effort */ }
  // Mark any pending assistant messages as cancelled
  messages.value = messages.value.map((m) =>
    m.role === 'assistant' && m.status === 'pending' ? { ...m, status: 'cancelled' as const } : m
  )
  isBusy.value = false
}

// ─── Regenerate ────────────────────────────────────────

async function regenerate() {
  if (!activeConversation.value || isBusy.value) return
  error.value = ''
  isBusy.value = true
  try {
    const result = await regenerateAIMessage(activeConversation.value.id)
    // Reload conversation to get fresh message list including superseded status
    const detail = await getAIConversation(activeConversation.value.id)
    messages.value = detail.messages ?? []
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to regenerate'
  } finally {
    isBusy.value = false
  }
}

// ─── Conversation management ───────────────────────────

function openRename(conversation: AIConversation) {
  renameTarget.value = conversation
  renameTitle.value = conversation.title
  showRenameModal.value = true
}

async function doRename() {
  if (!renameTarget.value || !renameTitle.value.trim()) return
  try {
    const updated = await updateAIConversation(renameTarget.value.id, { title: renameTitle.value.trim() })
    const idx = conversations.value.findIndex((c) => c.id === updated.id)
    if (idx >= 0) conversations.value[idx] = updated
    if (activeConversation.value?.id === updated.id) activeConversation.value = updated
    showRenameModal.value = false
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to rename'
  }
}

function confirmDelete(conversation: AIConversation) {
  deleteTarget.value = conversation
  showDeleteModal.value = true
}

async function doDelete() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  try {
    await deleteAIConversation(id)
    conversations.value = conversations.value.filter((c) => c.id !== id)
    if (activeConversation.value?.id === id) {
      activeConversation.value = null
      messages.value = []
    }
    showDeleteModal.value = false
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

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translate(-50%, 10px);
}
</style>
