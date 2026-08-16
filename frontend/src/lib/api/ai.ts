import type {
  AIConfig,
  AIVoiceConfig,
  AIMonitoring,
  AIRequestLogList,
  AIConversation,
  AIConversationDetail,
  AIImageAttachment,
  AIAttachmentSummary,
  AIMessage,
  AIStreamEvent,
  AITask,
  AITaskStatus,
  AITaskStatusResponse,
  AIToolDecisionResponse,
  AppendAIMessageInput,
  CreateAIConversationInput,
  CreateAITaskInput,
  CreateAITaskResponse,
  SendAIMessageInput,
  SendAIMessageResponse,
  StopAIGenerationResponse,
  AIImageConfig,
  GeneratedImage,
  GenerateImageInput,
  GenerateImageResponse,
  AIVideoConfig,
  GeneratedVideo,
  GenerateVideoInput,
  GenerateVideoResponse,
  AITTSConfig,
  GeneratedSpeech,
  GenerateSpeechInput,
  GenerateSpeechResponse,
  UpdateAIConversationInput,
} from '@browser-server/shared-types';
import { API_BASE, authHeaders, client } from './client';

// ─── AI Chat ────────────────────────────────────────────

export function getAIConfig(): Promise<AIConfig> {
  return client.getAIConfig();
}
export function getAIImageConfig(): Promise<AIImageConfig> {
  return client.getAIImageConfig();
}
export function listGeneratedImages(limit?: number): Promise<GeneratedImage[]> {
  return client.listGeneratedImages(limit);
}
export function generateImage(data: GenerateImageInput): Promise<GenerateImageResponse> {
  return client.generateImage(data);
}
export function deleteGeneratedImage(id: string): Promise<void> {
  return client.deleteGeneratedImage(id);
}
export function getGeneratedImageUrl(id: string): string {
  return client.getGeneratedImageUrl(id);
}

export function getAITTSConfig(): Promise<AITTSConfig> {
  return client.getAITTSConfig();
}
export function listGeneratedSpeeches(limit?: number): Promise<GeneratedSpeech[]> {
  return client.listGeneratedSpeeches(limit);
}
export function generateSpeech(data: GenerateSpeechInput): Promise<GenerateSpeechResponse> {
  return client.generateSpeech(data);
}
export function deleteGeneratedSpeech(id: string): Promise<void> {
  return client.deleteGeneratedSpeech(id);
}
export function getGeneratedSpeechUrl(id: string, withToken = true): string {
  return client.getGeneratedSpeechUrl(id, withToken);
}

export function getAIVideoConfig(): Promise<AIVideoConfig> {
  return client.getAIVideoConfig();
}
export function listGeneratedVideos(limit?: number): Promise<GeneratedVideo[]> {
  return client.listGeneratedVideos(limit);
}
export function generateVideo(data: GenerateVideoInput): Promise<GenerateVideoResponse> {
  return client.generateVideo(data);
}
export function deleteGeneratedVideo(id: string): Promise<void> {
  return client.deleteGeneratedVideo(id);
}
export function getGeneratedVideoUrl(id: string, withToken = true): string {
  return client.getGeneratedVideoUrl(id, withToken);
}

export function getAIVoiceConfig(): Promise<AIVoiceConfig> {
  return client.getAIVoiceConfig();
}

export function getAIMonitoring(windowHours?: number): Promise<AIMonitoring> {
  return client.getAIMonitoring(windowHours);
}

export function getAIRequestLogs(
  filters: {
    source?: 'chat' | 'task_agent';
    status?: 'success' | 'error' | 'cancelled';
    conversationId?: string;
    taskId?: string;
    limit?: number;
    offset?: number;
  } = {},
): Promise<AIRequestLogList> {
  return client.getAIRequestLogs(filters);
}

export function listAIConversations(query?: string, limit?: number): Promise<AIConversation[]> {
  return client.listAIConversations(query, limit);
}

export function createAIConversation(
  data: CreateAIConversationInput = {},
): Promise<AIConversation> {
  return client.createAIConversation(data);
}

export function forkAIConversation(id: string, messageId: string): Promise<AIConversation> {
  return client.forkAIConversation(id, { message_id: messageId });
}

export function getAIConversation(id: string): Promise<AIConversationDetail> {
  return client.getAIConversation(id);
}

export function updateAIConversation(
  id: string,
  data: UpdateAIConversationInput,
): Promise<AIConversation> {
  return client.updateAIConversation(id, data);
}

export function deleteAIConversation(id: string): Promise<void> {
  return client.deleteAIConversation(id);
}

export function sendAIMessage(
  id: string,
  data: SendAIMessageInput,
): Promise<SendAIMessageResponse> {
  return client.sendAIMessage(id, data);
}

export function uploadAIImageAttachment(
  id: string,
  file: Blob,
  filename?: string,
): Promise<AIImageAttachment> {
  return client.uploadAIImageAttachment(id, file, filename);
}

export function deleteAIImageAttachment(id: string, attachmentId: string): Promise<void> {
  return client.deleteAIImageAttachment(id, attachmentId);
}

export function renameAIImageAttachment(
  id: string,
  attachmentId: string,
  filename: string,
): Promise<AIAttachmentSummary> {
  return client.renameAIImageAttachment(id, attachmentId, filename);
}

export function getAIImageAttachmentUrl(id: string, attachmentId: string): string {
  return client.getAIImageAttachmentUrl(id, attachmentId);
}

export function getAIImageAttachmentBlob(id: string, attachmentId: string): Promise<Blob> {
  return client.getAIImageAttachmentBlob(id, attachmentId);
}

export function listAIAttachments(limit?: number): Promise<AIAttachmentSummary[]> {
  return client.listAIAttachments(limit);
}

export function appendAIMessage(id: string, data: AppendAIMessageInput): Promise<AIMessage> {
  return client.appendAIMessage(id, data);
}

export function sendAIMessageStream(
  id: string,
  data: SendAIMessageInput,
  onEvent: (event: AIStreamEvent) => void,
  onError?: (err: Error) => void,
): AbortController {
  return client.sendAIMessageStream(id, data, onEvent, onError);
}

export function regenerateAIMessage(id: string): Promise<SendAIMessageResponse> {
  return client.regenerateAIMessage(id);
}

export function decideAIToolCall(
  id: string,
  callId: string,
  approved: boolean,
  comment?: string,
): Promise<AIToolDecisionResponse> {
  return client.decideAIToolCall(id, callId, approved, comment);
}

export function stopAIGeneration(id: string): Promise<StopAIGenerationResponse> {
  return client.stopAIGeneration(id);
}

export function updateAIMessage(
  conversationId: string,
  messageId: string,
  data: { content: string },
): Promise<AIMessage> {
  return client.updateAIMessage(conversationId, messageId, data);
}

export function deleteAIMessage(conversationId: string, messageId: string): Promise<void> {
  return client.deleteAIMessage(conversationId, messageId);
}

// ─── AI Chat Archiving ──────────────────────────────────────────────────────

export function archiveAIConversation(id: string): Promise<void> {
  return fetch(`${API_BASE}/api/ai/conversations/${id}/archive`, {
    method: 'POST',
    headers: authHeaders(),
  }).then((res) => {
    if (!res.ok) throw new Error('Failed to archive conversation');
  });
}

export function restoreAIConversation(id: string): Promise<void> {
  return fetch(`${API_BASE}/api/ai/conversations/${id}/restore`, {
    method: 'POST',
    headers: authHeaders(),
  }).then((res) => {
    if (!res.ok) throw new Error('Failed to restore conversation');
  });
}

export function listArchivedAIConversations(): Promise<AIConversation[]> {
  return fetch(`${API_BASE}/api/ai/conversations/archived`, { headers: authHeaders() }).then(
    (res) => {
      if (!res.ok) throw new Error('Failed to load archived conversations');
      return res.json() as Promise<AIConversation[]>;
    },
  );
}

// ─── AI Background Tasks ────────────────────────────────────────────────────

export function createAITask(data: CreateAITaskInput): Promise<CreateAITaskResponse> {
  return client.createAITask(data);
}

export function listAITasks(status?: AITaskStatus, limit?: number): Promise<AITask[]> {
  return client.listAITasks(status, limit);
}

export function getAITask(id: string): Promise<AITask> {
  return client.getAITask(id);
}

export function cancelAITask(id: string): Promise<void> {
  return client.cancelAITask(id);
}

export function deleteAITask(id: string): Promise<void> {
  return client.deleteAITask(id);
}

export function getAITaskStatus(): Promise<AITaskStatusResponse> {
  return client.getAITaskStatus();
}
