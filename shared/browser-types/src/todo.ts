export type TodoPriority = 'low' | 'medium' | 'high' | 'urgent'
export type TodoStatus = 'pending' | 'completed' | 'archived'

export interface Todo {
  id: number
  user_id: number
  title: string
  description: string
  domain: string
  screenshot_path: string
  pinned: boolean
  status: TodoStatus
  priority: TodoPriority
  color: string
  start_date: string | null
  end_date: string | null
  rrule: string
  tags: string[]
  parent_id: number | null
  position: number
  subtasks?: Todo[]
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
  priority?: TodoPriority
  status?: TodoStatus
  color?: string
  start_date?: string | null
  end_date?: string | null
  rrule?: string | null
  tags?: string[]
  parent_id?: number | null
}

export interface UpdateTodoInput {
  user_id?: number
  title?: string
  description?: string
  domain?: string
  screenshot_path?: string
  pinned?: boolean
  status?: TodoStatus
  priority?: TodoPriority
  color?: string
  start_date?: string | null
  end_date?: string | null
  rrule?: string | null
  tags?: string[]
  position?: number
}

export interface ReorderItem {
  id: number
  position: number
}

export interface ReorderTodosInput {
  items: ReorderItem[]
}
