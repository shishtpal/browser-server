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
