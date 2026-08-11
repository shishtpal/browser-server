import { ref, type Ref } from 'vue';
import type { QuestionPaper, QuestionPaperSection, QuestionResponse } from '../../../types';
import { useQuestions } from './useQuestions';
import { useQuizPapers } from './useQuizPapers';

export type QuizTab = 'dashboard' | 'questions' | 'cards' | 'generate' | 'papers';

/** Optional hooks for cross-tab coordination (e.g. syncing the Cards queue
 *  when a question is edited from the Questions tab modal). */
export interface UseQuizPageOptions {
  /** Fired after a question is successfully edited via the modal. */
  onQuestionEdited?: (question: QuestionResponse) => void;
}

/**
 * Orchestrates the whole Quiz page: the active tab, the question editor
 * modal, the exam runner modal and the post-generation navigation flow.
 * Domain state lives in useQuestions / useQuizPapers; this composable only
 * owns cross-tab UI coordination so QuizPage.vue stays pure wiring.
 */
export function useQuizPage(
  userId: Ref<number | null>,
  options?: UseQuizPageOptions,
) {
  const questions = useQuestions(userId);
  const papers = useQuizPapers(userId);

  const activeTab = ref<QuizTab>('dashboard');

  /* --------------------------- question modal ---------------------------- */

  const isQuestionModalOpen = ref(false);
  const editingQuestion = ref<QuestionResponse | null>(null);
  const isSavingQuestion = ref(false);

  const openAddQuestion = () => {
    editingQuestion.value = null;
    isQuestionModalOpen.value = true;
  };

  const openEditQuestion = (question: QuestionResponse) => {
    editingQuestion.value = question;
    isQuestionModalOpen.value = true;
  };

  const closeQuestionModal = () => {
    isQuestionModalOpen.value = false;
    editingQuestion.value = null;
  };

  const saveQuestion = async (
    id: number | null,
    payload: Record<string, unknown>,
    image: File | null,
  ) => {
    isSavingQuestion.value = true;
    try {
      const saved = id
        ? await questions.editQuestion(id, payload as never, image)
        : await questions.addQuestion(payload as never, image);
      if (saved) {
        if (id) options?.onQuestionEdited?.(saved);
        closeQuestionModal();
      }
    } finally {
      isSavingQuestion.value = false;
    }
  };

  /* ----------------------------- exam runner ----------------------------- */

  const runnerPaper = ref<QuestionPaper | null>(null);
  const isRunnerOpen = ref(false);

  const attemptPaper = async (paper: QuestionPaper) => {
    try {
      const fullPaper = await papers.fetchPaper(paper.id);
      runnerPaper.value = fullPaper;
      isRunnerOpen.value = true;
    } catch (e) {
      console.error('Failed to load paper details for attempt', e);
      papers.error.value = e instanceof Error ? e.message : 'Failed to load paper';
    }
  };

  const closeRunner = () => {
    isRunnerOpen.value = false;
    runnerPaper.value = null;
  };

  /* --------------------------- navigation flows --------------------------- */

  const generatePaperFlow = async (input: {
    title: string;
    sections: QuestionPaperSection[];
    autoAttempt?: boolean;
  }) => {
    const paper = await papers.generate(input);
    if (!paper) return;

    if (input.autoAttempt) {
      await attemptPaper(paper);
    } else {
      activeTab.value = 'papers';
      await papers.openPaper(paper.id);
    }
  };

  /** Dashboard "recent papers" → jump to Papers tab and open the viewer. */
  const openPaperFromDashboard = async (id: number) => {
    activeTab.value = 'papers';
    await papers.openPaper(id);
  };

  return {
    questions,
    papers,
    activeTab,
    // question modal
    isQuestionModalOpen,
    editingQuestion,
    isSavingQuestion,
    openAddQuestion,
    openEditQuestion,
    closeQuestionModal,
    saveQuestion,
    // exam runner
    runnerPaper,
    isRunnerOpen,
    attemptPaper,
    closeRunner,
    // flows
    generatePaperFlow,
    openPaperFromDashboard,
  };
}
