export interface HealthResponse {
  status: string
  uptime_seconds: number
}

export interface Todo {
  id: number
  user_id: number
  title: string
  description: string
  domain: string
  screenshot_path: string
  completed: boolean
  created_at: string
  updated_at: string
}

export interface Screenshot {
  id: number
  todo_id: number
  filename: string
  created_at: string
}

export interface Bookmark {
  id: number
  user_id: number
  title: string
  url: string
  description: string
  tags: string[]
  folder_path: string
  created_at: string
  updated_at: string
}

export interface BookmarkResponse {
  id: number
  user_id: number
  title: string
  url: string
  description: string
  tags: string[]
  folder_path: string
  created_at: string
  updated_at: string
}

export interface ImportResult {
  imported: number
  skipped: number
  bookmarks: BookmarkResponse[]
}

export interface History {
  id: number
  user_id: number
  url: string
  title: string
  visited_at: string
  duration: number
}

export interface HistoryImportResult {
  imported: number
  skipped: number
  errors: string[]
}

export type HistorySearchColumn = 'all' | 'title' | 'url'

export interface GroupedHistoryEntry {
  url: string
  title: string
  count: number
  total_duration: number
  last_visited: string
}

export interface GroupedHistoryResponse {
  entries: GroupedHistoryEntry[]
  total: number
  limit: number
  offset: number
}

export interface GroupedHistoryParams {
  user_id?: number
  domain?: string
  q?: string
  column?: HistorySearchColumn
  limit?: number
  offset?: number
}

export interface HistoryDomainSummary {
  domain: string
  visit_count: number
  url_count: number
  total_duration: number
  last_visited: string
}

export interface OmniboxSearchResult {
  source: 'history' | 'bookmark'
  title: string
  url: string
  description?: string
  tags?: string[]
  folder_path?: string
  visit_count?: number
  last_visited?: string
  updated_at?: string
}

export interface OmniboxSearchParams {
  user_id?: number
  q: string
  limit?: number
}

export interface WalletEntry {
  id: number
  user_id: number
  username: string
  password: string
  website: string
  login_provider: string
  description: string
  created_at: string
  updated_at: string
}

export interface WalletImportResult {
  imported: number
  skipped: number
  errors: string[]
}

export interface CreateWalletInput {
  user_id: number
  website: string
  login_provider?: string
  username: string
  password: string
  description?: string
}

export interface User {
  id: number
  username: string
  email: string
}

export interface Route {
  method: string
  path: string
  description: string
}

export interface CreateTodoInput {
  user_id: number
  title: string
  description?: string
  domain?: string
  capture_id?: string
}

export interface UpdateTodoInput {
  user_id?: number
  title?: string
  description?: string
  domain?: string
  completed?: boolean
  screenshot_path?: string
}

export interface CreateHistoryInput {
  user_id: number
  url: string
  title: string
  duration?: number
}

export interface CreateBookmarkInput {
  user_id: number
  title: string
  url: string
  description?: string
  tags?: string[]
  folder_path?: string
  capture_id?: string
}

export interface UpdateBookmarkInput {
  user_id: number
  title: string
  url: string
  description?: string
  tags?: string[]
  folder_path?: string
}

export interface UpdateWalletInput {
  username?: string
  password?: string
  website?: string
  login_provider?: string
  description?: string
}

export interface UsageEntry {
  domain: string
  date: string
  seconds: number
}

export interface UsageBatchRequest {
  user_id: number
  entries: UsageEntry[]
}

export interface UsageBatchResponse {
  upserted: number
}

export interface DomainUsage {
  domain: string
  total_seconds: number
  percentage: number
}

export interface TimelinePoint {
  period: string
  total_seconds: number
}

export interface AnalyticsSummary {
  total_seconds: number
  domains: DomainUsage[]
  timeline: TimelinePoint[]
}

export interface AnalyticsSummaryParams {
  user_id: number
  start_date: string
  end_date: string
  group_by?: 'day' | 'week' | 'month'
  limit?: number
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
}

export interface AIStreamToolResultEvent {
  type: 'tool_result'
  message_id: string
  content: string
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
