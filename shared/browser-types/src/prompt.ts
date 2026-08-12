export interface Prompt {
  id: number
  user_id: number
  title: string
  content: string
  description: string
  tags: string[]
  pinned: boolean
  created_at: string
  updated_at: string
}

export type PromptResponse = Prompt

export interface CreatePromptInput {
  user_id: number
  title: string
  content: string
  description?: string
  tags?: string[]
  pinned?: boolean
}

export interface UpdatePromptInput {
  title?: string
  content?: string
  description?: string
  tags?: string[]
  pinned?: boolean
}
