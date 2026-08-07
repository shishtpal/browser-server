export interface QuestionOption {
  index: number
  text: string
  correct?: boolean
}

export interface ChronologyItem {
  index: number
  text: string
  correct_order: number
}

export type QuestionType = 'single_choice' | 'multiple_choice' | 'input' | 'chronology'
export type QuestionDifficulty = 'easy' | 'medium' | 'hard'

export interface Question {
  id: number
  user_id: number
  type: QuestionType
  difficulty: QuestionDifficulty
  question: string
  explanation: string
  image_filename: string
  subject: string
  topic: string
  sub_topic: string
  source: string
  created_at: string
  updated_at: string
}

export interface QuestionResponse extends Question {
  options?: QuestionOption[]
  correct_answers?: number[]
  expected_text?: string
  chronology_items?: ChronologyItem[]
  /** Decoded tag list — the canonical wire shape for tags. */
  tags?: string[]
  image_url?: string
}

export interface CreateQuestionInput {
  user_id: number
  type: QuestionType
  difficulty?: QuestionDifficulty
  question: string
  explanation?: string
  options?: QuestionOption[]
  chronology_items?: ChronologyItem[]
  expected_text?: string
  tags?: string[]
  subject?: string
  topic?: string
  sub_topic?: string
  source?: string
}

export interface UpdateQuestionInput {
  type?: QuestionType
  difficulty?: QuestionDifficulty
  question?: string
  explanation?: string
  options?: QuestionOption[]
  chronology_items?: ChronologyItem[]
  expected_text?: string
  tags?: string[]
  subject?: string
  topic?: string
  sub_topic?: string
  source?: string
}

export interface QuestionPaperSection {
  tags?: string[]
  subject?: string
  topic?: string
  sub_topic?: string
  type?: QuestionType
  difficulty?: QuestionDifficulty
  count: number
}

export interface QuestionPaper {
  id: number
  user_id: number
  title: string
  sections: QuestionPaperSection[]
  question_count: number
  questions?: QuestionResponse[]
  created_at: string
}

export interface GeneratePaperInput {
  user_id: number
  title: string
  sections: QuestionPaperSection[]
}

export interface TagVocabulary {
  tags: string[]
  subjects: string[]
  topics: string[]
  sub_topics: string[]
}

export interface QuizStats {
  total: number
  paper_count: number
  by_type: Record<string, number>
  by_difficulty: Record<string, number>
  by_tags: Record<string, number>
}

export interface ListQuestionsOptions {
  type?: QuestionType
  difficulty?: QuestionDifficulty
  /** One or more tags. A question matches if its tags array contains ANY of these. */
  tags?: string[]
  subject?: string
  topic?: string
  sub_topic?: string
  q?: string
  limit?: number
  offset?: number
}
