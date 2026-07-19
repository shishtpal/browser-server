import type { CreateTodoInput, Todo, UpdateTodoInput } from '@browser-server/shared-types'
import { type TokenProvider, apiFetch, buildQuery } from '../internals'

export function createTodoMethods(baseUrl: string, getToken?: TokenProvider) {
  return {
    getTodos(userId?: number, domain?: string): Promise<Todo[]> {
      return apiFetch<Todo[]>(baseUrl, 'GET', `/api/todos${buildQuery({ user_id: userId, domain })}`, undefined, getToken)
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
  }
}
