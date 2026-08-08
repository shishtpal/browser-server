import type {
  CreateQuestionInput,
  GeneratePaperInput,
  ListQuestionCardsOptions,
  ListQuestionsOptions,
  QuestionCardQueue,
  QuestionPaper,
  QuestionResponse,
  QuestionReviewState,
  QuizStats,
  ReviewQuestionInput,
  TagVocabulary,
  UpdateQuestionInput,
} from '@browser-server/shared-types';
import { client } from './client';

export const getQuestions = (
  userId: number,
  options: ListQuestionsOptions = {},
): Promise<QuestionResponse[]> => client.listQuestions(userId, options);

export const getQuestion = (id: number): Promise<QuestionResponse> => client.getQuestion(id);

export const createQuestion = (data: CreateQuestionInput): Promise<QuestionResponse> =>
  client.createQuestion(data);

export const updateQuestion = (id: number, data: UpdateQuestionInput): Promise<QuestionResponse> =>
  client.updateQuestion(id, data);

export const deleteQuestion = (id: number): Promise<void> => client.deleteQuestion(id);

export const uploadQuestionImage = (
  id: number,
  file: File,
): Promise<{ id: number; filename: string; image_url: string }> =>
  client.uploadQuestionImage(id, file);

export const getQuestionImageUrl = (id: number): string => client.getQuestionImageUrl(id);

export const generatePaper = (data: GeneratePaperInput): Promise<QuestionPaper> =>
  client.generatePaper(data);

export const getPapers = (userId: number, limit = 100, offset = 0): Promise<QuestionPaper[]> =>
  client.listPapers(userId, limit, offset);

export const getPaper = (id: number): Promise<QuestionPaper> => client.getPaper(id);

export const deletePaper = (id: number): Promise<void> => client.deletePaper(id);

export const getTagVocabulary = (userId: number): Promise<TagVocabulary> =>
  client.getTagVocabulary(userId);

export const getQuizStats = (userId: number): Promise<QuizStats> => client.getQuizStats(userId);

export const getQuestionCards = (
  userId: number,
  options: ListQuestionCardsOptions = {},
): Promise<QuestionCardQueue> => client.listQuestionCards(userId, options);

export const reviewQuestion = (
  id: number,
  input: ReviewQuestionInput,
): Promise<QuestionReviewState> => client.reviewQuestion(id, input);
