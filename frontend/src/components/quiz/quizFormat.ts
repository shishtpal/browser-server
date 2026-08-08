import { ArrowDownUp, CircleDot, ListChecks, PenLine, type LucideIcon } from '@lucide/vue';
import type {
  ChronologyItem,
  QuestionDifficulty,
  QuestionResponse,
  QuestionType,
} from '../../types';
import type { UserAnswer } from './composables/attempts';

/* ------------------------------------------------------------------ */
/* Question type / difficulty metadata (single source of truth)        */
/* ------------------------------------------------------------------ */

export const QUESTION_TYPES: QuestionType[] = [
  'single_choice',
  'multiple_choice',
  'input',
  'chronology',
];

export interface QuestionTypeMeta {
  label: string;
  /** Tailwind classes for a pill/badge */
  badgeClass: string;
  icon: LucideIcon;
}

export const QUESTION_TYPE_META: Record<QuestionType, QuestionTypeMeta> = {
  single_choice: {
    label: 'Single choice',
    badgeClass: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300',
    icon: CircleDot,
  },
  multiple_choice: {
    label: 'Multiple choice',
    badgeClass: 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300',
    icon: ListChecks,
  },
  input: {
    label: 'Free text',
    badgeClass: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300',
    icon: PenLine,
  },
  chronology: {
    label: 'Chronology',
    badgeClass: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
    icon: ArrowDownUp,
  },
};

export const QUESTION_DIFFICULTIES: QuestionDifficulty[] = ['easy', 'medium', 'hard'];

export const DIFFICULTY_META: Record<
  QuestionDifficulty,
  { label: string; badgeClass: string; dotClass: string }
> = {
  easy: {
    label: 'Easy',
    badgeClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300',
    dotClass: 'bg-emerald-500',
  },
  medium: {
    label: 'Medium',
    badgeClass: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
    dotClass: 'bg-amber-500',
  },
  hard: {
    label: 'Hard',
    badgeClass: 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300',
    dotClass: 'bg-rose-500',
  },
};

export const formatQuestionType = (type: string): string =>
  QUESTION_TYPE_META[type as QuestionType]?.label ?? type.replaceAll('_', ' ');

/* ------------------------------------------------------------------ */
/* Date / time                                                         */
/* ------------------------------------------------------------------ */

/** mm:ss formatting for the exam timer. */
export const formatTime = (secs: number): string => {
  const m = Math.floor(secs / 60);
  const s = secs % 60;
  return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
};

export const formatShortDate = (iso: string): string =>
  new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });

export const formatDateTime = (iso: string): string => new Date(iso).toLocaleString();

/* ------------------------------------------------------------------ */
/* Question helpers                                                    */
/* ------------------------------------------------------------------ */

export const questionImageSrc = (
  url: string | undefined | null,
  apiBase: string,
): string | undefined => {
  if (!url) return undefined;
  return url.startsWith('http') ? url : `${apiBase}${url}`;
};

/** Option label: 0 -> "A", 1 -> "B"… */
export const optionLetter = (index: number): string => String.fromCharCode(65 + index);

export const chronologyItemText = (q: QuestionResponse, itemIndex: number): string =>
  q.chronology_items?.find((i) => i.index === itemIndex)?.text ?? '';

export const orderedChronology = (items: ChronologyItem[] | undefined | null): ChronologyItem[] =>
  [...(items ?? [])].sort((a, b) => a.correct_order - b.correct_order);

/**
 * Human readable rendering of a stored exam answer, used on the
 * post-submission review list.
 */
export const formatUserAnswerText = (q: QuestionResponse, ans: UserAnswer | undefined): string => {
  if (!ans) return 'No answer given';

  if (q.type === 'single_choice') {
    if (ans.singleChoice === undefined) return 'No selection';
    const opt = q.options?.find((o) => o.index === ans.singleChoice);
    return opt ? `${optionLetter(opt.index)}. ${opt.text}` : 'N/A';
  }
  if (q.type === 'multiple_choice') {
    if (!ans.multipleChoice?.length) return 'No selection';
    const opts = q.options?.filter((o) => ans.multipleChoice?.includes(o.index)) || [];
    return opts.map((o) => `${optionLetter(o.index)}. ${o.text}`).join(' | ');
  }
  if (q.type === 'input') {
    return ans.inputText?.trim() || 'No response typed';
  }
  if (q.type === 'chronology') {
    const order = ans.chronologyOrder || [];
    if (!order.length) return 'No ordering set';
    return order.map((idx, seq) => `${seq + 1}. ${chronologyItemText(q, idx)}`).join(' → ');
  }
  return 'N/A';
};
