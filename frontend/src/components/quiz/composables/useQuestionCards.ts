import { computed, ref, type Ref } from 'vue';
import { getQuestionCards, reviewQuestion, updateQuestion } from '../../../lib/api';
import type {
  QuestionCardItem,
  QuestionDifficulty,
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
  const ratingCounts = ref<Record<ReviewRating, number>>({ again: 0, hard: 0, good: 0, easy: 0 });
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
    ratingCounts.value = { again: 0, hard: 0, good: 0, easy: 0 };
  };

  const start = async (practice = false) => {
    if (!userId.value || !canStart.value) return;
    const activeSession = ++session;
    phase.value = 'loading';
    error.value = null;
    try {
      practiceMode.value = practice;
      const queue = await getQuestionCards(userId.value, {
        tags: allQuestions.value ? undefined : selectedTags.value,
        limit: limit.value,
        practice,
      });
      if (activeSession !== session || userId.value === null) return;
      items.value = queue.items;
      dueCount.value = queue.due_count;
      newCount.value = queue.new_count;
      answerRevealed.value = false;
      nothingDue.value = queue.items.length === 0;
      ratingCounts.value = { again: 0, hard: 0, good: 0, easy: 0 };
      phase.value = queue.items.length ? 'reviewing' : 'idle';
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
    error.value = null;
    phase.value = 'idle';
  };
  const nextPractice = () => {
    items.value.shift();
    answerRevealed.value = false;
    if (!items.value.length) phase.value = 'complete';
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

  return {
    selectedTags,
    allQuestions,
    limit,
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
    ratingCounts,
    current,
    reviewed,
    canStart,
    tagOptions,
    reset,
    start,
    end,
    nextPractice,
    reveal,
    submitRating,
    changeDifficulty,
  };
}
