<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Exam prep" title="Quiz" color="violet">
      <template #stats>
        <StatCard :value="stats?.total ?? 0" label="Questions" variant="dark" color="violet" />
        <StatCard :value="papersList.length" label="Papers" variant="primary" color="violet" />
      </template>
      <template #controls>
        <UserSelector id="quiz-user" v-model="selectedUserId" :users="users" color="violet" />
      </template>
    </PageHeader>

    <SelectUserPrompt
      title="Select a user to manage their question bank"
      :users-count="users.length"
      :selected-user-id="selectedUserId"
    />

    <template v-if="selectedUserId">
      <QuizTabs
        v-model="activeTab"
        class="mb-4"
        :question-count="stats?.total ?? questionList.length"
        :paper-count="papersList.length"
      />

      <LoadingSpinner v-if="showInitialLoader" message="Loading..." color="violet" />
      <ErrorBanner v-else-if="primaryError" :message="primaryError" :on-retry="handleRetry" />

      <template v-else>
        <QuestionDashboard
          v-if="activeTab === 'dashboard'"
          :stats="stats"
          :papers="papersList"
          @open-paper="openPaperFromDashboard"
          @navigate="activeTab = $event"
        />

        <QuestionList
          v-else-if="activeTab === 'questions'"
          :questions="questionList"
          :vocabulary="vocabulary"
          :has-active-filters="hasActiveFilters"
          v-model:search-query="searchQuery"
          v-model:filter-type="filterType"
          v-model:filter-difficulty="filterDifficulty"
          v-model:filter-tags="filterTags"
          v-model:filter-subject="filterSubject"
          @apply-filters="loadQuestions"
          @clear-filters="
            clearFilters();
            loadQuestions();
          "
          @refresh="loadQuestions"
          @add="openAddQuestion"
          @edit="openEditQuestion"
          @delete="removeQuestion"
        />

        <QuestionCards
          v-else-if="activeTab === 'cards'"
          ref="questionCards"
          :user-id="selectedUserId"
          :vocabulary="vocabulary"
          :on-difficulty-changed="loadStats"
          @edit="openEditQuestion"
        />

        <PaperGenerator
          v-else-if="activeTab === 'generate'"
          :vocabulary="vocabulary"
          :is-generating="isGenerating"
          @generate="generatePaperFlow"
        />

        <template v-else>
          <ErrorBanner v-if="papersError" :message="papersError" :on-retry="loadPapers" />
          <PaperList
            :papers="papersList"
            @open="openPaper"
            @attempt="attemptPaper"
            @delete="removePaper"
          />
        </template>
      </template>
    </template>

    <QuestionModal
      :open="isQuestionModalOpen"
      :question="editingQuestion"
      :vocabulary="vocabulary"
      :is-saving="isSavingQuestion"
      @close="closeQuestionModal"
      @save="saveQuestion"
    />

    <PaperDetail :paper="activePaper" @close="closePaper" />

    <PaperRunnerModal :open="isRunnerOpen" :paper="runnerPaper" @close="closeRunner" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useUser } from '../composables/useUser';
import { useQuizPage } from './quiz/composables/useQuizPage';
import type { QuestionResponse } from '../types';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';
import UserSelector from './ui/UserSelector.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import SelectUserPrompt from './ui/SelectUserPrompt.vue';
import QuizTabs from './quiz/QuizTabs.vue';
import QuestionDashboard from './quiz/dashboard/QuestionDashboard.vue';
import QuestionList from './quiz/questions/QuestionList.vue';
import QuestionModal from './quiz/questions/QuestionModal.vue';
import QuestionCards from './quiz/cards/QuestionCards.vue';
import PaperGenerator from './quiz/papers/generator/PaperGenerator.vue';
import PaperList from './quiz/papers/PaperList.vue';
import PaperDetail from './quiz/papers/PaperDetail.vue';
import PaperRunnerModal from './quiz/papers/PaperRunnerModal.vue';

const { users, currentUserId, setUser, clearUser } = useUser();
const selectedUserId = ref<number | null>(currentUserId.value);

const {
  questions: {
    questions: questionList,
    isLoading: isLoadingQuestions,
    error: questionsError,
    stats,
    vocabulary,
    filterType,
    filterDifficulty,
    filterTags,
    filterSubject,
    searchQuery,
    hasActiveFilters,
    clearFilters,
    loadQuestions,
    loadStats,
    refreshAll,
    removeQuestion,
  },
  papers: {
    papers: papersList,
    isGenerating,
    error: papersError,
    activePaper,
    loadPapers,
    openPaper,
    closePaper,
    removePaper,
  },
  activeTab,
  isQuestionModalOpen,
  editingQuestion,
  isSavingQuestion,
  openAddQuestion,
  openEditQuestion,
  closeQuestionModal,
  saveQuestion,
  runnerPaper,
  isRunnerOpen,
  attemptPaper,
  closeRunner,
  generatePaperFlow,
  openPaperFromDashboard,
} = useQuizPage(selectedUserId, {
  onQuestionEdited: (q) => questionCards.value?.syncUpdatedQuestion?.(q),
});

/** Flashcard session handle (exposed by QuestionCards). */
const questionCards = ref<{
  reset: () => void;
  syncUpdatedQuestion?: (q: QuestionResponse) => void;
} | null>(null);

watch(selectedUserId, (id) => {
  questionCards.value?.reset();
  if (id) setUser(id);
  else clearUser();
});

/** Only show the full-page loader for data-backed tabs before first paint. */
const showInitialLoader = computed(
  () =>
    isLoadingQuestions.value &&
    questionList.value.length === 0 &&
    activeTab.value !== 'papers' &&
    activeTab.value !== 'cards',
);

const primaryError = computed(() => questionsError.value);

const handleRetry = () => {
  refreshAll();
  loadPapers();
};
</script>
