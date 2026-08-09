import type { AITaskStatus } from '@browser-server/shared-types';
import type { LucideIcon } from '@lucide/vue';
import { CircleCheck, CircleX, CircleDot, Play } from '@lucide/vue';

export interface TaskStatusMeta {
  label: string;
  icon: LucideIcon;
  /** Badge classes (cards + filter chips). */
  badgeClass: string;
}

export const TASK_STATUS_META: Record<AITaskStatus, TaskStatusMeta> = {
  queued: {
    label: 'Queued',
    icon: CircleDot,
    badgeClass: 'bg-gray-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300',
  },
  running: {
    label: 'Running',
    icon: Play,
    badgeClass: 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400',
  },
  completed: {
    label: 'Completed',
    icon: CircleCheck,
    badgeClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
  },
  failed: {
    label: 'Failed',
    icon: CircleX,
    badgeClass: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  },
};

export const TASK_STATUS_ORDER: AITaskStatus[] = ['queued', 'running', 'completed', 'failed'];

/** "5" / "26" proof-readable worker subtitle for the submit form. */
export function workersLabel(workers: number): string {
  return `${workers} worker${workers === 1 ? '' : 's'}`;
}
