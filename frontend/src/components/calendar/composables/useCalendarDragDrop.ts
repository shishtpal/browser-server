import { ref, type ComputedRef, type Ref } from 'vue';
import type { Todo } from '../../../types';
import type { CalendarDay } from '../types';

export const DRAG_MIME_TYPE = 'application/x-browser-server-calendar-todo';

export interface CalendarDragPayload {
  id: number;
  startDate: string;
}

export function getDragPayload(dataTransfer: DataTransfer | null): CalendarDragPayload | null {
  const raw = dataTransfer?.getData(DRAG_MIME_TYPE);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (
      parsed &&
      typeof parsed === 'object' &&
      'id' in parsed &&
      'startDate' in parsed &&
      typeof parsed.id === 'number' &&
      typeof parsed.startDate === 'string'
    ) {
      return parsed as CalendarDragPayload;
    }
  } catch {
    // ignore malformed payload
  }
  return null;
}

export function hasCalendarPayload(dataTransfer: DataTransfer | null): boolean {
  return dataTransfer?.types.includes(DRAG_MIME_TYPE) ?? false;
}

export function isDropAllowed(
  payload: CalendarDragPayload | null,
  date: string,
): payload is CalendarDragPayload {
  return payload !== null && payload.startDate !== date;
}

export function todoFromPayload(payload: { id: number }, todos: Todo[]): Todo | undefined {
  return todos.find((t) => t.id === payload.id);
}

/**
 * Ready-made drop handlers for day cells (month + week views share one set).
 *
 * Usage:
 *   const drop = useCalendarDayDrop(days, (payload) => moveTodo(payload));
 *   <div @dragover.prevent="drop.onDragOver(day, $event)" ... />
 */
export function useCalendarDayDrop(
  days: Ref<CalendarDay[]> | ComputedRef<CalendarDay[]>,
  onMove: (payload: { todo: Todo; date: string }) => void,
) {
  const dragOverDate = ref<string | null>(null);

  function onDragOver(day: CalendarDay, event: DragEvent) {
    if (!hasCalendarPayload(event.dataTransfer)) return;
    const payload = getDragPayload(event.dataTransfer);
    if (!isDropAllowed(payload, day.date)) return;
    event.dataTransfer!.dropEffect = 'move';
    dragOverDate.value = day.date;
  }

  function onDragLeave(day: CalendarDay, event: DragEvent) {
    if (dragOverDate.value !== day.date) return;
    const target = event.currentTarget as HTMLElement | null;
    const related = event.relatedTarget as Node | null;
    if (target && related && target.contains(related)) return;
    dragOverDate.value = null;
  }

  function onDrop(day: CalendarDay, event: DragEvent) {
    const payload = getDragPayload(event.dataTransfer);
    dragOverDate.value = null;
    if (!isDropAllowed(payload, day.date)) return;
    const todo = todoFromPayload(
      payload,
      days.value.flatMap((d) => d.todos),
    );
    if (!todo) return;
    onMove({ todo, date: day.date });
  }

  function clearDrag() {
    dragOverDate.value = null;
  }

  /** True while this day is the current drop target (used for highlight classes). */
  function isDragTarget(day: CalendarDay) {
    return dragOverDate.value === day.date;
  }

  return { dragOverDate, onDragOver, onDragLeave, onDrop, clearDrag, isDragTarget };
}
