import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue';
import { useLocalStorage } from '@vueuse/core';
import {
  deleteAIMessage,
  getAIConfig,
  getAIConversation,
  getAIImageAttachmentBlob,
} from '../../../lib/api';
import type { AIAttachmentSummary, AIConversation, AIMessage } from '@browser-server/shared-types';
import { filenameSafe, normalizePromptContent } from '../chatFormat';
import { useChatConfig } from './useChatConfig';
import { useChatConversations } from './useChatConversations';
import { useChatMessaging } from './useChatMessaging';
import { useChatRouting } from './useChatRouting';
import { useSpeechPlayback } from './useSpeechPlayback';
import { deriveToolCallEntries } from '../messages/messageTools';
import type { ChatToast } from '../ChatPageToast.vue';

export type PendingConversationAction = {
  kind: 'rename' | 'archive' | 'restore' | 'delete';
  conversation: AIConversation;
  title: string;
} | null;

/**
 * Orchestrates the whole Chat page: config + conversations + messaging wiring,
 * panel/modal/toast visibility, conversation routing (/chat/<id>), message
 * actions (send/append/regenerate/stop/branch/delete/export), and lifecycle —
 * so ChatPage.vue stays a thin template.
 */
export function useChatPage() {
  /* ------------------------------ config layer ------------------------------ */

  const chatConfig = useChatConfig();
  const {
    config,
    selectedProvider,
    selectedModel,
    selectedProfile,
    activeSkills,
    yoloMode,
    userToolsEnabled,
    includeAllToolDefinitions,
    showThinking,
    toolsEnabled,
    activeTools,
    attachmentsConfig,
    selectedModelSupportsTools,
    selectedModelSupportsVision,
    selectedProviderURL,
    setActiveSkills,
    initFromConfig,
  } = chatConfig;

  /* --------------------------- conversations layer -------------------------- */

  const conv = useChatConversations();
  const {
    conversations,
    activeConversation,
    messages,
    search,
    filteredConversations,
    loadConversations,
    createConversation,
    forkConversation,
    selectConversation,
    refreshConversation,
    autoTitle,
    loadArchivedConversations,
  } = conv;

  /* ----------------------------- messaging layer ---------------------------- */

  const messaging = useChatMessaging(
    () => activeConversation.value,
    () => messages.value,
    (msgs) => {
      messages.value = msgs;
    },
  );
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
  } = messaging;

  const speech = useSpeechPlayback();
  const {
    ttsAvailable,
    speakingMessageId,
    speakingBusyId,
    loadTTSAvailability,
    speak,
    cleanup: cleanupSpeech,
  } = speech;

  /* --------------------------- local page state ----------------------------- */

  const draft = ref('');
  const error = ref('');

  const showMobileSidebar = ref(false);
  const showToolsPanel = ref(true);
  const showMemoryExplorer = ref(false);
  const showNewConversationModal = ref(false);
  const showPromptManager = ref(false);
  const showAttachmentGallery = ref(false);
  const showVoiceModal = ref(false);

  // Persisted display preferences
  const chatFontFamily = useLocalStorage('bs.ai.chatFontFamily', 'system-ui');
  const chatFontSize = useLocalStorage('bs.ai.chatFontSize', 14);
  const rawToolOutput = useLocalStorage<boolean | null>('bs.ai.rawToolOutput', null, {
    serializer: {
      read: (v) => (v == null ? null : v === 'true' ? true : v === 'false' ? false : null),
      write: (v) => String(v),
    },
  });

  // Toast notifications (auto-dismiss)
  const toast = ref<ChatToast | null>(null);
  let toastTimer: ReturnType<typeof setTimeout> | null = null;
  function flashToast(kind: ChatToast['kind'], ms = 2000, message?: string) {
    if (toastTimer) clearTimeout(toastTimer);
    toast.value = { kind, id: Date.now(), message };
    toastTimer = setTimeout(() => (toast.value = null), ms);
  }

  // Template refs (typed loosely to avoid circular import of component types)
  const messageListRef = ref<{ scrollToBottom: () => void } | null>(null);
  const chatInputRef = ref<{ focus: () => void } | null>(null);

  /* ------------------------------ derived state ------------------------------ */

  const profileLocked = computed(() =>
    activeConversation.value ? messages.value.length > 0 : false,
  );

  const gridClass = computed(() =>
    showToolsPanel.value
      ? 'lg:grid-cols-[300px_minmax(0,1fr)_auto]'
      : 'lg:grid-cols-[300px_minmax(0,1fr)]',
  );

  const chatFontStyle = computed(() => ({
    fontFamily: chatFontFamily.value,
    fontSize: chatFontSize.value + 'px',
  }));

  const toolCallEntries = computed(() => deriveToolCallEntries(messages.value));

  /* ---------------------- conversation routing (/chat/<id>) ------------------ */

  const { conversationIdFromLocation, updateConversationURL } = useChatRouting(async (id) => {
    if (id === activeConversation.value?.id) return;
    await handleSelectConversation(id, false);
  });

  /* -------------------------------- lifecycle ------------------------------- */

  async function reload() {
    error.value = '';
    try {
      const cfg = await getAIConfig();
      initFromConfig(cfg);
      void loadTTSAvailability();
      if (!cfg.enabled) return;
      await loadConversations();
      const requestedID = conversationIdFromLocation();
      if (requestedID) {
        try {
          await handleSelectConversation(requestedID, false);
          return;
        } catch {
          // Deleted/unavailable shared link → fall back to the latest conversation.
        }
      }
      if (conversations.value.length > 0) {
        await handleSelectConversation(conversations.value[0].id);
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load AI chat';
    }
  }

  function onTokenChanged() {
    reload();
  }

  onMounted(async () => {
    window.addEventListener('api-token-changed', onTokenChanged);
    await reload();
  });

  onUnmounted(() => {
    window.removeEventListener('api-token-changed', onTokenChanged);
    cleanup();
    cleanupSpeech();
  });

  /* --------------------------- conversation actions -------------------------- */

  function startConversation() {
    if (!config.value?.enabled) return;
    showNewConversationModal.value = true;
  }

  async function handleNewConversationCreate(result: {
    provider: string;
    model: string;
    profile: string;
    skills: string[];
  }) {
    showNewConversationModal.value = false;
    error.value = '';
    try {
      selectedProvider.value = result.provider;
      selectedModel.value = result.model;
      selectedProfile.value = result.profile;
      setActiveSkills(result.skills);

      const conversation = await createConversation(
        result.provider,
        result.model,
        result.profile || undefined,
      );
      updateConversationURL(conversation.id);
      nextTick(() => chatInputRef.value?.focus());
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to create conversation';
    }
  }

  async function handleSelectConversation(id: string, updateURL = true) {
    error.value = '';
    try {
      const { provider, model } = await selectConversation(id);
      selectedProvider.value = provider;
      selectedModel.value = model;
      // Profile is conversation-owned (locked once the conversation has messages).
      selectedProfile.value = activeConversation.value?.profile || '';
      // Restore per-conversation skill selection.
      setActiveSkills(activeConversation.value?.skills ?? []);
      if (updateURL) updateConversationURL(id);
      nextTick(() => messageListRef.value?.scrollToBottom());
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load conversation';
    }
  }

  /** One pending action modal replaces 4 separate confirm dialogs. */
  const pendingAction = ref<PendingConversationAction>(null);

  function requestRename(conversation: AIConversation) {
    pendingAction.value = { kind: 'rename', conversation, title: conversation.title };
  }
  function requestArchive(conversation: AIConversation) {
    pendingAction.value = { kind: 'archive', conversation, title: conversation.title };
  }
  function requestRestore(conversation: AIConversation) {
    pendingAction.value = { kind: 'restore', conversation, title: conversation.title };
  }
  function requestDelete(conversation: AIConversation) {
    pendingAction.value = { kind: 'delete', conversation, title: conversation.title };
  }
  function cancelAction() {
    pendingAction.value = null;
  }

  async function runPendingAction() {
    const pending = pendingAction.value;
    if (!pending) return;
    error.value = '';
    try {
      if (pending.kind === 'rename') {
        conv.renameTarget.value = pending.conversation;
        conv.renameTitle.value = pending.title;
        await conv.doRename();
      } else if (pending.kind === 'archive') {
        conv.archiveTarget.value = pending.conversation;
        await conv.doArchive();
        if (!activeConversation.value) updateConversationURL(null);
        await loadArchivedConversations();
      } else if (pending.kind === 'restore') {
        conv.restoreTarget.value = pending.conversation;
        await conv.doRestore();
        await loadArchivedConversations();
        await loadConversations();
      } else {
        conv.deleteTarget.value = pending.conversation;
        await conv.doDelete();
        if (!activeConversation.value) updateConversationURL(null);
      }
      pendingAction.value = null;
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Action failed';
    }
  }

  /* ----------------------------- message actions ----------------------------- */

  async function sendMessage(content?: string) {
    const text = content?.trim() || draft.value.trim();
    if (!config.value?.enabled || !selectedProvider.value || !selectedModel.value || isBusy.value)
      return;
    if (!text && stagedAttachments.value.length === 0) return;
    error.value = '';
    draft.value = '';

    try {
      if (!activeConversation.value) startConversation();
      if (!activeConversation.value) return;

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
          await refreshConversation(conversationId);
          await autoTitle(firstMessage);
          await loadConversations();
        },
        (msg) => {
          error.value = msg;
        },
      );
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to send message';
    }
  }

  async function appendContext(content: string) {
    const text = content.trim();
    if (!text || !activeConversation.value) return;
    const conversationId = activeConversation.value.id;
    const draftAtSubmit = draft.value;
    error.value = '';
    const appended = await append(text, conversationId, (msg) => {
      error.value = msg;
    });
    if (
      appended &&
      activeConversation.value?.id === conversationId &&
      draft.value === draftAtSubmit
    ) {
      draft.value = '';
    }
  }

  async function handleToolDecision(callId: string, approved: boolean, comment: string) {
    await decideToolCall(callId, approved, comment, (msg) => {
      error.value = msg;
    });
  }

  async function handleRegenerate() {
    if (!activeConversation.value) return;
    error.value = '';
    await regenerate(activeConversation.value.id, (msg) => {
      error.value = msg;
    });
    if (activeConversation.value) {
      try {
        const detail = await getAIConversation(activeConversation.value.id);
        messages.value = detail.messages ?? [];
      } catch {
        /* messages refresh on next action */
      }
    }
  }

  async function handleStop() {
    if (!activeConversation.value) return;
    await stop(activeConversation.value.id);
  }

  async function copyMessage(content: string) {
    try {
      await navigator.clipboard.writeText(content);
      flashToast('copy');
    } catch {
      /* clipboard unavailable — silent */
    }
  }

  async function deleteMessage(messageId: string) {
    if (!activeConversation.value || isBusy.value || messageId.startsWith('temp-')) return;
    if (!window.confirm('Delete this message? This action cannot be undone.')) return;

    error.value = '';
    try {
      await deleteAIMessage(activeConversation.value.id, messageId);
      messages.value = messages.value.filter((message) => message.id !== messageId);
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete message';
    }
  }

  async function handleSpeak(payload: { messageId: string; content: string }) {
    const err = await speak(payload.messageId, payload.content);
    if (err) flashToast('error', 3200, err);
  }

  async function handleBranch(messageId: string) {
    if (!activeConversation.value || isBusy.value || messageId.startsWith('temp-')) return;
    error.value = '';
    try {
      const source = activeConversation.value;
      const forked = await forkConversation(source.id, messageId);
      // Carry the source conversation's runtime selection into the new branch.
      selectedProvider.value = forked.provider;
      selectedModel.value = forked.model;
      selectedProfile.value = forked.profile || '';
      setActiveSkills(forked.skills ?? []);
      await loadConversations();
      updateConversationURL(forked.id);
      flashToast('branch', 2200);
      nextTick(() => {
        messageListRef.value?.scrollToBottom();
        chatInputRef.value?.focus();
      });
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to branch conversation';
    }
  }

  /* ------------------------------ input helpers ------------------------------ */

  function useSuggestion(text: string | { content?: string }) {
    draft.value = typeof text === 'string' ? text : (text.content ?? '');
    nextTick(() => chatInputRef.value?.focus());
  }

  function useVoiceTranscript(text: string) {
    const existing = draft.value.trimEnd();
    draft.value = existing ? `${existing} ${text.trim()}` : text.trim();
    showVoiceModal.value = false;
    nextTick(() => chatInputRef.value?.focus());
  }

  function applyPromptFromManager(prompt: unknown) {
    const text = normalizePromptContent(prompt);
    if (!text) return;
    draft.value = text;
    showPromptManager.value = false;
    nextTick(() => chatInputRef.value?.focus());
  }

  async function addImages(files: File[]) {
    if (!activeConversation.value) return;
    await addImageAttachments(activeConversation.value.id, files);
  }

  async function removeStagedImage(id: string) {
    if (!activeConversation.value) return;
    await removeStagedAttachment(activeConversation.value.id, id);
  }

  async function handleReuseAttachment(att: AIAttachmentSummary) {
    if (!activeConversation.value) {
      error.value = 'Select or create a conversation before reusing an attachment.';
      return;
    }
    error.value = '';
    try {
      const blob = await getAIImageAttachmentBlob(att.conversation_id, att.id);
      const file = new File([blob], att.filename, { type: att.content_type });
      await addImages([file]);
      showAttachmentGallery.value = false;
      nextTick(() => chatInputRef.value?.focus());
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to reuse attachment';
    }
  }

  /* ------------------------------- export ------------------------------------ */

  function downloadConversation() {
    const conversation = activeConversation.value;
    if (!conversation) return;

    const title = conversation.title.replace(/[\r\n]+/g, ' ').trim() || 'AI conversation';
    const sections = visibleMessages.value.map((message) => {
      const role = message.role.charAt(0).toUpperCase() + message.role.slice(1);
      return `## ${role}\n\n${message.content.trim()}`;
    });
    const markdown =
      [
        `# ${title}`,
        `- **Provider:** ${conversation.provider}`,
        `- **Model:** ${conversation.model}`,
        `- **Created:** ${conversation.created_at}`,
        '',
        '---',
        '',
        ...sections.flatMap((section) => [section, '']),
      ]
        .join('\n')
        .trimEnd() + '\n';

    const blobUrl = URL.createObjectURL(
      new Blob([markdown], { type: 'text/markdown;charset=utf-8' }),
    );
    const link = document.createElement('a');
    link.href = blobUrl;
    link.download = `${filenameSafe(title)}.md`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(blobUrl);
  }

  return {
    // passthroughs
    chats: chatConfig,
    conversationsApi: conv,
    messaging,
    // state
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
    // derived
    profileLocked,
    gridClass,
    chatFontStyle,
    toolCallEntries,
    // flows
    reload,
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
    ttsAvailable,
    speakingMessageId,
    speakingBusyId,
    copyMessage,
    deleteMessage,
    handleSpeak,
    handleBranch,
    useSuggestion,
    useVoiceTranscript,
    applyPromptFromManager,
    addImages,
    removeStagedImage,
    handleReuseAttachment,
    downloadConversation,
  };
}
