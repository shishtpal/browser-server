import { ref, computed, watch, type Ref } from 'vue'
import {
  getQuestions,
  createQuestion,
  updateQuestion,
  deleteQuestion,
  uploadQuestionImage,
  getQuizStats,
  getTagVocabulary,
} from '../lib/api'
import type {
  CreateQuestionInput,
  ListQuestionsOptions,
  QuestionResponse,
  QuizStats,
  TagVocabulary,
  UpdateQuestionInput,
} from '../types'

export function useQuestions(userId: Ref<number | null>) {
  const questions = ref<QuestionResponse[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const stats = ref<QuizStats | null>(null)
  const vocabulary = ref<TagVocabulary | null>(null)

  // Filters
  const filterType = ref<string>('')
  const filterDifficulty = ref<string>('')
  const filterTags = ref<string[]>([])
  const filterSubject = ref<string>('')
  const searchQuery = ref<string>('')

  const filters = computed<ListQuestionsOptions>(() => ({
    type: (filterType.value || undefined) as ListQuestionsOptions['type'],
    difficulty: (filterDifficulty.value || undefined) as ListQuestionsOptions['difficulty'],
    tags: filterTags.value.length ? filterTags.value : undefined,
    subject: filterSubject.value || undefined,
    q: searchQuery.value.trim() || undefined,
    limit: 500,
  }))

  const loadQuestions = async () => {
    if (!userId.value) return
    isLoading.value = true
    error.value = null
    try {
      questions.value = await getQuestions(userId.value, filters.value)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load questions'
    } finally {
      isLoading.value = false
    }
  }

  const loadStats = async () => {
    if (!userId.value) return
    try {
      stats.value = await getQuizStats(userId.value)
    } catch {
      stats.value = null
    }
  }

  const loadVocabulary = async () => {
    if (!userId.value) return
    try {
      vocabulary.value = await getTagVocabulary(userId.value)
    } catch {
      vocabulary.value = null
    }
  }

  const refreshAll = async () => {
    await Promise.all([loadQuestions(), loadStats(), loadVocabulary()])
  }

  const addQuestion = async (data: CreateQuestionInput, image?: File | null) => {
    if (!userId.value) return undefined
    try {
      const created = await createQuestion({ ...data, user_id: userId.value })
      if (image) {
        await uploadQuestionImage(created.id, image)
        created.image_url = `/api/quiz/questions/${created.id}/image`
        created.image_filename = 'uploaded'
      }
      questions.value.unshift(created)
      loadStats()
      loadVocabulary()
      return created
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create question'
      return undefined
    }
  }

  const editQuestion = async (id: number, data: UpdateQuestionInput, image?: File | null) => {
    try {
      const resp = await updateQuestion(id, data)
      if (image) {
        await uploadQuestionImage(id, image)
        resp.image_url = `/api/quiz/questions/${id}/image`
      }
      const idx = questions.value.findIndex((q) => q.id === id)
      if (idx >= 0) questions.value[idx] = resp
      loadStats()
      loadVocabulary()
      return resp
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update question'
      return undefined
    }
  }

  const removeQuestion = async (id: number) => {
    try {
      await deleteQuestion(id)
      questions.value = questions.value.filter((q) => q.id !== id)
      loadStats()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete question'
    }
  }

  watch(userId, (val) => {
    if (val && val > 0) {
      refreshAll()
    } else {
      questions.value = []
      stats.value = null
      vocabulary.value = null
    }
  })

  return {
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
    refreshAll,
    addQuestion,
    editQuestion,
    removeQuestion,
  }
}
