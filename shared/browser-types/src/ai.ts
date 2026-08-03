export interface AIProfile {
  name: string
  label: string
}

export interface AISkill {
  name: string
  label: string
  description?: string
  category?: string
  tags?: string[]
  tools?: string[]
}

export interface AIConfig {
  enabled: boolean
  default_provider?: string
  providers: Record<string, AIProviderConfig>
  tools: {
    enabled: boolean
    allowed: string[]
    categories: Record<string, string>
    max_iterations: number
  }
  chat: {
    max_history_messages: number
    stream: boolean
    temperature: number
    attachments?: AIChatAttachmentsConfig
  }
  profiles: AIProfile[]
  skills: AISkill[]
}

export interface AIProviderConfig {
  type: string
  default_model: string
  models: AIModelConfig[]
}

export interface AIModelConfig {
  id: string
  label: string
  supports_tools: boolean
  supports_vision: boolean
  default: boolean
  max_output_tokens: number
}

/** Server-enforced image attachment limits exposed by the config endpoint. */
export interface AIChatAttachmentsConfig {
  enabled: boolean
  allowed_mime_types: string[]
  max_images: number
  max_image_bytes: number
  max_total_bytes: number
}

/**
 * A server-issued image attachment for an AI conversation. `status` is
 * 'staged' until the attachment is claimed by a submitted user message.
 */
export interface AIImageAttachment {
  id: string
  message_id?: string
  filename: string
  content_type: 'image/png' | 'image/jpeg' | 'image/webp' | 'image/gif'
  size_bytes: number
  width?: number
  height?: number
  status: 'staged' | 'attached'
  created_at: string
}

/**
 * A gallery entry for the cross-conversation attachment library. Identical to
 * AIImageAttachment but additionally carries the owning conversation_id so the
 * client can build the authenticated image URL.
 */
export interface AIAttachmentSummary extends AIImageAttachment {
  conversation_id: string
}

/** Secret-free voice configuration returned to browser clients. */
export interface AIVoiceConfig {
  enabled: boolean
  default_provider?: string
  languages: AIVoiceLanguage[]
  recording: AIVoiceRecordingConfig
  providers: Record<string, AIVoiceProviderConfig>
}

export interface AIVoiceLanguage {
  code: string
  label: string
}

export interface AIVoiceRecordingConfig {
  silence_duration_ms: number
  speech_threshold: number
  max_duration_seconds: number
  max_frame_bytes: number
  max_audio_bytes: number
}

export interface AIVoiceProviderConfig {
  type: string
  enabled: boolean
  models: AIVoiceModelConfig[]
}

export interface AIVoiceModelConfig {
  id: string
  label: string
  sample_rate: 8000 | 16000
  mode?: string
  input_audio_codec?: string
  default: boolean
}

export interface AIConversation {
  id: string
  title: string
  provider: string
  model: string
  profile: string
  skills?: string[]
  preview?: string
  created_at: string
  updated_at: string
  archived?: boolean
}

export type AIMessageRole = 'system' | 'user' | 'assistant' | 'tool'
export type AIMessageStatus = 'pending' | 'completed' | 'error' | 'cancelled' | 'superseded'

export interface AIMessage {
  id: string
  conversation_id: string
  role: AIMessageRole
  content: string
  tool_call_id?: string
  status: AIMessageStatus
  created_at: string
  attachments?: AIImageAttachment[]
}

export interface AIConversationDetail {
  conversation: AIConversation
  messages: AIMessage[]
}

export interface CreateAIConversationInput {
  title?: string
  provider?: string
  model?: string
  profile?: string
}

export interface UpdateAIConversationInput {
  title?: string
  provider?: string
  model?: string
}

export interface ForkAIConversationInput {
  /** The message to branch from; the new conversation copies all messages up to and including this one. */
  message_id: string
}

export interface UpdateAIMessageInput {
  content: string
}

export interface SendAIMessageInput {
  content: string
  provider?: string
  model?: string
  stream?: boolean
  tools_enabled?: boolean
  yolo_mode?: boolean
  include_all_tool_definitions?: boolean
  active_tools?: string[]
  skills?: string[]
  /** true = force raw tool output, false = force JSON, omitted = follow server config allowlist */
  raw_tool_output?: boolean
  /** Server-issued staged attachment IDs to attach to this message. */
  attachment_ids?: string[]
}

export interface AppendAIMessageInput {
  content: string
}

export interface AIUsage {
  prompt_tokens?: number
  completion_tokens?: number
  total_tokens?: number
}

export interface SendAIMessageResponse {
  conversation_id: string
  user_message: AIMessage
  assistant_message: AIMessage
  tool_messages?: AIMessage[]
  usage: AIUsage
}

export interface StopAIGenerationResponse {
  stopped: boolean
}

export interface AIToolDecisionResponse {
  accepted: boolean
}

/** SSE event types emitted during streaming AI message generation. */
export type AIStreamEventType = 'delta' | 'tool_call' | 'tool_result' | 'append_window' | 'done' | 'error'

export interface AIStreamDeltaEvent {
  type: 'delta'
  message_id: string
  content: string
}

export interface AIStreamToolCallEvent {
  type: 'tool_call'
  message_id: string
  tool_call: { id: string; name: string; arguments: string }
  status: 'pending' | 'approved' | 'error'
}

export interface AIStreamAppendWindowEvent {
  type: 'append_window'
  status: 'open' | 'closed'
}

export interface AIStreamToolResultEvent {
  type: 'tool_result'
  message_id: string
  tool_call: { id: string; name: string; arguments: string }
  content: string
  status: 'completed' | 'error'
}

export interface AIStreamDoneEvent {
  type: 'done'
  conversation_id: string
  message_id: string
  status: string
  usage: AIUsage
}

export interface AIStreamErrorEvent {
  type: 'error'
  code: string
  message: string
}

export type AIStreamEvent =
  | AIStreamDeltaEvent
  | AIStreamToolCallEvent
  | AIStreamToolResultEvent
  | AIStreamAppendWindowEvent
  | AIStreamDoneEvent
  | AIStreamErrorEvent
