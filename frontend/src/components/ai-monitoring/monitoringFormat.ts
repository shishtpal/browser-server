/** AI-monitoring presentation helpers — single source for page + detail. */

/** Request-metrics window select options (hours). */
export const WINDOW_OPTIONS = [
  { value: 24, label: '24 hours' },
  { value: 168, label: '7 days' },
  { value: 720, label: '30 days' },
  { value: 2160, label: '90 days' },
] as const;

/** Milliseconds → human duration ("1.23s" / "480ms"). */
export function formatMs(ms: number | null | undefined): string {
  if (ms === undefined || ms === null) return '—';
  return ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${Math.round(ms)}ms`;
}

export const formatNumber = (value?: number): string => (value ?? 0).toLocaleString();

export const formatOptionalNumber = (value?: number | null): string =>
  value === undefined || value === null ? '—' : value.toLocaleString();

export const formatTimestamp = (value?: string | null): string =>
  value ? new Date(value).toLocaleString() : 'No activity';

/** Backend source names → display labels. */
export function sourceLabel(value?: string): string {
  if (value === 'task_agent') return 'Task agent';
  if (value === 'chat') return 'Chat';
  return value || '—';
}

/** Status pill classes — one shared scale for requests and tool calls. */
export function statusPillClass(status?: string): string {
  const value = (status ?? '').toLowerCase();
  if (['success', 'completed', 'complete', 'ok'].includes(value)) {
    return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-950/30 dark:text-emerald-300';
  }
  if (['failed', 'error', 'cancelled'].includes(value)) {
    return 'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-900/50 dark:bg-rose-950/30 dark:text-rose-300';
  }
  if (['pending', 'queued', 'running', 'processing'].includes(value)) {
    return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-300';
  }
  return 'border-slate-200 bg-slate-50 text-slate-600 dark:border-white/10 dark:bg-white/5 dark:text-slate-300';
}
