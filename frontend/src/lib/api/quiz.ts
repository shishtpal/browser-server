import type {
  CreateQuestionInput,
  GeneratePaperInput,
  ListQuestionsOptions,
  QuestionPaper,
  QuestionResponse,
  QuizStats,
  TagVocabulary,
  UpdateQuestionInput,
} from '@browser-server/shared-types'
import { API_BASE, authHeaders } from './client'

function request<T>(path: string, init?: RequestInit): Promise<T> {
  return fetch(`${API_BASE}${path}`, { ...init, headers: { ...authHeaders(), ...(init?.headers ?? {}) } }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
    return res.json() as Promise<T>
  })
}

export function getQuestions(userId: number, options: ListQuestionsOptions = {}): Promise<QuestionResponse[]> {
  const params = new URLSearchParams({ user_id: String(userId) })
  for (const [key, value] of Object.entries(options)) {
    if (value !== undefined && value !== '') params.set(key, String(value))
  }
  return request<QuestionResponse[]>(`/api/quiz/questions?${params.toString()}`)
}

export function getQuestion(id: number): Promise<QuestionResponse> {
  return request<QuestionResponse>(`/api/quiz/questions/${id}`)
}

export function createQuestion(data: CreateQuestionInput): Promise<QuestionResponse> {
  return request<QuestionResponse>('/api/quiz/questions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export function updateQuestion(id: number, data: UpdateQuestionInput): Promise<QuestionResponse> {
  return request<QuestionResponse>(`/api/quiz/questions/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export function deleteQuestion(id: number): Promise<void> {
  return fetch(`${API_BASE}/api/quiz/questions/${id}`, { method: 'DELETE', headers: authHeaders() }).then((res) => {
    if (res.status === 204) return
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  })
}

export async function uploadQuestionImage(id: number, file: File): Promise<{ id: number; filename: string; image_url: string }> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch(`${API_BASE}/api/quiz/questions/${id}/image`, { method: 'POST', headers: authHeaders(), body: form })
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  return res.json()
}

export function getQuestionImageUrl(id: number): string {
  return `${API_BASE}/api/quiz/questions/${id}/image`
}

export function generatePaper(data: GeneratePaperInput): Promise<QuestionPaper> {
  return request<QuestionPaper>('/api/quiz/papers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export function getPapers(userId: number, limit = 100, offset = 0): Promise<QuestionPaper[]> {
  const params = new URLSearchParams({ user_id: String(userId), limit: String(limit), offset: String(offset) })
  return request<QuestionPaper[]>(`/api/quiz/papers?${params.toString()}`)
}

export function getPaper(id: number): Promise<QuestionPaper> {
  return request<QuestionPaper>(`/api/quiz/papers/${id}`)
}

export function deletePaper(id: number): Promise<void> {
  return fetch(`${API_BASE}/api/quiz/papers/${id}`, { method: 'DELETE', headers: authHeaders() }).then((res) => {
    if (res.status === 204) return
    if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  })
}

export function getTagVocabulary(userId: number): Promise<TagVocabulary> {
  return request<TagVocabulary>(`/api/quiz/tags?user_id=${userId}`)
}

export function getQuizStats(userId: number): Promise<QuizStats> {
  return request<QuizStats>(`/api/quiz/stats?user_id=${userId}`)
}
