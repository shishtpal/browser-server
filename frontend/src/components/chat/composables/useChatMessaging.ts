import type { AIConversation, AIMessage, AIStreamEvent } from '@browser-server/shared-types'
import { computed, ref } from 'vue'
import {
  decideAIToolCall,
  regenerateAIMessage,
  sendAIMessage,
  sendAIMessageStream,
  stopAIGeneration,
} from '../../../lib/api'

interface SendOptions {
  provider: string
  model: string
  toolsEnabled: boolean
  yoloMode: boolean
  streamEnabled: boolean
  activeTools?: string[]
}

export function useChatMessaging(
  getActiveConversation: () => AIConversation | null,
  getMessages: () => AIMessage[],
  setMessages: (msgs: AIMessage[]) => void,
) {
  const isBusy = ref(false)
  let streamController: AbortController | null = null

  const canRegenerate = computed(() => {
    const conv = getActiveConversation()
    if (!conv || isBusy.value) return false
    return getMessages().some((m) => m.role === 'assistant' && m.status !== 'superseded')
  })

  const visibleMessages = computed(() => {
    const visible = getMessages().filter((m) => m.status !== 'superseded')
    const ordered: AIMessage[] = []
    for (let index = 0; index < visible.length; index++) {
      const message = visible[index]
      if (message.role !== 'assistant') {
        ordered.push(message)
        continue
      }
      let next = index + 1
      while (next < visible.length && visible[next].role === 'tool') {
        ordered.push(visible[next])
        next++
      }
      ordered.push(message)
      index = next - 1
    }
    return ordered
  })

  async function send(
    text: string,
    conversationId: string,
    options: SendOptions,
    onDone: (conversationId: string, firstMessage: string) => Promise<void>,
    onError: (msg: string) => void,
  ) {
    isBusy.value = true
    const msgs = getMessages()

    // Optimistic user message
    const tempUserId = 'temp-user-' + Date.now()
    const tempUserMsg: AIMessage = {
      id: tempUserId,
      conversation_id: conversationId,
      role: 'user',
      content: text,
      status: 'completed',
      created_at: new Date().toISOString(),
    }

    // Pending assistant message for streaming
    const tempAssistantId = 'temp-assistant-' + Date.now()
    const tempAssistantMsg: AIMessage = {
      id: tempAssistantId,
      conversation_id: conversationId,
      role: 'assistant',
      content: '',
      status: 'pending',
      created_at: new Date().toISOString(),
    }

    setMessages([...msgs, tempUserMsg, tempAssistantMsg])

    const useStream = options.streamEnabled || options.toolsEnabled

    if (useStream) {
      streamController = sendAIMessageStream(
        conversationId,
        {
          content: text,
          provider: options.provider,
          model: options.model,
          stream: true,
          tools_enabled: options.toolsEnabled,
          yolo_mode: options.yoloMode,
          active_tools: options.activeTools,
        },
        (event: AIStreamEvent) => {
          const currentMessages = getMessages()
          switch (event.type) {
            case 'delta': {
              const idx = currentMessages.findIndex((m) => m.id === tempAssistantId)
              if (idx >= 0) {
                const msg = { ...currentMessages[idx], content: currentMessages[idx].content + event.content }
                setMessages([...currentMessages.slice(0, idx), msg, ...currentMessages.slice(idx + 1)])
              }
              break
            }
            case 'tool_call': {
              if (!event.tool_call || currentMessages.some((m) => m.tool_call_id === event.tool_call.id)) break
              let args: unknown = event.tool_call.arguments
              try { args = JSON.parse(event.tool_call.arguments) } catch { /* display malformed arguments verbatim */ }
              const toolMsg: AIMessage = {
                id: 'temp-tool-' + event.tool_call.id,
                conversation_id: conversationId,
                role: 'tool',
                content: JSON.stringify({ tool: event.tool_call.name, args, result: null, decision: event.status === 'approved' ? 'approved' : null }),
                tool_call_id: event.tool_call.id,
                status: 'pending',
                created_at: new Date().toISOString(),
              }
              const assistIdx = currentMessages.findIndex((m) => m.id === tempAssistantId)
              if (assistIdx >= 0) {
                setMessages([...currentMessages.slice(0, assistIdx), toolMsg, ...currentMessages.slice(assistIdx)])
              } else {
                setMessages([...currentMessages, toolMsg])
              }
              break
            }
            case 'tool_result': {
              const idx = currentMessages.findIndex((m) => m.tool_call_id === event.tool_call?.id)
              if (idx >= 0) {
                const updated = { ...currentMessages[idx], content: event.content, status: event.status }
                setMessages([...currentMessages.slice(0, idx), updated, ...currentMessages.slice(idx + 1)])
              }
              break
            }
            case 'done': {
              void onDone(conversationId, text).finally(() => {
                isBusy.value = false
                streamController = null
              })
              break
            }
            case 'error': {
              const idx = currentMessages.findIndex((m) => m.id === tempAssistantId)
              if (idx >= 0) {
                const msg = { ...currentMessages[idx], status: 'error' as const }
                setMessages([...currentMessages.slice(0, idx), msg, ...currentMessages.slice(idx + 1)])
              }
              onError(event.message || 'AI generation failed')
              isBusy.value = false
              streamController = null
              break
            }
          }
        },
        (err) => {
          const currentMessages = getMessages()
          const idx = currentMessages.findIndex((m) => m.id === tempAssistantId)
          if (idx >= 0) {
            const msg = { ...currentMessages[idx], status: 'error' as const }
            setMessages([...currentMessages.slice(0, idx), msg, ...currentMessages.slice(idx + 1)])
          }
          onError(err.message || 'Stream connection failed')
          isBusy.value = false
          streamController = null
        },
      )
    } else {
      // Non-streaming fallback
      const result = await sendAIMessage(conversationId, {
        content: text,
        provider: options.provider,
        model: options.model,
        stream: false,
        tools_enabled: options.toolsEnabled,
        yolo_mode: options.yoloMode,
        active_tools: options.activeTools,
      })

      const currentMessages = getMessages().filter((m) => m.id !== tempUserId && m.id !== tempAssistantId)
      const newMessages: AIMessage[] = [result.user_message]
      if (result.tool_messages && result.tool_messages.length > 0) {
        newMessages.push(...result.tool_messages)
      }
      newMessages.push(result.assistant_message)
      setMessages([...currentMessages, ...newMessages])

      await onDone(conversationId, text)
      isBusy.value = false
    }
  }

  async function decideToolCall(callId: string, approved: boolean, onError: (msg: string) => void) {
    const conv = getActiveConversation()
    if (!conv) return
    const currentMessages = getMessages()
    const index = currentMessages.findIndex((message) => message.tool_call_id === callId)
    if (index < 0) return
    const original = currentMessages[index]
    try {
      const content = JSON.parse(original.content)
      const updated = { ...original, content: JSON.stringify({ ...content, decision: approved ? 'approved' : 'rejected' }) }
      setMessages([...currentMessages.slice(0, index), updated, ...currentMessages.slice(index + 1)])
      await decideAIToolCall(conv.id, callId, approved)
    } catch (err) {
      const msgs = getMessages()
      const current = msgs.findIndex((message) => message.tool_call_id === callId)
      if (current >= 0) setMessages([...msgs.slice(0, current), original, ...msgs.slice(current + 1)])
      onError(err instanceof Error ? err.message : 'Failed to submit tool decision')
    }
  }

  async function regenerate(conversationId: string, onError: (msg: string) => void): Promise<void> {
    if (isBusy.value) return
    isBusy.value = true
    try {
      await regenerateAIMessage(conversationId)
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Failed to regenerate')
    } finally {
      isBusy.value = false
    }
  }

  async function stop(conversationId: string) {
    if (streamController) {
      streamController.abort()
      streamController = null
    }
    try { await stopAIGeneration(conversationId) } catch { /* best-effort */ }
    const currentMessages = getMessages()
    setMessages(currentMessages.map((m) =>
      m.role === 'assistant' && m.status === 'pending' ? { ...m, status: 'cancelled' as const } : m
    ))
    isBusy.value = false
  }

  function cleanup() {
    streamController?.abort()
    streamController = null
  }

  return {
    isBusy,
    canRegenerate,
    visibleMessages,
    send,
    decideToolCall,
    regenerate,
    stop,
    cleanup,
  }
}
