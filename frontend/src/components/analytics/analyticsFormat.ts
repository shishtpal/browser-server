/** Usage-page presentation constants (presets, group-by options, bar colors). */

export type DatePreset = 'today' | '7days' | '30days' | 'custom';
export type GroupBy = 'day' | 'week' | 'month';

export const DATE_PRESETS: { value: DatePreset; label: string }[] = [
  { value: 'today', label: 'Today' },
  { value: '7days', label: '7 Days' },
  { value: '30days', label: '30 Days' },
  { value: 'custom', label: 'Custom' },
];

export const GROUP_OPTIONS: { value: GroupBy; label: string }[] = [
  { value: 'day', label: 'Day' },
  { value: 'week', label: 'Week' },
  { value: 'month', label: 'Month' },
];

/** yyyy-MM-dd range for a preset (custom needs explicit start/end). */
export function presetRange(
  preset: DatePreset,
  customStart: string,
  customEnd: string,
): { start: string; end: string } {
  const today = new Date().toISOString().slice(0, 10);
  switch (preset) {
    case 'today':
      return { start: today, end: today };
    case '7days': {
      const d = new Date();
      d.setDate(d.getDate() - 6);
      return { start: d.toISOString().slice(0, 10), end: today };
    }
    case '30days': {
      const d = new Date();
      d.setDate(d.getDate() - 29);
      return { start: d.toISOString().slice(0, 10), end: today };
    }
    case 'custom':
      return { start: customStart || today, end: customEnd || today };
  }
}

/** "W12" for "2024-W12", "Mar 4" for a day period, raw period for months. */
export function periodLabel(period: string, groupBy: GroupBy): string {
  if (groupBy === 'month') return period;
  if (groupBy === 'week') {
    const [, week] = period.split('-');
    return week ? `W${week}` : period;
  }
  const d = new Date(period);
  if (Number.isNaN(d.getTime())) return period;
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

/** Ranked bar palette: hot rose for the top domains, cooling to slate. */
export const DOMAIN_BAR_COLORS = [
  'bg-rose-500',
  'bg-rose-400',
  'bg-rose-300',
  'bg-rose-200',
  'bg-slate-400',
  'bg-slate-500',
  'bg-slate-600',
  'bg-slate-700',
  'bg-slate-800',
  'bg-slate-900',
] as const;

export const domainBarColor = (index: number): string =>
  DOMAIN_BAR_COLORS[index % DOMAIN_BAR_COLORS.length];

/** Google favicon service URL (external thumbnail for a domain). */
export const domainFaviconUrl = (domain: string): string =>
  `https://www.google.com/s2/favicons?domain=${domain}&sz=16`;
