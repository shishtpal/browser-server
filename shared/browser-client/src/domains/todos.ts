import type { CreateTodoInput, ReorderItem, ReorderTodosInput, Todo, UpdateTodoInput } from '@browser-server/shared-types'
import { type TokenProvider, apiFetch, buildQuery } from '../internals'

export function createTodoMethods(baseUrl: string, getToken?: TokenProvider) {
  const queryBuilder = (params: Record<string, string | number | undefined>) => {
    const qs = buildQuery(params)
    return typeof qs === 'string' ? qs : ''
  }

  return {
    getTodos(userId?: number, domain?: string, options?: { priority?: string; tag?: string; parent_id?: number; sort?: string; order?: string }): Promise<Todo[]> {
      const qs = queryBuilder({ user_id: userId, domain, priority: options?.priority, tag: options?.tag, parent_id: options?.parent_id, sort: options?.sort, order: options?.order })
      return apiFetch<Todo[]>(baseUrl, 'GET', `/api/todos${qs}`, undefined, getToken)
    },

    createTodo(data: CreateTodoInput): Promise<Todo> {
      return apiFetch<Todo>(baseUrl, 'POST', '/api/todos', { ...data, completed: false }, getToken)
    },

    updateTodo(id: number, data: UpdateTodoInput): Promise<Todo> {
      return apiFetch<Todo>(baseUrl, 'PUT', `/api/todos/${id}`, data, getToken)
    },

    deleteTodo(id: number): Promise<void> {
      return apiFetch<void>(baseUrl, 'DELETE', `/api/todos/${id}`, undefined, getToken)
    },

    reorderTodos(items: ReorderItem[]): Promise<void> {
      const body: ReorderTodosInput = { items }
      return apiFetch<void>(baseUrl, 'POST', '/api/todos/reorder', body, getToken)
    },

    getSubtasks(todoId: number): Promise<Todo[]> {
      return apiFetch<Todo[]>(baseUrl, 'GET', `/api/todos/${todoId}/subtasks`, undefined, getToken)
    },

    createSubtask(todoId: number, data: CreateTodoInput): Promise<Todo> {
      return apiFetch<Todo>(baseUrl, 'POST', `/api/todos/${todoId}/subtasks`, data, getToken)
    },
  }
}
