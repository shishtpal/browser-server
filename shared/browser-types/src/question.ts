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

export type ReviewRating = 'again' | 'hard' | 'good' | 'easy'
export type QuestionCardStatus = 'due' | 'new' | 'scheduled'

export interface QuestionReviewState {
  question_id: number
  repetitions: number
  interval_seconds: number
  ease_factor: number
  due_at: string
  last_rating: ReviewRating
  last_reviewed_at: string
  skip_count: number
  /** Null when the card has never been skipped. */
  last_skipped_at: string | null
}

export interface QuestionCardItem {
  question: QuestionResponse
  review: QuestionReviewState | null
  status: QuestionCardStatus
}

export interface QuestionCardQueue {
  items: QuestionCardItem[]
  due_count: number
  new_count: number
  available_count: number
}

export interface ListQuestionCardsOptions {
  /** Tags use ANY-match semantics. Omit to explicitly include all questions. */
  tags?: string[]
  limit?: number
  practice?: boolean
  /** Practice mode: mix in a specific bucket. "new" shows only unrated cards;
   *  "skipped" returns ones the user punted at least once; "hard" filters by
   *  question difficulty. Omitting mode keeps the due-then-new default. */
  mode?: CardFilterMode
}

export type CardFilterMode = 'new' | 'skipped' | 'hard'

export interface ReviewQuestionInput {
  user_id: number
  rating: ReviewRating
}

export interface SkipQuestionCardInput {
  user_id: number
}
