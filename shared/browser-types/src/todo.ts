export type TodoView = 'list' | 'kanban' | 'grid'

export type TodoPriority = 'low' | 'medium' | 'high' | 'urgent'

export type TodoStatus = 'pending' | 'in_progress' | 'completed' | 'archived'

export type TodoSortField = 'position' | 'priority' | 'start_date' | 'created_at' | 'title'

export type TodoFilter = 'all' | 'active' | 'in_progress' | 'completed' | 'archived'

export type DueDateFilter = 'overdue' | 'today' | 'this_week' | null

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
  subtasks: Todo[]
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

export interface GetTodosOptions {
  status?: TodoStatus
  priority?: TodoPriority
  tag?: string
  parent_id?: number
  archived?: boolean
  sort?: TodoSortField
  order?: 'asc' | 'desc'
}

export interface ReorderItem {
  id: number
  position: number
}

export interface ReorderTodosInput {
  items: ReorderItem[]
}
