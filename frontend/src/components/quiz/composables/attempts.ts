/**
 * Paper attempt records — localStorage persistence + shared types.
 *
 * Attempts are kept client-side only (no backend involvement); each paper id
 * maps to a list of attempts, newest first, capped per paper.
 */

export interface UserAnswer {
  singleChoice?: number;
  multipleChoice?: number[];
  inputText?: string;
  chronologyOrder?: number[]; // list of item indices in user-ordered sequence
}

export interface QuestionAttemptResult {
  questionId: number;
  isCorrect: boolean;
  score: number;
  maxScore: number;
  userAnswer: UserAnswer;
  expectedAnswerText: string;
}

export interface PaperAttemptRecord {
  paperId: number;
  paperTitle: string;
  totalQuestions: number;
  score: number;
  maxScore: number;
  percentage: number;
  correctCount: number;
  incorrectCount: number;
  unansweredCount: number;
  durationSeconds: number;
  completedAt: string;
  results: QuestionAttemptResult[];
}

const ATTEMPTS_STORAGE_KEY = 'bs_quiz_paper_attempts';
const MAX_ATTEMPTS_PER_PAPER = 10;

export function getPaperAttemptsMap(): Record<number, PaperAttemptRecord[]> {
  try {
    const raw = localStorage.getItem(ATTEMPTS_STORAGE_KEY);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

/** Newest attempt for a paper, or null. */
export function getLatestPaperAttempt(paperId: number): PaperAttemptRecord | null {
  const list = getPaperAttemptsMap()[paperId] || [];
  return list.length > 0 ? list[0] : null;
}

/** Highest-percentage attempt for a paper, or null. */
export function getBestPaperAttempt(paperId: number): PaperAttemptRecord | null {
  const list = getPaperAttemptsMap()[paperId] || [];
  if (list.length === 0) return null;
  return [...list].sort((a, b) => b.percentage - a.percentage)[0];
}

/** Attempt count for a paper. */
export function getPaperAttemptCount(paperId: number): number {
  return (getPaperAttemptsMap()[paperId] || []).length;
}

export function savePaperAttempt(record: PaperAttemptRecord): void {
  try {
    const map = getPaperAttemptsMap();
    if (!map[record.paperId]) map[record.paperId] = [];
    map[record.paperId].unshift(record); // newest first
    if (map[record.paperId].length > MAX_ATTEMPTS_PER_PAPER) {
      map[record.paperId] = map[record.paperId].slice(0, MAX_ATTEMPTS_PER_PAPER);
    }
    localStorage.setItem(ATTEMPTS_STORAGE_KEY, JSON.stringify(map));
  } catch (e) {
    console.error('Failed to save paper attempt', e);
  }
}
