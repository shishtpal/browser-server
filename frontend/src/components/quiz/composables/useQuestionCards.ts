import { computed, ref, type Ref } from 'vue';
import {
  getQuestionCards,
  getQuizSettings,
  reviewQuestion,
  skipQuestionCard,
  updateQuestion,
} from '../../../lib/api';
import type {
  CardFilterMode,
  QuestionCardItem,
  QuestionDifficulty,
  QuestionResponse,
  QuizScheduler,
  ReviewRating,
  TagVocabulary,
} from '../../../types';

export function useQuestionCards(
  userId: Ref<number | null>,
  vocabulary: Ref<TagVocabulary | null>,
  onDifficultyChanged?: () => void,
) {
  const selectedTags = ref<string[]>([]);
  const allQuestions = ref(false);
  const limit = ref(20);
  const mode = ref<CardFilterMode | ''>('');
  const skippedTagIDs = ref<Set<string>>(new Set());
  const items = ref<QuestionCardItem[]>([]);
  const dueCount = ref(0);
  const newCount = ref(0);
  const phase = ref<'idle' | 'loading' | 'reviewing' | 'complete'>('idle');
  const error = ref<string | null>(null);
  const answerRevealed = ref(false);
  const nothingDue = ref(false);
  const practiceMode = ref(false);
  const isRating = ref(false);
  const isSavingDifficulty = ref(false);
  const skippedCount = ref(0);
  const skippedQuestionIDs = new Set<number>();
  const ratingCounts = ref<Record<ReviewRating, number>>({ again: 0, hard: 0, good: 0, easy: 0 });
  const scheduler = ref<QuizScheduler>('sm2');
  let session = 0;

  const current = computed(() => items.value[0] ?? null);
  const reviewed = computed(() =>
    Object.values(ratingCounts.value).reduce((sum, count) => sum + count, 0),
  );
  const canStart = computed(() => allQuestions.value || selectedTags.value.length > 0);
  const tagOptions = computed(() => vocabulary.value?.tags ?? []);

  const reset = () => {
    session++;
    selectedTags.value = [];
    allQuestions.value = false;
    mode.value = '';
    skippedTagIDs.value = new Set();
    items.value = [];
    dueCount.value = 0;
    newCount.value = 0;
    phase.value = 'idle';
    error.value = null;
    answerRevealed.value = false;
    nothingDue.value = false;
    practiceMode.value = false;
    isRating.value = false;
    isSavingDifficulty.value = false;
    skippedCount.value = 0;
    skippedQuestionIDs.clear();
    ratingCounts.value = { again: 0, hard: 0, good: 0, easy: 0 };
  };

  const start = async (practice = false) => {
    if (!userId.value || !canStart.value) return;
    const activeSession = ++session;
    phase.value = 'loading';
    error.value = null;
    try {
      practiceMode.value = practice;
      const uid = userId.value;
      const settings = await getQuizSettings(uid).catch(() => null);
      if (activeSession !== session) return;
      if (settings) scheduler.value = settings.scheduler;
      const queue = await getQuestionCards(uid, {
        tags: allQuestions.value ? undefined : selectedTags.value,
        limit: limit.value,
        practice,
        mode: mode.value || undefined,
      });
      if (activeSession !== session || userId.value === null) return;
      items.value = filterIgnoredTags(queue.items);
      dueCount.value = queue.due_count;
      newCount.value = queue.new_count;
      answerRevealed.value = false;
      const visibleCount = items.value.length;
      nothingDue.value = visibleCount === 0;
      skippedCount.value = 0;
      skippedQuestionIDs.clear();
      ratingCounts.value = { again: 0, hard: 0, good: 0, easy: 0 };
      phase.value = visibleCount ? 'reviewing' : 'idle';
    } catch (e) {
      if (activeSession === session) {
        phase.value = 'idle';
        error.value = e instanceof Error ? e.message : 'Failed to load review cards';
      }
    }
  };

  const end = () => {
    session++;
    items.value = [];
    answerRevealed.value = false;
    nothingDue.value = false;
    practiceMode.value = false;
    skippedCount.value = 0;
    skippedQuestionIDs.clear();
    error.value = null;
    phase.value = 'idle';
  };
  const nextPractice = () => {
    if (current.value) skippedQuestionIDs.delete(current.value.question.id);
    items.value.shift();
    answerRevealed.value = false;
    if (!items.value.length) phase.value = 'complete';
  };

  /** Tags the user chose to ignore surface here; dropping an item from the
   *  queue does not cancel practice mode. */
  const ignoreTag = (tag: string) => {
    if (!tag) return;
    skippedTagIDs.value = new Set([...skippedTagIDs.value, tag]);
    items.value = items.value.filter((i) => !i.question.tags?.includes(tag));
    if (!items.value.length) phase.value = 'complete';
  };

  const clearIgnoredTags = () => {
    skippedTagIDs.value = new Set();
  };

  const filterIgnoredTags = (list: QuestionCardItem[]) => {
    const ignored = skippedTagIDs.value;
    if (!ignored.size) return list;
    return list.filter((i) => !i.question.tags?.some((t) => ignored.has(t)));
  };

  /** Persist the skip so tomorrow's Only Skipped can collect these. Still
   *  rotates to the back of the session queue for local fairness. */
  const skip = async () => {
    if (!current.value || !userId.value || isRating.value) return;
    const currentId = current.value.question.id;
    skippedQuestionIDs.add(currentId);
    skippedCount.value++;
    if (items.value.length > 1) {
      items.value.push(items.value.shift()!);
    } else {
      items.value.shift();
      phase.value = 'complete';
    }
    answerRevealed.value = false;
    if (skippedQuestionIDs.size >= items.value.length) phase.value = 'complete';
    try {
      await skipQuestionCard(currentId, { user_id: userId.value });
    } catch (e) {
      console.warn('skip persist failed; continuing session', e);
    }
  };
  const reveal = () => {
    answerRevealed.value = true;
  };

  const submitRating = async (rating: ReviewRating) => {
    if (!current.value || !userId.value || isRating.value) return;
    isRating.value = true;
    error.value = null;
    try {
      await reviewQuestion(current.value.question.id, { user_id: userId.value, rating });
      skippedQuestionIDs.delete(current.value.question.id);
      items.value.shift();
      ratingCounts.value[rating]++;
      answerRevealed.value = false;
      if (!items.value.length) phase.value = 'complete';
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to save review; please try again.';
    } finally {
      isRating.value = false;
    }
  };

  const changeDifficulty = async (difficulty: QuestionDifficulty) => {
    if (!current.value || isSavingDifficulty.value) return;
    const item = current.value;
    if (item.question.difficulty === difficulty) return;
    isSavingDifficulty.value = true;
    error.value = null;
    try {
      const updated = await updateQuestion(item.question.id, { difficulty });
      item.question = updated;
      onDifficultyChanged?.();
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update question difficulty';
    } finally {
      isSavingDifficulty.value = false;
    }
  };

  /** Replace any queued copy of an externally-edited question (e.g. Quick Edit
   *  from the modal). No-ops when the id is not in the queue. */
  const syncUpdatedQuestion = (updated: QuestionResponse) => {
    const hit = items.value.find((i) => i.question.id === updated.id);
    if (hit) hit.question = updated;
  };

  return {
    selectedTags,
    allQuestions,
    limit,
    mode,
    skippedTagIDs,
    items,
    dueCount,
    newCount,
    phase,
    error,
    answerRevealed,
    nothingDue,
    practiceMode,
    isRating,
    isSavingDifficulty,
    skippedCount,
    ratingCounts,
    scheduler,
    current,
    reviewed,
    canStart,
    tagOptions,
    reset,
    start,
    end,
    nextPractice,
    skip,
    ignoreTag,
    clearIgnoredTags,
    reveal,
    submitRating,
    changeDifficulty,
    syncUpdatedQuestion,
  };
}
