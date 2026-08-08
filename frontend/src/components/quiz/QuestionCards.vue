<template>
  <section class="space-y-4">
    <ErrorBanner
      v-if="error"
      :message="error"
      :on-retry="phase === 'reviewing' ? undefined : start"
    />

    <!-- IDLE PHASE: Setup Flashcard Session -->
    <div
      v-if="phase === 'idle'"
      class="overflow-hidden rounded-2xl border border-slate-200/80 bg-white shadow-sm dark:border-slate-700/80 dark:bg-slate-800/60"
    >
      <!-- Header -->
      <div
        class="border-b border-slate-100 bg-slate-50/50 px-5 py-4 dark:border-slate-700/50 dark:bg-slate-900/30"
      >
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div class="flex items-center gap-2.5">
              <div
                class="flex h-9 w-9 items-center justify-center rounded-xl bg-violet-100 text-violet-600 shadow-2xs dark:bg-violet-900/40 dark:text-violet-300"
              >
                <svg
                  class="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                  />
                </svg>
              </div>
              <div>
                <h2 class="text-lg font-black text-slate-800 dark:text-slate-100">
                  Review Flashcards
                </h2>
                <p class="text-xs text-slate-500 dark:text-slate-400">
                  Choose tags to study or practice all cards with spaced repetition.
                </p>
              </div>
            </div>
          </div>
          <div
            v-if="nothingDue"
            class="flex items-center gap-1.5 rounded-full border border-amber-200/80 bg-amber-50 px-3 py-1 text-xs font-semibold text-amber-700 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-300"
          >
            <svg
              class="h-3.5 w-3.5 shrink-0"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
            <span>Nothing due for this selection</span>
          </div>
        </div>
      </div>

      <div class="space-y-5 p-5">
        <!-- Mode & Session Size Controls -->
        <div class="grid grid-cols-1 gap-3.5 sm:grid-cols-2">
          <!-- All Questions Toggle -->
          <label
            class="relative flex cursor-pointer items-center justify-between rounded-xl border p-3.5 transition-all select-none"
            :class="
              allQuestions
                ? 'border-violet-400 bg-violet-50/60 ring-2 ring-violet-200 dark:border-violet-500 dark:bg-violet-950/30 dark:ring-violet-900/40'
                : 'border-slate-200 bg-white hover:border-slate-300 dark:border-slate-700 dark:bg-slate-900/40 dark:hover:border-slate-600'
            "
          >
            <div class="flex items-center gap-3">
              <input
                v-model="allQuestions"
                type="checkbox"
                class="h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-slate-600 dark:bg-slate-800"
              />
              <div>
                <span class="block text-xs font-bold text-slate-800 dark:text-slate-200"
                  >All Questions</span
                >
                <span class="block text-[11px] text-slate-500 dark:text-slate-400"
                  >Ignore tag filters and review all available cards</span
                >
              </div>
            </div>
          </label>

          <!-- Session Size Selector -->
          <div
            class="flex items-center justify-between rounded-xl border border-slate-200 bg-white p-3.5 dark:border-slate-700 dark:bg-slate-900/40"
          >
            <div>
              <span class="block text-xs font-bold text-slate-800 dark:text-slate-200"
                >Session Size</span
              >
              <span class="block text-[11px] text-slate-500 dark:text-slate-400"
                >Maximum cards per review batch</span
              >
            </div>
            <select
              v-model.number="limit"
              class="rounded-lg border border-slate-300 bg-slate-50 px-3 py-1.5 text-xs font-bold text-slate-700 transition outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-violet-900/30"
            >
              <option :value="10">10 cards</option>
              <option :value="20">20 cards</option>
              <option :value="50">50 cards</option>
            </select>
          </div>
        </div>

        <!-- Tags Selection Section -->
        <div class="space-y-2.5">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <span
                class="text-xs font-bold tracking-wider text-slate-500 uppercase dark:text-slate-400"
              >
                Filter by Tags
              </span>
              <span
                v-if="selectedTags.length && !allQuestions"
                class="rounded-full bg-violet-100 px-2 py-0.5 text-[10px] font-bold text-violet-700 dark:bg-violet-900/40 dark:text-violet-300"
              >
                {{ selectedTags.length }} selected
              </span>
            </div>

            <!-- Quick Batch Selection Controls -->
            <div
              v-if="!allQuestions && tagOptions.length > 0"
              class="flex items-center gap-1.5 text-xs"
            >
              <button
                type="button"
                class="rounded-md px-2 py-1 text-[11px] font-semibold text-violet-600 transition hover:bg-violet-50 dark:text-violet-400 dark:hover:bg-violet-950/40"
                @click="toggleAllFiltered"
              >
                {{ areAllFilteredSelected ? 'Deselect visible' : 'Select visible' }}
              </button>
              <span v-if="selectedTags.length" class="text-slate-300 dark:text-slate-700">•</span>
              <button
                v-if="selectedTags.length"
                type="button"
                class="rounded-md px-2 py-1 text-[11px] font-semibold text-slate-500 transition hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
                @click="clearAllTags"
              >
                Clear all
              </button>
            </div>
          </div>

          <!-- Tag Search Input -->
          <div v-if="tagOptions.length > 0" class="relative">
            <div
              class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-slate-400"
            >
              <svg
                class="h-3.5 w-3.5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
            </div>
            <input
              v-model="tagSearchQuery"
              :disabled="allQuestions"
              type="text"
              placeholder="Search tags..."
              class="w-full rounded-xl border border-slate-200 bg-slate-50/70 py-1.5 pr-8 pl-9 text-xs text-slate-800 placeholder-slate-400 transition outline-none focus:border-violet-400 focus:bg-white focus:ring-2 focus:ring-violet-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-700 dark:bg-slate-900/60 dark:text-slate-200 dark:placeholder-slate-500 dark:focus:border-violet-500 dark:focus:bg-slate-900 dark:focus:ring-violet-900/30"
            />
            <button
              v-if="tagSearchQuery"
              type="button"
              class="absolute inset-y-0 right-0 flex items-center pr-2.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
              @click="clearTagSearch"
            >
              <svg
                class="h-3.5 w-3.5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <!-- Scrollable Tags Box -->
          <div
            class="max-h-48 scrollbar-thin scrollbar-thumb-slate-300 overflow-y-auto rounded-xl border border-slate-200 bg-slate-50/40 p-3 dark:scrollbar-thumb-slate-600 dark:border-slate-700/80 dark:bg-slate-900/40"
            :class="{ 'pointer-events-none opacity-50': allQuestions }"
          >
            <p
              v-if="!tagOptions.length"
              class="py-4 text-center text-xs text-slate-500 dark:text-slate-400"
            >
              No question tags found in your question bank.
            </p>
            <div
              v-else-if="!filteredTags.length"
              class="py-4 text-center text-xs text-slate-500 dark:text-slate-400"
            >
              No tags match "<span class="font-semibold">{{ tagSearchQuery }}</span
              >"
              <button
                type="button"
                class="ml-1.5 text-violet-600 underline dark:text-violet-400"
                @click="clearTagSearch"
              >
                Clear search
              </button>
            </div>
            <div v-else class="flex flex-wrap gap-1.5">
              <label
                v-for="tag in filteredTags"
                :key="tag"
                class="group flex cursor-pointer items-center gap-1.5 rounded-lg border px-2.5 py-1 text-xs font-semibold transition-all select-none"
                :class="
                  selectedTags.includes(tag) && !allQuestions
                    ? 'border-violet-300 bg-violet-100/80 text-violet-800 shadow-2xs dark:border-violet-600/70 dark:bg-violet-950/60 dark:text-violet-200'
                    : 'border-slate-200/90 bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-100/60 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300 dark:hover:border-slate-600 dark:hover:bg-slate-700/50'
                "
              >
                <input
                  v-model="selectedTags"
                  :disabled="allQuestions"
                  type="checkbox"
                  :value="tag"
                  class="h-3.5 w-3.5 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-slate-600 dark:bg-slate-700"
                />
                <span>{{ tag }}</span>
              </label>
            </div>
          </div>
        </div>

        <!-- Action Buttons -->
        <div class="flex flex-wrap items-center gap-3 pt-2">
          <Button
            variant="gradient-violet"
            size="md"
            :disabled="!canStart"
            :loading="false"
            @click="() => start()"
          >
            Start review session
          </Button>

          <Button v-if="nothingDue" variant="secondary" size="md" @click="() => start(true)">
            Practice these cards anyway
          </Button>
        </div>
      </div>
    </div>

    <!-- LOADING PHASE -->
    <LoadingSpinner
      v-else-if="phase === 'loading'"
      message="Loading review cards..."
      color="violet"
    />

    <!-- REVIEWING PHASE -->
    <template v-else-if="phase === 'reviewing' && current">
      <header
        class="space-y-3 rounded-2xl border border-slate-200/80 bg-white p-4 shadow-sm dark:border-slate-700/80 dark:bg-slate-800/60"
      >
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <span class="h-2.5 w-2.5 animate-pulse rounded-full bg-emerald-500"></span>
            <p aria-live="polite" class="font-black text-slate-800 dark:text-slate-100">
              Card {{ reviewed + 1 }}
              <span class="text-sm font-normal text-slate-400"
                >of {{ reviewed + items.length }}</span
              >
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <span
              class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-600 dark:bg-slate-700/70 dark:text-slate-300"
            >
              {{ items.length - 1 }} remaining
            </span>
            <span
              class="rounded-full bg-amber-100 px-2.5 py-1 text-xs font-semibold text-amber-700 dark:bg-amber-950/50 dark:text-amber-300"
            >
              {{ dueCount }} due
            </span>
            <span
              class="rounded-full bg-blue-100 px-2.5 py-1 text-xs font-semibold text-blue-700 dark:bg-blue-950/50 dark:text-blue-300"
            >
              {{ newCount }} new
            </span>
            <Button
              size="sm"
              variant="ghost"
              class="ml-1 text-slate-500 hover:text-slate-700 dark:text-slate-400"
              @click="end"
            >
              End session
            </Button>
          </div>
        </div>

        <!-- Progress Bar -->
        <div class="h-1.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-700">
          <div
            class="h-full bg-gradient-to-r from-violet-500 to-indigo-500 transition-all duration-300"
            :style="{
              width: `${Math.min(100, Math.round((reviewed / (reviewed + items.length)) * 100))}%`,
            }"
          ></div>
        </div>

        <!-- Active Filter Pills -->
        <div
          v-if="allQuestions || selectedTags.length"
          class="flex items-center gap-1.5 overflow-x-auto py-0.5 text-xs"
        >
          <span class="text-[11px] font-bold text-slate-400 uppercase">Filters:</span>
          <span
            v-if="allQuestions"
            class="rounded-md bg-violet-100 px-2 py-0.5 text-[11px] font-semibold text-violet-700 dark:bg-violet-900/40 dark:text-violet-300"
          >
            All questions
          </span>
          <span
            v-for="tag in selectedTags"
            :key="tag"
            class="rounded-md bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-600 dark:bg-slate-700 dark:text-slate-300"
          >
            {{ tag }}
          </span>
        </div>
      </header>

      <ReviewCard
        :question="current.question"
        :revealed="answerRevealed"
        :is-rating="isRating"
        :is-saving-difficulty="isSavingDifficulty"
        :practice-mode="practiceMode"
        @reveal="reveal"
        @rate="submitRating"
        @difficulty-change="changeDifficulty($event as QuestionDifficulty)"
        @next="nextPractice"
      />
    </template>

    <!-- COMPLETE PHASE -->
    <div
      v-else-if="phase === 'complete'"
      class="overflow-hidden rounded-2xl border border-emerald-200/80 bg-gradient-to-b from-emerald-50/70 to-emerald-100/40 p-6 text-center shadow-sm dark:border-emerald-800/60 dark:from-emerald-950/30 dark:to-slate-900/60"
    >
      <div
        class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-900/50 dark:text-emerald-300"
      >
        <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
      </div>

      <h2 class="mt-3 text-xl font-black text-emerald-900 dark:text-emerald-100">
        {{ practiceMode ? 'Practice Complete!' : 'Session Complete!' }}
      </h2>

      <p v-if="practiceMode" class="mt-1 text-sm text-emerald-800 dark:text-emerald-300">
        You practiced the selected cards without altering their spaced repetition schedule.
      </p>
      <p v-else class="mt-1 text-sm text-emerald-800 dark:text-emerald-300">
        Great job! You reviewed <span class="font-bold">{{ reviewed }}</span> cards in this session.
      </p>

      <!-- Rating Breakdown Badges -->
      <div
        v-if="!practiceMode && reviewed > 0"
        class="mx-auto mt-5 grid max-w-lg grid-cols-2 gap-2 sm:grid-cols-4"
      >
        <div
          class="rounded-xl border border-rose-200 bg-rose-50/80 p-2.5 dark:border-rose-900/40 dark:bg-rose-950/30"
        >
          <span class="block text-[11px] font-bold text-rose-600 uppercase dark:text-rose-400"
            >Again</span
          >
          <span class="text-lg font-black text-rose-700 dark:text-rose-300">{{
            ratingCounts.again
          }}</span>
        </div>
        <div
          class="rounded-xl border border-amber-200 bg-amber-50/80 p-2.5 dark:border-amber-900/40 dark:bg-amber-950/30"
        >
          <span class="block text-[11px] font-bold text-amber-600 uppercase dark:text-amber-400"
            >Hard</span
          >
          <span class="text-lg font-black text-amber-700 dark:text-amber-300">{{
            ratingCounts.hard
          }}</span>
        </div>
        <div
          class="rounded-xl border border-sky-200 bg-sky-50/80 p-2.5 dark:border-sky-900/40 dark:bg-sky-950/30"
        >
          <span class="block text-[11px] font-bold text-sky-600 uppercase dark:text-sky-400"
            >Good</span
          >
          <span class="text-lg font-black text-sky-700 dark:text-sky-300">{{
            ratingCounts.good
          }}</span>
        </div>
        <div
          class="rounded-xl border border-emerald-200 bg-emerald-100/60 p-2.5 dark:border-emerald-800/50 dark:bg-emerald-950/50"
        >
          <span class="block text-[11px] font-bold text-emerald-700 uppercase dark:text-emerald-400"
            >Easy</span
          >
          <span class="text-lg font-black text-emerald-800 dark:text-emerald-300">{{
            ratingCounts.easy
          }}</span>
        </div>
      </div>

      <div class="mt-6 flex justify-center gap-3">
        <Button variant="gradient-violet" size="md" @click="() => start(practiceMode)">
          {{ practiceMode ? 'Practice more' : 'Review more' }}
        </Button>
        <Button variant="secondary" size="md" @click="end">Change tags</Button>
      </div>
    </div>

    <!-- EMPTY STATE -->
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
import type { QuestionDifficulty, TagVocabulary } from '../../types';
import { computed, ref, toRef } from 'vue';
import Button from '../ui/Button.vue';
import EmptyState from '../ui/EmptyState.vue';
import ErrorBanner from '../ui/ErrorBanner.vue';
import LoadingSpinner from '../ui/LoadingSpinner.vue';
import ReviewCard from './ReviewCard.vue';
import { useQuestionCards } from '../../composables/useQuestionCards';

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
  reveal,
  submitRating,
  changeDifficulty,
  ratingCounts,
} = cards;

// Tag Search & Selection Management
const tagSearchQuery = ref('');

const filteredTags = computed(() => {
  const query = tagSearchQuery.value.trim().toLowerCase();
  if (!query) return tagOptions.value;
  return tagOptions.value.filter((tag) => tag.toLowerCase().includes(query));
});

const areAllFilteredSelected = computed(() => {
  if (!filteredTags.value.length) return false;
  return filteredTags.value.every((tag) => selectedTags.value.includes(tag));
});

const toggleAllFiltered = () => {
  if (allQuestions.value) return;
  if (areAllFilteredSelected.value) {
    const toRemove = new Set(filteredTags.value);
    selectedTags.value = selectedTags.value.filter((tag) => !toRemove.has(tag));
  } else {
    const combined = new Set([...selectedTags.value, ...filteredTags.value]);
    selectedTags.value = Array.from(combined);
  }
};

const clearAllTags = () => {
  if (allQuestions.value) return;
  selectedTags.value = [];
};

const clearTagSearch = () => {
  tagSearchQuery.value = '';
};

defineExpose({ reset: cards.reset });
</script>
