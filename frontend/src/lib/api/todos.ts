import type { CreateTodoInput, GetTodosOptions, ReorderItem, Screenshot, Todo } from '@browser-server/shared-types'
import { API_BASE, authHeaders, client } from './client'

export function getTodos(userId?: number, domain?: string, options?: GetTodosOptions): Promise<Todo[]> {
  return client.getTodos(userId, domain, options)
}

export function getTodo(id: number): Promise<Todo> {
  return fetch(`${API_BASE}/api/todos/${id}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<Todo>
  })
}

export function createTodo(data: CreateTodoInput): Promise<Todo> {
  return client.createTodo(data)
}

export function updateTodo(id: number, data: Partial<Todo>): Promise<Todo> {
  return client.updateTodo(id, data as Parameters<typeof client.updateTodo>[1])
}

export function deleteTodo(id: number): Promise<void> {
  return client.deleteTodo(id)
}

export function reorderTodos(items: ReorderItem[]): Promise<void> {
  return client.reorderTodos(items)
}

export function getSubtasks(todoId: number): Promise<Todo[]> {
  return client.getSubtasks(todoId)
}

export function createSubtask(todoId: number, data: CreateTodoInput): Promise<Todo> {
  return client.createSubtask(todoId, data)
}

export function uploadScreenshot(todoId: number, file: Blob): Promise<Screenshot> {
  return client.uploadScreenshot(todoId, file)
}

export function getScreenshotUrl(todoId: number): string {
  return client.getScreenshotUrl(todoId)
}
