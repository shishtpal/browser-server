import { computed, onScopeDispose, ref } from 'vue';
import type { QuestionPaper, QuestionResponse } from '../../../types';
import {
  savePaperAttempt,
  type PaperAttemptRecord,
  type QuestionAttemptResult,
  type UserAnswer,
} from './attempts';

export type { PaperAttemptRecord, QuestionAttemptResult, UserAnswer } from './attempts';
export {
  getBestPaperAttempt,
  getLatestPaperAttempt,
  getPaperAttemptCount,
  getPaperAttemptsMap,
} from './attempts';

/**
 * Exam-runner state machine: tracks answers, flags, the elapsed-time timer and
 * produces a scored PaperAttemptRecord on submission.
 */
export function usePaperAttempt() {
  const activePaper = ref<QuestionPaper | null>(null);
  const isExamActive = ref(false);
  const isExamSubmitted = ref(false);
  const currentQuestionIndex = ref(0);

  /** Question ID -> UserAnswer */
  const answers = ref<Record<number, UserAnswer>>({});
  /** Question ID -> flagged for review */
  const flagged = ref<Record<number, boolean>>({});

  const startTime = ref(0);
  const elapsedTime = ref(0);
  let timerInterval: ReturnType<typeof setInterval> | null = null;

  const attemptRecord = ref<PaperAttemptRecord | null>(null);

  /* ------------------------------ lifecycle ------------------------------ */

  const startExam = (paper: QuestionPaper) => {
    activePaper.value = paper;
    isExamActive.value = true;
    isExamSubmitted.value = false;
    currentQuestionIndex.value = 0;
    answers.value = {};
    flagged.value = {};
    attemptRecord.value = null;

    // Chronology questions start from the display order of their items.
    for (const q of paper.questions || []) {
      if (q.type === 'chronology' && q.chronology_items?.length) {
        answers.value[q.id] = {
          chronologyOrder: q.chronology_items.map((item) => item.index),
        };
      }
    }

    startTime.value = Date.now();
    elapsedTime.value = 0;
    stopTimer();
    timerInterval = setInterval(() => {
      elapsedTime.value = Math.floor((Date.now() - startTime.value) / 1000);
    }, 1000);
  };

  const stopTimer = () => {
    if (timerInterval) {
      clearInterval(timerInterval);
      timerInterval = null;
    }
  };

  onScopeDispose(stopTimer);

  const closeExam = () => {
    stopTimer();
    activePaper.value = null;
    isExamActive.value = false;
    isExamSubmitted.value = false;
  };

  /* ------------------------------- answers ------------------------------- */

  const setSingleChoice = (questionId: number, optionIndex: number) => {
    answers.value[questionId] = { ...answers.value[questionId], singleChoice: optionIndex };
  };

  const toggleMultipleChoice = (questionId: number, optionIndex: number) => {
    const current = answers.value[questionId]?.multipleChoice || [];
    const updated = current.includes(optionIndex)
      ? current.filter((i) => i !== optionIndex)
      : [...current, optionIndex];
    answers.value[questionId] = { ...answers.value[questionId], multipleChoice: updated };
  };

  const setInputText = (questionId: number, text: string) => {
    answers.value[questionId] = { ...answers.value[questionId], inputText: text };
  };

  const moveChronologyItem = (questionId: number, fromIndex: number, toIndex: number) => {
    const current = [...(answers.value[questionId]?.chronologyOrder || [])];
    if (fromIndex < 0 || fromIndex >= current.length || toIndex < 0 || toIndex >= current.length) {
      return;
    }
    const [moved] = current.splice(fromIndex, 1);
    current.splice(toIndex, 0, moved);
    answers.value[questionId] = { ...answers.value[questionId], chronologyOrder: current };
  };

  const toggleFlag = (questionId: number) => {
    flagged.value[questionId] = !flagged.value[questionId];
  };

  const isQuestionAnswered = (q: QuestionResponse): boolean => {
    const ans = answers.value[q.id];
    if (!ans) return false;
    if (q.type === 'single_choice') return ans.singleChoice !== undefined;
    if (q.type === 'multiple_choice') return (ans.multipleChoice?.length ?? 0) > 0;
    if (q.type === 'input') return Boolean(ans.inputText && ans.inputText.trim().length > 0);
    if (q.type === 'chronology') return Boolean(ans.chronologyOrder?.length);
    return false;
  };

  /* ------------------------------ evaluation ----------------------------- */

  const evaluateQuestion = (q: QuestionResponse): QuestionAttemptResult => {
    const userAns = answers.value[q.id] || {};
    let isCorrect = false;
    let expectedText = '';

    if (q.type === 'single_choice') {
      const correctIdx = q.options?.findIndex((o) => o.correct) ?? q.correct_answers?.[0] ?? -1;
      isCorrect = userAns.singleChoice !== undefined && userAns.singleChoice === correctIdx;
      const correctOpt = q.options?.find((o) => o.index === correctIdx);
      expectedText = correctOpt
        ? `${String.fromCharCode(65 + correctOpt.index)}. ${correctOpt.text}`
        : 'N/A';
    } else if (q.type === 'multiple_choice') {
      const correctSet = new Set(
        q.options?.filter((o) => o.correct).map((o) => o.index) ?? q.correct_answers ?? [],
      );
      const userSet = new Set(userAns.multipleChoice || []);
      isCorrect =
        correctSet.size === userSet.size && Array.from(correctSet).every((v) => userSet.has(v));
      const correctOpts = q.options?.filter((o) => correctSet.has(o.index)) || [];
      expectedText = correctOpts
        .map((o) => `${String.fromCharCode(65 + o.index)}. ${o.text}`)
        .join(' | ');
    } else if (q.type === 'input') {
      const expected = (q.expected_text || '').trim().toLowerCase();
      const actual = (userAns.inputText || '').trim().toLowerCase();
      isCorrect = expected.length > 0 && expected === actual;
      expectedText = q.expected_text || '';
    } else if (q.type === 'chronology') {
      const items = q.chronology_items || [];
      const correctOrder = [...items].sort((a, b) => a.correct_order - b.correct_order);
      const correctOrderIndices = correctOrder.map((i) => i.index);
      const userOrder = userAns.chronologyOrder || [];
      isCorrect =
        correctOrderIndices.length > 0 &&
        correctOrderIndices.length === userOrder.length &&
        correctOrderIndices.every((val, i) => val === userOrder[i]);
      expectedText = correctOrder.map((item, idx) => `${idx + 1}. ${item.text}`).join(' → ');
    }

    return {
      questionId: q.id,
      isCorrect,
      score: isCorrect ? 1 : 0,
      maxScore: 1,
      userAnswer: userAns,
      expectedAnswerText: expectedText,
    };
  };

  const submitExam = (): PaperAttemptRecord | null => {
    if (!activePaper.value?.questions) return null;
    stopTimer();

    const questions = activePaper.value.questions;
    const results = questions.map(evaluateQuestion);

    let correctCount = 0;
    let incorrectCount = 0;
    let unansweredCount = 0;

    questions.forEach((q, idx) => {
      const res = results[idx];
      if (!isQuestionAnswered(q)) unansweredCount++;
      else if (res.isCorrect) correctCount++;
      else incorrectCount++;
    });

    const totalQuestions = questions.length;
    const score = correctCount;
    const maxScore = totalQuestions;
    const percentage = totalQuestions > 0 ? Math.round((score / maxScore) * 100) : 0;

    const record: PaperAttemptRecord = {
      paperId: activePaper.value.id,
      paperTitle: activePaper.value.title,
      totalQuestions,
      score,
      maxScore,
      percentage,
      correctCount,
      incorrectCount,
      unansweredCount,
      durationSeconds: elapsedTime.value,
      completedAt: new Date().toISOString(),
      results,
    };

    attemptRecord.value = record;
    savePaperAttempt(record);
    isExamSubmitted.value = true;
    return record;
  };

  /* ------------------------------ selectors ------------------------------ */

  const answeredCount = computed(() => {
    if (!activePaper.value?.questions) return 0;
    return activePaper.value.questions.filter(isQuestionAnswered).length;
  });

  const flaggedCount = computed(() => Object.values(flagged.value).filter(Boolean).length);

  return {
    activePaper,
    isExamActive,
    isExamSubmitted,
    currentQuestionIndex,
    answers,
    flagged,
    elapsedTime,
    attemptRecord,
    startExam,
    closeExam,
    submitExam,
    setSingleChoice,
    toggleMultipleChoice,
    setInputText,
    moveChronologyItem,
    toggleFlag,
    isQuestionAnswered,
    answeredCount,
    flaggedCount,
  };
}
