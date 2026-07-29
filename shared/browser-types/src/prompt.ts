export interface Prompt {
  id: number
  user_id: number
  title: string
  content: string
  description: string
  tags: string[]
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
}

export interface UpdatePromptInput {
  title?: string
  content?: string
  description?: string
  tags?: string[]
}
