import { ref, computed, watch } from 'vue';
import type { CalendarView, DateRange } from '../components/calendar/types';

export function useCalendar() {
  const currentDate = ref(new Date());
  const view = ref<CalendarView>('month');

  const dateRange = computed<DateRange>(() => {
    const d = currentDate.value;
    if (view.value === 'year') {
      const start = new Date(d.getFullYear(), 0, 1);
      start.setHours(0, 0, 0, 0);
      const end = new Date(d.getFullYear(), 11, 31);
      end.setHours(23, 59, 59, 999);
      return { start, end };
    }
    if (view.value === 'month') {
      // Include leading days from previous month and trailing days to fill the grid
      const firstDay = new Date(d.getFullYear(), d.getMonth(), 1);
      const lastDay = new Date(d.getFullYear(), d.getMonth() + 1, 0);
      const start = new Date(firstDay);
      start.setDate(start.getDate() - firstDay.getDay());
      start.setHours(0, 0, 0, 0);
      const end = new Date(lastDay);
      end.setDate(end.getDate() + (6 - lastDay.getDay()));
      end.setHours(23, 59, 59, 999);
      return { start, end };
    }
    if (view.value === 'week') {
      const day = d.getDay();
      const start = new Date(d);
      start.setDate(d.getDate() - day);
      start.setHours(0, 0, 0, 0);
      const end = new Date(start);
      end.setDate(start.getDate() + 6);
      end.setHours(23, 59, 59, 999);
      return { start, end };
    }
    // day view
    const start = new Date(d.getFullYear(), d.getMonth(), d.getDate());
    start.setHours(0, 0, 0, 0);
    const end = new Date(start);
    end.setHours(23, 59, 59, 999);
    return { start, end };
  });

  const periodLabel = computed(() => {
    const d = currentDate.value;
    if (view.value === 'year') {
      return String(d.getFullYear());
    }
    if (view.value === 'month') {
      return d.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
    }
    if (view.value === 'week') {
      const start = dateRange.value.start;
      const end = dateRange.value.end;
      const sameMonth = start.getMonth() === end.getMonth();
      if (sameMonth) {
        return `${start.toLocaleDateString('en-US', { month: 'long' })} ${start.getDate()} – ${end.getDate()}, ${start.getFullYear()}`;
      }
      return `${start.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} – ${end.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} ${end.getFullYear()}`;
    }
    return d.toLocaleDateString('en-US', {
      weekday: 'long',
      month: 'long',
      day: 'numeric',
      year: 'numeric',
    });
  });

  function navigate(dir: number) {
    const d = currentDate.value;
    if (view.value === 'year') {
      currentDate.value = new Date(d.getFullYear() + dir, d.getMonth(), 1);
    } else if (view.value === 'month') {
      currentDate.value = new Date(d.getFullYear(), d.getMonth() + dir, 1);
    } else if (view.value === 'week') {
      currentDate.value = new Date(d.getFullYear(), d.getMonth(), d.getDate() + dir * 7);
    } else {
      currentDate.value = new Date(d.getFullYear(), d.getMonth(), d.getDate() + dir);
    }
  }

  function goToToday() {
    currentDate.value = new Date();
  }

  watch(view, () => {
    // Keep current date when switching views
  });

  return {
    currentDate,
    view,
    dateRange,
    periodLabel,
    navigate,
    goToToday,
  };
}
