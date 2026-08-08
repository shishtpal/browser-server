<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Exam prep" title="Quiz" color="violet">
      <template #stats>
        <StatCard :value="stats?.total ?? 0" label="Questions" variant="dark" color="violet" />
        <StatCard :value="papers.length" label="Papers" variant="primary" color="violet" />
      </template>
      <template #controls>
        <UserSelector id="quiz-user" v-model="selectedUserId" :users="users" color="violet" />
        <div class="flex gap-1">
          <FilterPill v-for="tab in tabs" :key="tab.key" :active="activeTab === tab.key" @click="activeTab = tab.key">
            {{ tab.label }}
          </FilterPill>
        </div>
      </template>
    </PageHeader>

    <SelectUserPrompt title="Select a user to manage their question bank" :users-count="users.length" :selected-user-id="selectedUserId" />

    <template v-if="selectedUserId">
      <LoadingSpinner v-if="isLoading && questions.length === 0 && activeTab !== 'papers' && activeTab !== 'cards'" message="Loading..." color="violet" />
      <ErrorBanner v-else-if="error" :message="error" :on-retry="refreshAll" />

      <template v-else>
        <QuestionDashboard
          v-if="activeTab === 'dashboard'"
          :stats="stats"
          :papers="papers"
          @open-paper="openPaperAndSwitch"
        />

        <div v-else-if="activeTab === 'questions'" class="space-y-4">
          <div class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60">
            <h2 class="mb-3 text-sm font-black text-slate-800 dark:text-slate-100">Add question</h2>
            <QuestionForm :vocabulary="vocabulary" :is-saving="isSaving" @save="handleCreate" />
          </div>
          <QuestionList
            :questions="questions"
            :vocabulary="vocabulary"
            v-model:search-query="searchQuery"
            v-model:filter-type="filterType"
            v-model:filter-difficulty="filterDifficulty"
            v-model:filter-tags="filterTags"
            v-model:filter-subject="filterSubject"
            @apply-filters="loadQuestions"
            @edit="editing = $event"
            @delete="removeQuestion"
          />
        </div>

        <QuestionCards v-else-if="activeTab === 'cards'" ref="questionCards" :user-id="selectedUserId" :vocabulary="vocabulary" :on-difficulty-changed="loadStats" />

        <PaperGenerator
          v-else-if="activeTab === 'generate'"
          :vocabulary="vocabulary"
          :is-generating="isGenerating"
          @generate="handleGenerate"
        />

        <PaperList
          v-else
          :papers="papers"
          @open="openPaper"
          @delete="removePaper"
        />
      </template>
    </template>

    <QuestionEditModal
      :question="editing"
      :vocabulary="vocabulary"
      :is-saving="isSaving"
      @close="editing = null"
      @save="handleEditSave"
    />

    <PaperDetail :paper="activePaper" @close="closePaper" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useUser } from '../composables/useUser'
import { useQuestions } from '../composables/useQuestions'
import { useQuizPapers } from '../composables/useQuizPapers'
import PageHeader from './ui/PageHeader.vue'
import StatCard from './ui/StatCard.vue'
import UserSelector from './ui/UserSelector.vue'
import FilterPill from './ui/FilterPill.vue'
import LoadingSpinner from './ui/LoadingSpinner.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import SelectUserPrompt from './ui/SelectUserPrompt.vue'
import QuestionDashboard from './quiz/QuestionDashboard.vue'
import QuestionForm from './quiz/QuestionForm.vue'
import QuestionList from './quiz/QuestionList.vue'
import QuestionEditModal from './quiz/QuestionEditModal.vue'
import PaperGenerator from './quiz/PaperGenerator.vue'
import PaperList from './quiz/PaperList.vue'
import PaperDetail from './quiz/PaperDetail.vue'
import QuestionCards from './quiz/QuestionCards.vue'
import type { QuestionResponse, QuestionPaperSection } from '../types'

const tabs = [
  { key: 'dashboard', label: 'Dashboard' },
  { key: 'questions', label: 'Questions' },
  { key: 'cards', label: 'Cards' },
  { key: 'generate', label: 'Generate Paper' },
  { key: 'papers', label: 'Papers' },
] as const

type TabKey = (typeof tabs)[number]['key']
const activeTab = ref<TabKey>('dashboard')

const { users, currentUserId, setUser, clearUser } = useUser()
const selectedUserId = ref<number | null>(currentUserId.value)

const {
  questions,
  isLoading,
  error,
  stats,
  vocabulary,
  filterType,
  filterDifficulty,
  filterTags,
  filterSubject,
  searchQuery,
  loadQuestions,
  loadStats,
  refreshAll,
  addQuestion,
  editQuestion,
  removeQuestion,
} = useQuestions(selectedUserId)

const {
  papers,
  isGenerating,
  activePaper,
  loadPapers,
  generate,
  openPaper,
  closePaper,
  removePaper,
} = useQuizPapers(selectedUserId)

const editing = ref<QuestionResponse | null>(null)
const isSaving = ref(false)
const questionCards = ref<{ reset: () => void } | null>(null)

watch(selectedUserId, (id) => {
	questionCards.value?.reset()
  if (id) {
    setUser(id)
  } else {
    clearUser()
  }
})

const handleCreate = async (payload: Record<string, unknown>, image: File | null) => {
  isSaving.value = true
  try {
    await addQuestion(payload as never, image)
  } finally {
    isSaving.value = false
  }
}

const handleEditSave = async (id: number, payload: Record<string, unknown>, image: File | null) => {
  isSaving.value = true
  try {
    const resp = await editQuestion(id, payload as never, image)
    if (resp) editing.value = null
  } finally {
    isSaving.value = false
  }
}

const handleGenerate = async (input: { title: string; sections: QuestionPaperSection[] }) => {
  const paper = await generate(input)
  if (paper) {
    activeTab.value = 'papers'
    await openPaper(paper.id)
  }
}

const openPaperAndSwitch = async (id: number) => {
  activeTab.value = 'papers'
  await openPaper(id)
}

if (selectedUserId.value) {
  setUser(selectedUserId.value)
  refreshAll()
  loadPapers()
}
</script>
