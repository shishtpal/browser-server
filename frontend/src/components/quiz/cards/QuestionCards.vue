<template>
  <section class="space-y-4">
    <ErrorBanner
      v-if="error"
      :message="error"
      :on-retry="phase === 'reviewing' ? undefined : () => start()"
    />

    <!-- Idle: configure the session -->
    <ReviewSetupPanel
      v-if="phase === 'idle'"
      v-model:all-questions="allQuestions"
      v-model:limit="limit"
      v-model:selected-tags="selectedTags"
      :tag-options="tagOptions"
      :can-start="canStart"
      :nothing-due="nothingDue"
      @start="() => start()"
      @start-practice="() => start(true)"
    />

    <!-- Loading -->
    <LoadingSpinner
      v-else-if="phase === 'loading'"
      message="Loading review cards..."
      color="violet"
    />

    <!-- Reviewing -->
    <template v-else-if="phase === 'reviewing' && current">
      <ReviewHeader
        :current-number="reviewed + 1"
        :total="reviewed + items.length"
        :remaining="items.length - 1"
        :due-count="dueCount"
        :new-count="newCount"
        :all-questions="allQuestions"
        :selected-tags="selectedTags"
        :skipped-count="skippedCount"
        @end="end"
      />

      <ReviewCard
        :question="current.question"
        :revealed="answerRevealed"
        :is-rating="isRating"
        :is-saving-difficulty="isSavingDifficulty"
        :practice-mode="practiceMode"
        :can-skip="!!current"
        @reveal="reveal"
        @rate="submitRating"
        @difficulty-change="changeDifficulty($event as QuestionDifficulty)"
        @next="nextPractice"
        @skip="skip"
      />
    </template>

    <!-- Complete -->
    <ReviewComplete
      v-else-if="phase === 'complete'"
      :practice-mode="practiceMode"
      :reviewed="reviewed"
      :rating-counts="ratingCounts"
      :skipped-count="skippedCount"
      @again="() => start(practiceMode)"
      @change-tags="end"
    />

    <!-- Nothing due -->
    <EmptyState
      v-else
      title="Nothing is due for this selection"
      description="Change tags or return to Questions to add more cards."
      icon="clock"
      color="violet"
    />
  </section>
</template>

<script setup lang="ts">
import { toRef } from 'vue';
import type { QuestionDifficulty, TagVocabulary } from '../../../types';
import EmptyState from '../../ui/EmptyState.vue';
import ErrorBanner from '../../ui/ErrorBanner.vue';
import LoadingSpinner from '../../ui/LoadingSpinner.vue';
import ReviewCard from './ReviewCard.vue';
import ReviewComplete from './ReviewComplete.vue';
import ReviewHeader from './ReviewHeader.vue';
import ReviewSetupPanel from './ReviewSetupPanel.vue';
import { useQuestionCards } from '../composables/useQuestionCards';

const props = defineProps<{
  userId: number | null;
  vocabulary: TagVocabulary | null;
  onDifficultyChanged: () => void;
}>();

const cards = useQuestionCards(
  toRef(props, 'userId'),
  toRef(props, 'vocabulary'),
  props.onDifficultyChanged,
);

const {
  phase,
  error,
  start,
  nothingDue,
  practiceMode,
  allQuestions,
  tagOptions,
  selectedTags,
  limit,
  canStart,
  current,
  reviewed,
  items,
  dueCount,
  newCount,
  end,
  nextPractice,
  answerRevealed,
  isRating,
  isSavingDifficulty,
  skippedCount,
  reveal,
  submitRating,
  skip,
  changeDifficulty,
  ratingCounts,
} = cards;

defineExpose({ reset: cards.reset });
</script>
