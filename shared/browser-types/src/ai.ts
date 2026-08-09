export interface AIProfile {
  name: string
  label: string
}

export interface AIMCPServerStatus {
  name: string
  status: 'connected' | 'unavailable' | 'disabled'
  tools?: string[]
  warnings?: string[]
  error?: string
}

export interface AIMCPConfig {
  configured: boolean
  servers: AIMCPServerStatus[]
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
    show_thinking: boolean
    temperature: number
    attachments?: AIChatAttachmentsConfig
  }
  profiles: AIProfile[]
  skills: AISkill[]
  mcp?: AIMCPConfig
  tasks?: AITasksConfig
}

/** Server-side limits for the durable background task runner. */
export interface AITasksConfig {
  enabled: boolean
  max_concurrent: number
  max_steps: number
  max_attempts: number
  tools_enabled: boolean
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
  /** Model thinking/chain-of-thought for assistant messages, when the
   * provider returned reasoning content. */
  reasoning?: string
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

/** ask_questions AI Tool */
export type ChatQuestionKind = 'text' | 'choice' | 'multi_choice' | 'multiple_choice' | 'confirm'

export interface ChatQuestion {
  id: string
  prompt: string
  kind?: ChatQuestionKind
  options?: string[]
  default?: string
  required?: boolean
}

/** SSE event types emitted during streaming AI message generation. */
export type AIStreamEventType = 'delta' | 'reasoning' | 'tool_call' | 'tool_result' | 'append_window' | 'done' | 'error'

export interface AIStreamDeltaEvent {
  type: 'delta'
  message_id: string
  content: string
}

export interface AIStreamReasoningEvent {
  type: 'reasoning'
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
  | AIStreamReasoningEvent
  | AIStreamToolCallEvent
  | AIStreamToolResultEvent
  | AIStreamAppendWindowEvent
  | AIStreamDoneEvent
  | AIStreamErrorEvent

/**
 * Lifecycle of a durable background task. There is deliberately no 'stale'
 * status: a task whose worker lease expired is still 'running' until the
 * watchdog requeues or fails it. Staleness is reported separately on
 * `AITask.stale` so the UI can flag it without it becoming a state that can be
 * observed but never acted upon.
 */
export type AITaskStatus = 'queued' | 'running' | 'completed' | 'failed'

export interface AITask {
  id: string
  conversation_id?: string
  prompt: string
  status: AITaskStatus
  /** Present on completed tasks: the agent's structured result. */
  result?: AITaskResult
  lease_owner?: string
  lease_until?: string
  last_heartbeat?: string
  /** Last time a checkpoint was written — the signal that the agent, not just
   * its worker, is alive. */
  last_progress?: string
  available_at: string
  attempts: number
  max_attempts: number
  last_error?: string
  created_at: string
  completed_at?: string
  /** True when the task is 'running' but its lease has already expired; the
   * watchdog has not swept it yet. */
  stale: boolean
  /** Whether resumable state exists, so a retry resumes rather than restarts. */
  has_checkpoint: boolean
}

export interface AITaskResult {
  status: 'completed'
  response: string
  steps: number
}

export interface CreateAITaskInput {
  prompt: string
  conversation_id?: string
  max_attempts?: number
}

export interface CreateAITaskResponse {
  task_id: string
}

export interface AITaskStatusResponse {
  enabled: boolean
  workers: number
  counts: Record<AITaskStatus, number>
}

export type AIAuditSource = 'chat' | 'task_agent'
export type AIAuditStatus = 'success' | 'error' | 'cancelled'

export interface AIToolCallLog {
  id: string
  request_id: string
  message_id?: string
  tool_name: string
  arguments?: string
  result?: string
  error_message?: string
  status: 'success' | 'error' | 'cancelled' | 'rejected'
  decision: 'approved' | 'rejected' | 'commented' | 'answered' | 'unauthorized' | 'replayed' | string
  duration_ms: number
  payload_truncated: boolean
  created_at: string
}

export interface AIRequestLog {
  id: string
  conversation_id?: string
  message_id?: string
  source: AIAuditSource
  task_id?: string
  iteration: number
  provider: string
  model: string
  endpoint: string
  request_payload?: string
  response_payload?: string
  payload_truncated: boolean
  http_status?: number
  prompt_tokens?: number
  completion_tokens?: number
  total_tokens?: number
  latency_ms: number
  status: AIAuditStatus
  error_code?: string
  error_message?: string
  created_at: string
  tool_calls?: AIToolCallLog[]
}

export interface AIRequestLogList {
  logs: AIRequestLog[]
  limit: number
  offset: number
}

export interface AIMonitoring {
  window_hours: number
  requests: number
  errors: number
  cancellations: number
  tool_successes: number
  tool_errors: number
  tool_rejections: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  average_latency_ms: number
  max_latency_ms: number
  latest_activity?: string
}

// ─── Memory graph (v2) ────────────────────────────────────────────────

export interface AIMemoryStats {
  enabled: boolean
  fragments: number
  root: string
  index_file: string
}

export interface AIMemoryGraphNode {
  id: string
  kind: string
  title: string
  summary: string
}

export interface AIMemoryGraphEdge {
  from: string
  rel: string
  to: string
}

export interface AIMemoryGraph {
  nodes: AIMemoryGraphNode[]
  edges: AIMemoryGraphEdge[]
}

export interface AIMemoryLink {
  rel: string
  to: string
  note?: string
}

export interface AIMemoryFragment {
  id: string
  kind: string
  title: string
  summary: string
  body: string
  tags: string[]
  status: string
  pinned: boolean
  parent: string
  links: AIMemoryLink[]
}

export interface AIMemoryWriteLink {
  rel: string
  to: string
  note?: string
}

export interface AIMemoryWriteOp {
  op: 'upsert' | 'append' | 'link' | 'unlink' | 'move' | 'archive' | 'delete'
  id?: string
  kind?: string
  title?: string
  summary?: string
  body?: string
  tags?: string[]
  parent?: string
  pinned?: boolean
  status?: string
  confidence?: number
  links?: AIMemoryWriteLink[]
  from?: string
  rel?: string
  to?: string
  on_conflict?: 'merge' | 'new' | 'error'
  superseded_by?: string
  cascade?: boolean
}

export interface AIMemoryWriteResult {
  applied: number
  results: {
    op: string
    id: string
    created?: boolean
    merged_fields?: string[]
    duplicate_of?: string
    warning?: string
  }[]
  warnings?: string[]
}
