export interface PromptFolder {
  id: number
  user_id: number
  name: string
  created_at: string
  updated_at: string
}

export interface Prompt {
  id: number
  user_id: number
  folder_id: number | null
  title: string
  content: string
  description: string
  tags: string[]
  folder_name: string | null
  created_at: string
  updated_at: string
}

export interface CreatePromptFolderInput {
  user_id: number
  name: string
}

export interface UpdatePromptFolderInput {
  name: string
}

export interface CreatePromptInput {
  user_id: number
  folder_id?: number | null
  title: string
  content: string
  description?: string
  tags?: string[]
}

export interface UpdatePromptInput {
  title?: string
  content?: string
  description?: string
  folder_id?: number | null
  tags?: string[]
}
