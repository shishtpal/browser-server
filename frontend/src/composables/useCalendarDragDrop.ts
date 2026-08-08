import { ref } from 'vue'
import type { Todo } from '../types'

export const DRAG_MIME_TYPE = 'application/x-browser-server-calendar-todo'

export interface CalendarDragPayload {
  id: number
  startDate: string
}

export function useCalendarDragDrop() {
  const dragOverDate = ref<string | null>(null)

  function getDragPayload(dataTransfer: DataTransfer | null): CalendarDragPayload | null {
    const raw = dataTransfer?.getData(DRAG_MIME_TYPE)
    if (!raw) return null
    try {
      const parsed = JSON.parse(raw) as unknown
      if (
        parsed &&
        typeof parsed === 'object' &&
        'id' in parsed &&
        'startDate' in parsed &&
        typeof parsed.id === 'number' &&
        typeof parsed.startDate === 'string'
      ) {
        return parsed as CalendarDragPayload
      }
    } catch {
      // ignore malformed payload
    }
    return null
  }

  function hasCalendarPayload(dataTransfer: DataTransfer | null): boolean {
    return dataTransfer?.types.includes(DRAG_MIME_TYPE) ?? false
  }

  function isDropAllowed(
    payload: CalendarDragPayload | null,
    date: string,
  ): payload is CalendarDragPayload {
    return payload !== null && payload.startDate !== date
  }

  return {
    dragOverDate,
    getDragPayload,
    hasCalendarPayload,
    isDropAllowed,
  }
}

export function todoFromPayload(payload: { id: number }, todos: Todo[]): Todo | undefined {
  return todos.find((t) => t.id === payload.id)
}
