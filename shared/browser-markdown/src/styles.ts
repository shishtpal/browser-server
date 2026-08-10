import type { StyleKey } from './types';

/* ---------------------------------------------------------------- styling */

export const DEFAULT_CLASSES: Record<StyleKey, string> = {
  h1: 'mt-5 mb-2 text-lg font-black tracking-tight',
  h2: 'mt-4 mb-2 text-base font-bold',
  h3: 'mt-4 mb-1 font-bold',
  h4: 'mt-3 mb-1 font-bold',
  h5: 'mt-3 mb-1 text-[0.95em] font-semibold',
  h6: 'mt-3 mb-1 text-[0.9em] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400',
  p: 'my-2 leading-relaxed',
  a: 'font-medium text-sky-600 underline underline-offset-2 hover:text-sky-500 dark:text-sky-400',
  strong: 'font-semibold',
  em: 'italic',
  del: 'line-through opacity-70',
  mark: 'rounded bg-yellow-200 px-1 dark:bg-yellow-500/30',
  ul: 'my-2 ml-5 list-disc space-y-1',
  ol: 'my-2 ml-5 list-decimal space-y-1',
  li: 'pl-1 marker:text-slate-400',
  taskList: 'my-2 ml-1 list-none space-y-1',
  taskItem: 'flex items-start gap-2',
  checkbox: 'mt-1 h-3.5 w-3.5 shrink-0 accent-sky-600',
  blockquote:
    'my-3 border-l-4 border-slate-300 pl-4 text-slate-600 italic dark:border-slate-600 dark:text-slate-300',
  alert: 'my-3 rounded-lg border-l-4 px-4 py-2 text-[0.95em] not-italic',
  alertTitle: 'mb-1 flex items-center gap-1.5 font-semibold',
  hr: 'my-4 border-t border-slate-200 dark:border-slate-700',
  img: 'my-3 max-w-full rounded-lg',
  tableWrap:
    'not-prose my-3 overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-700',
  table: 'w-full border-collapse text-left text-[0.9em]',
  thead: 'bg-slate-50 dark:bg-slate-800',
  tbody: 'divide-y divide-slate-200 dark:divide-slate-700',
  tr: '',
  th: 'px-3 py-2 font-semibold whitespace-nowrap',
  td: 'px-3 py-2 align-top',
  code: 'rounded bg-slate-100 px-1.5 py-0.5 font-mono text-[0.85em] dark:bg-slate-800',
  pre: 'overflow-x-auto rounded-lg bg-slate-800 p-3 pr-14 text-[0.85em] text-slate-50 dark:bg-slate-900 dark:text-slate-100',
  codeWrap: 'not-prose group/code relative my-3',
  codeLang:
    'absolute left-3 top-2 select-none text-[0.65em] font-semibold uppercase tracking-wider text-slate-400',
  copyButton:
    'absolute right-2 top-2 rounded-md bg-slate-200/90 px-2 py-1 text-[0.7em] font-semibold text-slate-600 transition hover:bg-slate-300 sm:opacity-0 sm:group-hover/code:opacity-100 sm:focus:opacity-100 dark:bg-slate-700/90 dark:text-slate-200 dark:hover:bg-slate-600',
  footnotes: 'mt-6 text-[0.85em] text-slate-600 dark:text-slate-400',
  footnoteRef: 'text-sky-600 no-underline dark:text-sky-400',
  footnoteBackref: 'ml-1 text-sky-600 no-underline dark:text-sky-400',
};

export const ALERTS: Record<string, { label: string; icon: string; cls: string }> = {
  note: {
    label: 'Note',
    icon: 'ℹ️',
    cls: 'border-sky-500 bg-sky-50 text-sky-900 dark:bg-sky-500/10 dark:text-sky-200',
  },
  tip: {
    label: 'Tip',
    icon: '💡',
    cls: 'border-emerald-500 bg-emerald-50 text-emerald-900 dark:bg-emerald-500/10 dark:text-emerald-200',
  },
  important: {
    label: 'Important',
    icon: '❗',
    cls: 'border-violet-500 bg-violet-50 text-violet-900 dark:bg-violet-500/10 dark:text-violet-200',
  },
  warning: {
    label: 'Warning',
    icon: '⚠️',
    cls: 'border-amber-500 bg-amber-50 text-amber-900 dark:bg-amber-500/10 dark:text-amber-200',
  },
  caution: {
    label: 'Caution',
    icon: '🛑',
    cls: 'border-rose-500 bg-rose-50 text-rose-900 dark:bg-rose-500/10 dark:text-rose-200',
  },
};
