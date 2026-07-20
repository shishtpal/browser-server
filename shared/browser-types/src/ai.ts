export interface AIProfile {
  name: string
  label: string
}

export interface AIConfig {
  enabled: boolean
  default_provider?: string
  providers: Record<string, AIProviderConfig>
  tools: {
    enabled: boolean
    allowed: string[]
    max_iterations: number
  }
  chat: {
    max_history_messages: number
    stream: boolean
    temperature: number
  }
  profiles: AIProfile[]
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
  default: boolean
  max_output_tokens: number
}

export interface AIConversation {
  id: string
  title: string
  provider: string
  model: string
  profile: string
  preview?: string
  created_at: string
  updated_at: string
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

export interface SendAIMessageInput {
  content: string
  provider?: string
  model?: string
  stream?: boolean
  tools_enabled?: boolean
  yolo_mode?: boolean
  active_tools?: string[]
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
export type AIStreamEventType = 'delta' | 'tool_call' | 'tool_result' | 'done' | 'error'

export interface AIStreamDeltaEvent {
  type: 'delta'
  message_id: string
  content: string
}

export interface AIStreamToolCallEvent {
  type: 'tool_call'
  message_id: string
  tool_call: { id: string; name: string; arguments: string }
  status: 'pending' | 'approved'
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
  | AIStreamDoneEvent
  | AIStreamErrorEvent
