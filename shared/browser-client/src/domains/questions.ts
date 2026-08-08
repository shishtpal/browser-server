import type {
  GeneratePaperInput,
  ListQuestionsOptions,
  QuestionPaper,
  QuestionResponse,
  CreateQuestionInput,
  UpdateQuestionInput,
  QuizStats,
  TagVocabulary,
  ListQuestionCardsOptions,
  QuestionCardQueue,
  QuestionReviewState,
  ReviewQuestionInput,
} from '@browser-server/shared-types'
import { type TokenProvider, apiFetch, buildQuery } from '../internals'

export function createQuestionMethods(baseUrl: string, getToken?: TokenProvider) {
  return {
    listQuestions(userId: number, options: ListQuestionsOptions = {}): Promise<QuestionResponse[]> {
      // Multiple tag values become repeated ?tag=... query params; the
      // server treats that as "match any of these tags".
      const qs = buildQuery({
        user_id: userId,
        type: options.type,
        difficulty: options.difficulty,
        tag: options.tags,
        subject: options.subject,
        topic: options.topic,
        sub_topic: options.sub_topic,
        q: options.q,
        limit: options.limit,
        offset: options.offset,
      })
      return apiFetch<QuestionResponse[]>(baseUrl, 'GET', `/api/quiz/questions${qs}`, undefined, getToken)
    },

    getQuestion(id: number): Promise<QuestionResponse> {
      return apiFetch<QuestionResponse>(baseUrl, 'GET', `/api/quiz/questions/${id}`, undefined, getToken)
    },

    createQuestion(data: CreateQuestionInput): Promise<QuestionResponse> {
      return apiFetch<QuestionResponse>(baseUrl, 'POST', '/api/quiz/questions', data, getToken)
    },

    updateQuestion(id: number, data: UpdateQuestionInput): Promise<QuestionResponse> {
      return apiFetch<QuestionResponse>(baseUrl, 'PUT', `/api/quiz/questions/${id}`, data, getToken)
    },

    deleteQuestion(id: number): Promise<void> {
      return apiFetch<void>(baseUrl, 'DELETE', `/api/quiz/questions/${id}`, undefined, getToken)
    },

    async uploadQuestionImage(id: number, file: File | Blob): Promise<{ id: number; filename: string; image_url: string }> {
      const form = new FormData()
      form.append('file', file)
      const headers: Record<string, string> = {}
      const token = getToken?.()
      if (token) headers.Authorization = `Bearer ${token}`
      const response = await fetch(`${baseUrl}/api/quiz/questions/${id}/image`, { method: 'POST', headers, body: form })
      if (!response.ok) {
        const text = await response.text()
        throw new Error(text || `Request failed: ${response.status}`)
      }
      return response.json()
    },

    getQuestionImageUrl(id: number): string {
      return `${baseUrl}/api/quiz/questions/${id}/image`
    },

    generatePaper(data: GeneratePaperInput): Promise<QuestionPaper> {
      return apiFetch<QuestionPaper>(baseUrl, 'POST', '/api/quiz/papers', data, getToken)
    },

    listPapers(userId: number, limit?: number, offset?: number): Promise<QuestionPaper[]> {
      const qs = buildQuery({ user_id: userId, limit, offset })
      return apiFetch<QuestionPaper[]>(baseUrl, 'GET', `/api/quiz/papers${qs}`, undefined, getToken)
    },

    getPaper(id: number): Promise<QuestionPaper> {
      return apiFetch<QuestionPaper>(baseUrl, 'GET', `/api/quiz/papers/${id}`, undefined, getToken)
    },

    deletePaper(id: number): Promise<void> {
      return apiFetch<void>(baseUrl, 'DELETE', `/api/quiz/papers/${id}`, undefined, getToken)
    },

    getTagVocabulary(userId: number): Promise<TagVocabulary> {
      const qs = buildQuery({ user_id: userId })
      return apiFetch<TagVocabulary>(baseUrl, 'GET', `/api/quiz/tags${qs}`, undefined, getToken)
    },

    getQuizStats(userId: number): Promise<QuizStats> {
      const qs = buildQuery({ user_id: userId })
      return apiFetch<QuizStats>(baseUrl, 'GET', `/api/quiz/stats${qs}`, undefined, getToken)
    },

    listQuestionCards(userId: number, options: ListQuestionCardsOptions = {}): Promise<QuestionCardQueue> {
      const qs = buildQuery({ user_id: userId, tag: options.tags, limit: options.limit, practice: options.practice ? 'true' : undefined })
      return apiFetch<QuestionCardQueue>(baseUrl, 'GET', `/api/quiz/cards${qs}`, undefined, getToken)
    },

    reviewQuestion(questionId: number, input: ReviewQuestionInput): Promise<QuestionReviewState> {
      return apiFetch<QuestionReviewState>(baseUrl, 'POST', `/api/quiz/cards/${questionId}/review`, input, getToken)
    },
  }
}
