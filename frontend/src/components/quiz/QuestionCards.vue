<template>
  <section class="space-y-4">
    <ErrorBanner
      v-if="error"
      :message="error"
      :on-retry="phase === 'reviewing' ? undefined : start"
    />

    <div
      v-if="phase === 'idle'"
      class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
    >
      <h2 class="text-lg font-black text-slate-800 dark:text-slate-100">Review cards</h2>
      <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">
        Choose tags to study. Multiple tags use any-match behavior.
      </p>
      <p
        v-if="nothingDue"
        class="mt-3 rounded-lg bg-slate-50 p-3 text-sm font-semibold text-slate-600 dark:bg-slate-900/50 dark:text-slate-300"
      >
        Nothing is due for this selection. You can practice these cards without changing their
        schedule.
      </p>
      <label
        class="mt-4 flex items-center gap-2 text-sm font-semibold text-slate-700 dark:text-slate-200"
        ><input v-model="allQuestions" type="checkbox" /> All questions</label
      >
      <div class="mt-3">
        <p class="text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400">
          Tags
        </p>
        <p v-if="!tagOptions.length" class="mt-2 text-sm text-slate-500">No question tags found</p>
        <div v-else class="mt-2 flex flex-wrap gap-2">
          <label
            v-for="tag in tagOptions"
            :key="tag"
            class="rounded-lg border px-2 py-1 text-xs font-semibold"
            :class="
              allQuestions
                ? 'cursor-not-allowed opacity-50'
                : 'border-slate-300 dark:border-slate-600'
            "
          >
            <input
              v-model="selectedTags"
              :disabled="allQuestions"
              type="checkbox"
              :value="tag"
              class="mr-1"
            />{{ tag }}
          </label>
        </div>
      </div>
      <label class="mt-4 block text-sm font-semibold text-slate-700 dark:text-slate-200"
        >Session size
        <select
          v-model.number="limit"
          class="ml-2 rounded-md border border-slate-300 bg-white px-2 py-1 text-sm dark:border-slate-600 dark:bg-slate-900"
        >
          <option :value="10">10</option>
          <option :value="20">20</option>
          <option :value="50">50</option>
        </select>
      </label>
      <Button
        class="mt-5"
        variant="gradient-violet"
        :disabled="!canStart"
        :loading="false"
        @click="() => start()"
        >Start review</Button
      >
      <Button v-if="nothingDue" class="mt-2" variant="secondary" @click="() => start(true)">
        Practice these cards anyway
      </Button>
    </div>

    <LoadingSpinner
      v-else-if="phase === 'loading'"
      message="Loading review cards..."
      color="violet"
    />

    <template v-else-if="phase === 'reviewing' && current">
      <header
        class="flex flex-wrap items-center gap-2 rounded-xl border border-gray-200 bg-white px-4 py-3 dark:border-slate-700 dark:bg-slate-800/60"
      >
        <p aria-live="polite" class="font-black text-slate-800 dark:text-slate-100">
          Card {{ reviewed + 1 }} of {{ reviewed + items.length }}
        </p>
        <span class="text-xs text-slate-500"
          >{{ items.length - 1 }} remaining · {{ dueCount }} due / {{ newCount }} new loaded</span
        >
        <div class="ml-auto flex flex-wrap gap-1">
          <span v-if="allQuestions" class="rounded bg-slate-100 px-2 py-1 text-xs dark:bg-slate-700"
            >All questions</span
          ><span
            v-for="tag in selectedTags"
            :key="tag"
            class="rounded bg-slate-100 px-2 py-1 text-xs dark:bg-slate-700"
            >{{ tag }}</span
          ><Button size="sm" variant="ghost" @click="end">End session</Button>
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

    <div
      v-else-if="phase === 'complete'"
      class="rounded-xl border border-emerald-200 bg-emerald-50 p-6 text-center dark:border-emerald-900/50 dark:bg-emerald-900/20"
    >
      <h2 class="text-lg font-black text-emerald-900 dark:text-emerald-100">
        {{ practiceMode ? 'Practice complete' : 'Session complete' }}
      </h2>
      <p class="mt-2 text-sm text-emerald-800 dark:text-emerald-200">
        <template v-if="practiceMode">
          You practiced the selected cards without changing their review schedule.
        </template>
        <template v-else>
          {{ reviewed }} cards reviewed · Again {{ ratingCounts.again }} · Hard
          {{ ratingCounts.hard }} · Good {{ ratingCounts.good }} · Easy
          {{ ratingCounts.easy }}
        </template>
      </p>
      <div class="mt-4 flex justify-center gap-2">
        <Button variant="gradient-violet" @click="() => start(practiceMode)">
          {{ practiceMode ? 'Practice more' : 'Review more' }} </Button
        ><Button variant="secondary" @click="end">Change tags</Button>
      </div>
    </div>

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
import type { QuestionDifficulty, TagVocabulary } from '../../types'
import { toRef } from 'vue'
import { useQuestionCards } from '../../composables/useQuestionCards'
import Button from '../ui/Button.vue'
import ErrorBanner from '../ui/ErrorBanner.vue'
import EmptyState from '../ui/EmptyState.vue'
import LoadingSpinner from '../ui/LoadingSpinner.vue'
import ReviewCard from './ReviewCard.vue'

const props = defineProps<{
  userId: number | null
  vocabulary: TagVocabulary | null
  onDifficultyChanged: () => void
}>()

const cards = useQuestionCards(
  toRef(props, 'userId'),
  toRef(props, 'vocabulary'),
  props.onDifficultyChanged,
)

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
} = cards

defineExpose({ reset: cards.reset })
</script>
