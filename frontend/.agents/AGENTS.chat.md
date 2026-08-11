# AGENTS.chat.md — AI Chat Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers the AI chat UI in `components/chat/`.

The AI chat UI is fully modular, split into focused sub-components and composables. The list of providers and models is read from `bs-ai-models.json` via `/api/ai/config`; behavior toggles such as `tools.allowed`, `memory`, and `skills` come from `bs-ai-config.json`. Optional MCP servers are connected only by the backend; the frontend receives sanitized status plus their runtime tool names/categories through `/api/ai/config` and must use the same selection, approval, and SSE paths as built-in tools.

```
../components/chat/
├── ChatTopBar.vue            # Provider/model selects, YOLO mode toggle, prompt-manager + panel buttons
├── ChatSidebar.vue           # Desktop conversation list with search and actions
├── ChatMobileDrawer.vue      # Mobile drawer wrapping the conversation list
├── ConversationList.vue      # Conversation rows (active/archived), search + filter
├── ConversationActionModal.vue, ArchivedConversationsList.vue
├── ChatNewConversationModal.vue   # New conversation (provider/model/profile/skills/attachment)
├── ChatQuestionForm.vue      # "Ask a question" quick form
├── ChatMessageList.vue       # Scrollable message container, empty-state suggestions, typing indicator
├── ChatInput.vue             # Auto-resizing textarea w/ send/stop, staged images, voice button
├── ChatRegenerateButton.vue  # Regenerate the last assistant response
├── ChatAttachmentGallery.vue # Image attachments in the active conversation
├── ChatThinkingBlock.vue     # Collapsible "thinking" block on assistant messages
├── ChatToolsPanel.vue        # Right-side panel: Tools / History / Settings tabs (resizable via usePanelResize)
├── ChatMemoryExplorer.vue    # Memory graph explorer modal (see composables/useMemoryExplorer.ts)
├── ChatVoiceTypingModal.vue  # Web-Speech voice input modal (see composables/useVoiceTyping.ts)
├── ChatPageToast.vue         # In-page toast surface (copy, export, errors)
├── ChatDisabledState.vue     # Placeholder when bs-ai-config.json or bs-ai-models.json is missing
├── chatFormat.ts             # Shared constants/formatters (filenameSafe, normalizePromptContent)
├── input/
│   ├── StagedImageStrip.vue  # Pending image attachments before send
│   └── stagedImages.ts       # Staged-image state helpers
├── memory/
│   ├── MemoryMessageCard.vue # Renders a memory write/recall tool message
│   └── MemoryToolContent.vue # Raw memory tool payload rendering
├── messages/
│   ├── ChatMessageItem.vue   # One message row (user/assistant/tool)
│   ├── UserBubble.vue, AssistantBubble.vue, ToolMessageCard.vue, ToolApprovalCard.vue
│   ├── BubbleActions.vue     # Copy, branch, export, delete actions on a bubble
│   ├── ImageAttachmentStrip.vue
│   └── messageTools.ts       # Tool-call entry derivation helpers
└── panel/
    ├── ToolsTab.vue, HistoryTab.vue, SettingsTab.vue   # ChatToolsPanel tabs
    └── usePanelResize.ts     # Pointer-driven panel width
```

`composables/` (page-scoped; no other page imports these):

```
composables/
├── useChatPage.ts            # Orchestrates the whole page: wiring, modals, toasts, routing
├── useChatConfig.ts          # AI config, provider/model/profile/skills state, YOLO mode persistence
├── useChatConversations.ts   # Conversation CRUD, fork/branch, search/filter, archive, rename/delete modals
├── useChatMessaging.ts       # Send, stream (SSE), tool decisions, regenerate, stop
├── useChatRouting.ts         # /chat/<id> routing + deep links
├── useMemoryExplorer.ts      # Memory graph explorer state
├── usePromptMode.ts          # Prompt-library mode (see AGENTS.prompts.md)
└── useVoiceTyping.ts         # Voice typing modal + Web Speech plumbing
```

## Prompt library integration

The prompt manager (`components/prompts/`, app-wide composables `usePrompts` + `usePromptManager`) is embedded in both `ChatPage.vue` and `ImagePage.vue`; `ChatTopBar.vue` exposes the toggle. See [`AGENTS.prompts.md`](./AGENTS.prompts.md).

## Branch a conversation from any message

Every user and assistant bubble exposes a **Branch** action (git-branch icon). It calls `forkConversation(sourceId, messageId)` (in `useChatConversations.ts`), which POSTs to `POST /api/ai/conversations/{id}/fork` with `{ message_id }`. The backend copies every settled message up to **and including** the selected one into a brand-new conversation that inherits the source provider / model / profile / skills, then the UI activates the new branch so the user can keep chatting from that point. The source conversation is left untouched.

## Markdown rendering

Markdown rendering lives in the shared package `@browser-server/shared-markdown` (`shared/browser-markdown`), consumed via `renderMarkdown` / `typesetMath` in `messages/AssistantBubble.vue`. MathJax typesetting is opt-in per assistant message (Sigma toggle in the bubble actions, off by default); `renderMarkdown` accepts `{ math: false }` to render `$…$` literally. Because the renderer's Tailwind class strings live outside the frontend root, `src/styles/global.css` includes an explicit `@source '../../../shared/browser-markdown/src'` — keep it if the package moves.

`ChatPage.vue` composes these pieces and delegates business logic to the composables, keeping the top-level component focused on wiring.
