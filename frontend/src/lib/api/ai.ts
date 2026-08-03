import type {
  AIConfig,
  AIVoiceConfig,
  AIConversation,
  AIConversationDetail,
  AIImageAttachment,
  AIMessage,
  AIStreamEvent,
  AIToolDecisionResponse,
  AppendAIMessageInput,
  CreateAIConversationInput,
  SendAIMessageInput,
  SendAIMessageResponse,
  StopAIGenerationResponse,
  UpdateAIConversationInput,
} from '@browser-server/shared-types'
import { API_BASE, authHeaders, client } from './client'

// ─── AI Chat ────────────────────────────────────────────

export function getAIConfig(): Promise<AIConfig> {
  return client.getAIConfig()
}

export function getAIVoiceConfig(): Promise<AIVoiceConfig> {
  return client.getAIVoiceConfig()
}

export function listAIConversations(query?: string, limit?: number): Promise<AIConversation[]> {
  return client.listAIConversations(query, limit)
}

export function createAIConversation(data: CreateAIConversationInput = {}): Promise<AIConversation> {
  return client.createAIConversation(data)
}

export function forkAIConversation(id: string, messageId: string): Promise<AIConversation> {
  return client.forkAIConversation(id, { message_id: messageId })
}

export function getAIConversation(id: string): Promise<AIConversationDetail> {
  return client.getAIConversation(id)
}

export function updateAIConversation(id: string, data: UpdateAIConversationInput): Promise<AIConversation> {
  return client.updateAIConversation(id, data)
}

export function deleteAIConversation(id: string): Promise<void> {
  return client.deleteAIConversation(id)
}

export function sendAIMessage(id: string, data: SendAIMessageInput): Promise<SendAIMessageResponse> {
  return client.sendAIMessage(id, data)
}

export function uploadAIImageAttachment(id: string, file: Blob, filename?: string): Promise<AIImageAttachment> {
  return client.uploadAIImageAttachment(id, file, filename)
}

export function deleteAIImageAttachment(id: string, attachmentId: string): Promise<void> {
  return client.deleteAIImageAttachment(id, attachmentId)
}

export function getAIImageAttachmentUrl(id: string, attachmentId: string): string {
  return client.getAIImageAttachmentUrl(id, attachmentId)
}

export function appendAIMessage(id: string, data: AppendAIMessageInput): Promise<AIMessage> {
  return client.appendAIMessage(id, data)
}

export function sendAIMessageStream(
  id: string,
  data: SendAIMessageInput,
  onEvent: (event: AIStreamEvent) => void,
  onError?: (err: Error) => void,
): AbortController {
  return client.sendAIMessageStream(id, data, onEvent, onError)
}

export function regenerateAIMessage(id: string): Promise<SendAIMessageResponse> {
  return client.regenerateAIMessage(id)
}

export function decideAIToolCall(id: string, callId: string, approved: boolean, comment?: string): Promise<AIToolDecisionResponse> {
  return client.decideAIToolCall(id, callId, approved, comment)
}

export function stopAIGeneration(id: string): Promise<StopAIGenerationResponse> {
  return client.stopAIGeneration(id)
}

export function updateAIMessage(conversationId: string, messageId: string, data: { content: string }): Promise<AIMessage> {
  return client.updateAIMessage(conversationId, messageId, data)
}

export function deleteAIMessage(conversationId: string, messageId: string): Promise<void> {
  return client.deleteAIMessage(conversationId, messageId)
}

// ─── AI Chat Archiving ──────────────────────────────────────────────────────

export function archiveAIConversation(id: string): Promise<void> {
  return fetch(`${API_BASE}/api/ai/conversations/${id}/archive`, { method: 'POST', headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error('Failed to archive conversation')
  })
}

export function restoreAIConversation(id: string): Promise<void> {
  return fetch(`${API_BASE}/api/ai/conversations/${id}/restore`, { method: 'POST', headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error('Failed to restore conversation')
  })
}

export function listArchivedAIConversations(): Promise<AIConversation[]> {
  return fetch(`${API_BASE}/api/ai/conversations/archived`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error('Failed to load archived conversations')
    return res.json() as Promise<AIConversation[]>
  })
}
