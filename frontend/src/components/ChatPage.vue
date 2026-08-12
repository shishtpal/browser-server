<template>
  <div
    class="chat-shell grid h-[calc(100vh-57px)] max-w-full grid-cols-1 overflow-hidden bg-white text-slate-900 dark:bg-slate-950 dark:text-white"
    :class="gridClass"
    :style="chatFontStyle"
  >
    <!-- Desktop sidebar -->
    <ChatSidebar
      class="hidden lg:flex"
      :conversations="filteredConversations"
      :active-id="activeConversation?.id ?? null"
      :search="search"
      :status-label="configLabel"
      :disabled="!config?.enabled || isBusy"
      :archived-conversations="archivedConversations"
      :show-archived="showArchived"
      @new="startConversation"
      @select="handleSelectConversation"
      @rename="requestRename"
      @delete="requestDelete"
      @archive="requestArchive"
      @restore="requestRestore"
      @toggle-archived="toggleArchived"
      @update:search="search = $event"
    />

    <!-- Main panel -->
    <section class="flex h-full min-h-0 flex-col overflow-hidden">
      <ChatTopBar
        :profiles="profiles"
        :selected-profile="selectedProfile"
        :profile-locked="profileLocked"
        :skills="skills"
        :active-skills="activeSkills"
        :provider-names="providerNames"
        :selected-provider="selectedProvider"
        :selected-model="selectedModel"
        :models="providerModels"
        :supports-tools="selectedModelSupportsTools"
        :tools-enabled="toolsEnabled"
        :yolo-mode="yoloMode"
        :provider-url="selectedProviderURL"
        :disabled="!config?.enabled || isBusy"
        :title="activeConversation?.title"
        :download-disabled="!activeConversation"
        :show-tools-panel="showToolsPanel"
        :show-memory-explorer="showMemoryExplorer"
        :show-prompt-manager="showPromptManager"
        :show-attachment-gallery="showAttachmentGallery"
        @toggle-sidebar="showMobileSidebar = true"
        @update:selected-profile="selectedProfile = $event"
        @update:selected-provider="selectedProvider = $event"
        @update:selected-model="selectedModel = $event"
        @update:yolo-mode="yoloMode = $event"
        @update:active-skills="setActiveSkills($event)"
        @download="downloadConversation"
        @toggle-tools-panel="showToolsPanel = !showToolsPanel"
        @toggle-memory-explorer="showMemoryExplorer = !showMemoryExplorer"
        @toggle-prompt-manager="showPromptManager = !showPromptManager"
        @toggle-attachment-gallery="showAttachmentGallery = !showAttachmentGallery"
      />

      <ErrorBanner
        v-if="error"
        :message="error"
        :on-retry="() => (error = '')"
        class="mx-3 mt-3 shrink-0 sm:mx-4"
      />

      <!-- AI disabled state -->
      <ChatDisabledState v-if="config && !config.enabled" />

      <!-- Chat area -->
      <template v-else>
        <ChatMessageList
          ref="messageListRef"
          :messages="visibleMessages"
          :loading="isBusy"
          :show-thinking="showThinking"
          @suggestion="useSuggestion"
          @copy="copyMessage"
          @delete="deleteMessage"
          @branch="handleBranch"
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
          :can-append="canAppend"
          :is-appending="isAppending"
          :user-id="currentUserId"
          :conversation-id="activeConversation?.id"
          :attachments-config="attachmentsConfig"
          :supports-vision="selectedModelSupportsVision"
          :staged-images="stagedAttachments"
          @send="sendMessage"
          @append="appendContext"
          @stop="handleStop"
          @voice="showVoiceModal = true"
          @select-prompt="useSuggestion"
          @add-images="addImages"
          @remove-image="removeStagedImage"
        />
      </template>
    </section>

    <!-- Right tools panel (desktop) -->
    <ChatToolsPanel
      v-if="showToolsPanel"
      class="hidden lg:flex"
      :tools-enabled="userToolsEnabled"
      :model-supports-tools="selectedModelSupportsTools"
      :yolo-mode="yoloMode"
      :include-all-tool-definitions="includeAllToolDefinitions"
      :available-tools="availableTools"
      :tools-by-category="toolsByCategory"
      :disabled-tools="disabledTools"
      :tool-calls="toolCallEntries"
      :mcp="mcp"
      :font-family="chatFontFamily"
      :font-size="chatFontSize"
      v-model:show-thinking="showThinking"
      :raw-tool-output="rawToolOutput"
      @close="showToolsPanel = false"
      @update:tools-enabled="userToolsEnabled = $event"
      @update:yolo-mode="yoloMode = $event"
      @update:include-all-tool-definitions="includeAllToolDefinitions = $event"
      @update:font-family="chatFontFamily = $event"
      @update:font-size="chatFontSize = $event"
      @update:raw-tool-output="rawToolOutput = $event"
      @toggle-tool="toggleTool"
    />

    <!-- Mobile sidebar drawer -->
    <ChatMobileDrawer
      :open="showMobileSidebar"
      :conversations="filteredConversations"
      :active-id="activeConversation?.id ?? null"
      :disabled="!config?.enabled || isBusy"
      :archived-conversations="archivedConversations"
      :show-archived="showArchived"
      @close="showMobileSidebar = false"
      @new="
        startConversation();
        showMobileSidebar = false;
      "
      @select="
        handleSelectConversation($event);
        showMobileSidebar = false;
      "
      @rename="requestRename"
      @delete="requestDelete"
      @archive="requestArchive"
      @restore="requestRestore"
      @toggle-archived="toggleArchived"
    />

    <!-- Voice typing modal -->
    <ChatVoiceTypingModal
      :open="showVoiceModal"
      @close="showVoiceModal = false"
      @use="useVoiceTranscript"
    />

    <!-- Rename / archive / restore / delete -->
    <ConversationActionModal
      :open="pendingAction !== null"
      :kind="pendingAction?.kind ?? null"
      v-model="pendingActionTitle"
      :conversation-title="pendingAction?.conversation.title"
      @confirm="runPendingAction"
      @cancel="cancelAction"
    />

    <!-- Toasts -->
    <ChatPageToast :toast="toast" />

    <!-- Memory explorer -->
    <ChatMemoryExplorer
      :open="showMemoryExplorer"
      :conversation-id="activeConversation?.id ?? ''"
      :messages="messages"
      @close="showMemoryExplorer = false"
      @updated="messages = $event"
    />

    <!-- Prompt manager -->
    <PromptManager
      :open="showPromptManager"
      :user-id="currentUserId"
      @select="applyPromptFromManager"
      @close="showPromptManager = false"
    />

    <!-- Attachment library -->
    <ChatAttachmentGallery
      :open="showAttachmentGallery"
      @close="showAttachmentGallery = false"
      @reuse="handleReuseAttachment"
    />

    <!-- New conversation -->
    <ChatNewConversationModal
      :open="showNewConversationModal"
      :profiles="profiles"
      :provider-names="providerNames"
      :providers="config?.providers ?? {}"
      :skills="skills"
      :default-provider="selectedProvider"
      :default-model="selectedModel"
      :default-profile="profileLocked ? '' : selectedProfile"
      :default-skills="activeSkills"
      @close="showNewConversationModal = false"
      @create="handleNewConversationCreate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useUser } from '../composables/useUser';
import { useChatPage } from './chat/composables/useChatPage';
import ChatSidebar from './chat/ChatSidebar.vue';
import ChatTopBar from './chat/ChatTopBar.vue';
import ChatMessageList from './chat/ChatMessageList.vue';
import ChatInput from './chat/ChatInput.vue';
import ChatRegenerateButton from './chat/ChatRegenerateButton.vue';
import ChatMobileDrawer from './chat/ChatMobileDrawer.vue';
import ChatDisabledState from './chat/ChatDisabledState.vue';
import ChatPageToast from './chat/ChatPageToast.vue';
import ChatToolsPanel from './chat/ChatToolsPanel.vue';
import ChatMemoryExplorer from './chat/ChatMemoryExplorer.vue';
import ChatNewConversationModal from './chat/ChatNewConversationModal.vue';
import ChatAttachmentGallery from './chat/ChatAttachmentGallery.vue';
import ChatVoiceTypingModal from './chat/ChatVoiceTypingModal.vue';
import ConversationActionModal from './chat/ConversationActionModal.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import PromptManager from './prompts/PromptManager.vue';

const { currentUserId } = useUser();

const page = useChatPage();

// Config passthroughs
const {
  config,
  configLabel,
  profiles,
  skills,
  mcp,
  activeSkills,
  showThinking,
  yoloMode,
  userToolsEnabled,
  includeAllToolDefinitions,
  disabledTools,
  providerModels,
  selectedModelSupportsTools,
  selectedModelSupportsVision,
  toolsEnabled,
  availableTools,
  toolsByCategory,
  selectedProvider,
  selectedModel,
  selectedProfile,
  selectedProviderURL,
  attachmentsConfig,
  toggleTool,
  setActiveSkills,
} = page.chats;

// Conversations passthroughs
const {
  activeConversation,
  messages,
  search,
  filteredConversations,
  archivedConversations,
  showArchived,
  toggleArchived,
} = page.conversationsApi;

// Messaging passthroughs
const { isBusy, canAppend, isAppending, canRegenerate, visibleMessages, stagedAttachments } =
  page.messaging;

// Page orchestration
const {
  draft,
  error,
  showMobileSidebar,
  showToolsPanel,
  showMemoryExplorer,
  showNewConversationModal,
  showPromptManager,
  showAttachmentGallery,
  showVoiceModal,
  chatFontFamily,
  chatFontSize,
  rawToolOutput,
  toast,
  messageListRef,
  chatInputRef,
  profileLocked,
  gridClass,
  chatFontStyle,
  toolCallEntries,
  startConversation,
  handleNewConversationCreate,
  handleSelectConversation,
  pendingAction,
  requestRename,
  requestArchive,
  requestRestore,
  requestDelete,
  cancelAction,
  runPendingAction,
  sendMessage,
  appendContext,
  handleToolDecision,
  handleRegenerate,
  handleStop,
  copyMessage,
  deleteMessage,
  handleBranch,
  useSuggestion,
  useVoiceTranscript,
  applyPromptFromManager,
  addImages,
  removeStagedImage,
  handleReuseAttachment,
  downloadConversation,
} = page;

const providerNames = computed(() => Object.keys(config.value?.providers ?? {}));

/** Two-way proxy so the rename draft edits the pending action's title. */
const pendingActionTitle = computed({
  get: () => pendingAction.value?.title ?? '',
  set: (v: string) => {
    if (pendingAction.value) pendingAction.value.title = v;
  },
});
</script>
