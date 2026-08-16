import type { VideoStatus } from '@browser-server/shared-types';

export function formatBytes(bytes: number): string {
  if (!bytes) return '—';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function formatVideoDate(value: string): string {
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '—';
  const thisYear = new Date().getFullYear() === d.getFullYear();
  return d.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    // Old entries (e.g., last December) look identical to recent ones without
    // a year, so include it whenever the year differs.
    ...(thisYear ? {} : { year: 'numeric' }),
  });
}

/** Seconds (Agnes reports "seconds" as a number) → "0:05" / "1:02" style. */
export function formatDuration(seconds?: number): string {
  if (!seconds || seconds <= 0) return '—';
  const total = Math.round(seconds);
  const m = Math.floor(total / 60);
  const s = total % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}

export const videoStatusLabels: Record<VideoStatus, string> = {
  queued: 'Queued',
  in_progress: 'Generating',
  completed: 'Completed',
  failed: 'Failed',
};

/** Tailwind class fragments for each status badge. */
export function statusBadgeClass(status: VideoStatus): string {
  switch (status) {
    case 'completed':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300';
    case 'in_progress':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300';
    case 'failed':
      return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300';
    default:
      return 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300';
  }
}
