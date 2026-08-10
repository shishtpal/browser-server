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

/**
 * Resolve a question image URL. The image endpoint is token-protected and
 * the <img> tag cannot send an Authorization header, so the token (when
 * available) is appended as ?token=, mirroring the AI attachment URLs.
 */
export const questionImageSrc = (
  url: string | undefined | null,
  apiBase: string,
  token?: string | null,
): string | undefined => {
  if (!url) return undefined;
  const full = url.startsWith('http') ? url : `${apiBase}${url}`;
  return token ? `${full}?token=${encodeURIComponent(token)}` : full;
};

/** Option label: 0 -> "A", 1 -> "B"… */
export const optionLetter = (index: number): string => String.fromCharCode(65 + index);

export const chronologyItemText = (q: QuestionResponse, itemIndex: number): string =>
  q.chronology_items?.find((i) => i.index === itemIndex)?.text ?? '';

export const orderedChronology = (items: ChronologyItem[] | undefined | null): ChronologyItem[] =>
  [...(items ?? [])].sort((a, b) => a.correct_order - b.correct_order);

/**
 * Markdown rendering of a question for sharing — the question text followed by
 * its options (choices as a lettered bullet list, chronology as an ordered
 * list). Answers are intentionally omitted so the copy stays spoiler-free.
 */
export const questionToMarkdown = (q: QuestionResponse): string => {
  const lines: string[] = [`##### ${q.question.trim()}`];

  if (q.options?.length) {
    lines.push('');
    for (const o of q.options) lines.push(`- **${optionLetter(o.index)}.** ${o.text}`);
  } else if (q.chronology_items?.length) {
    lines.push('', '_Arrange in the correct order:_', '');
    orderedChronology(q.chronology_items).forEach((item, seq) =>
      lines.push(`${seq + 1}. ${item.text}`),
    );
  }

  return lines.join('\n');
};

/* ------------------------------------------------------------------ */
/* "Ask AI" prompts                                                    */
/* ------------------------------------------------------------------ */

/**
 * Human-readable rendering of the official answer of a question, used as
 * context for the flashcard "Ask AI" actions. The card must already be
 * revealed — these helpers intentionally include the answer.
 */
export const questionOfficialAnswerText = (q: QuestionResponse): string => {
  if (q.options?.length) {
    const correct = q.options.filter((o) => o.correct);
    if (!correct.length) return 'No option marked correct';
    return correct.map((o) => `${optionLetter(o.index)}. ${o.text}`).join('\n');
  }
  if (q.chronology_items?.length) {
    return orderedChronology(q.chronology_items)
      .map((item) => `${item.correct_order}. ${item.text}`)
      .join('\n');
  }
  return q.expected_text?.trim() || 'No expected answer recorded';
};

/** Shared context block for both Ask AI modes. */
const questionAIContext = (q: QuestionResponse): string => {
  const scope = [q.subject, q.topic, q.sub_topic].filter(Boolean).join(' › ');
  const lines = [questionToMarkdown(q)];
  lines.push('', `**Official answer:**\n${questionOfficialAnswerText(q)}`);
  lines.push('', `**Explanation in the question bank:** ${q.explanation?.trim() || '(none provided)'}`);
  if (scope) lines.push('', `Syllabus scope: ${scope} · difficulty: ${q.difficulty}`);
  if (q.image_url) lines.push('', '_This question also has an attached image that you cannot see._');
  return lines.join('\n');
};

export const questionExplainPrompt = (q: QuestionResponse): string =>
  `You are an expert exam-prep tutor. Explain the following ${q.difficulty} ${formatQuestionType(
    q.type,
  )} question to a student preparing for competitive exams.

${questionAIContext(q)}

Structure your answer as:
1. **Concept tested** — the core idea behind the question, one or two sentences.
2. **Why the correct answer is right** — the reasoning step by step.
3. **Why the alternatives fail** — briefly address the wrong options / common traps (skip for non-choice formats).
4. **Remember it** — one short memory tip or related fact.

Keep it concise and readable in a small card.`;

export const questionCrosscheckPrompt = (q: QuestionResponse): string =>
  `You are an expert examiner auditing a question-bank entry for correctness.

${questionAIContext(q)}

Instructions:
1. First solve the question independently from scratch and state YOUR answer (ignore the official answer at this stage).
2. Then compare your answer against the official answer and the provided explanation.
3. Finish with a verdict line, exactly one of: **Verdict: Consistent**, **Verdict: Ambiguous**, or **Verdict: Likely error** — followed by the reason, quoting exact option/item text where relevant. Flag wrong official answers, explanations that contradict the official answer, multiple defensible correct options, or factual mistakes.`;

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
