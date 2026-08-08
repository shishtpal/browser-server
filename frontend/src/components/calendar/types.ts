import type { Todo } from '../../types';

export type CalendarView = 'month' | 'week' | 'day' | 'year';

export interface CalendarDay {
  date: string;
  isToday: boolean;
  isCurrentMonth: boolean;
  isWeekend: boolean;
  todos: Todo[];
}

export interface DateRange {
  start: Date;
  end: Date;
}

export interface CalendarStats {
  todayCount: number;
  overdueCount: number;
  completedCount: number;
}
