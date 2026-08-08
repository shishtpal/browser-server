<template>
  <Modal
    :open="open"
    :title="paper?.title ?? 'Online Exam'"
    :description="modalDescription"
    fullscreen
    @close="handleClose"
  >
    <div v-if="paper" class="flex h-full min-h-0 flex-col bg-slate-50 dark:bg-slate-950">
      <!-- Exam in progress -->
      <template v-if="!isExamSubmitted">
        <ExamTopBar
          :elapsed-time="elapsedTime"
          :current-index="currentQuestionIndex"
          :total="totalQuestions"
          :palette-open="isPaletteDrawerOpen"
          @toggle-palette="isPaletteDrawerOpen = !isPaletteDrawerOpen"
          @submit="showSubmitConfirm = true"
        />

        <div class="flex min-h-0 flex-1 overflow-hidden">
          <div class="min-w-0 flex-1 overflow-y-auto overscroll-contain p-3 sm:p-6 lg:p-8">
            <ExamQuestionCard
              v-if="currentQuestion"
              :question="currentQuestion"
              :question-number="currentQuestionIndex + 1"
              :flagged="isFlagged(currentQuestion.id)"
              :answer="answers[currentQuestion.id]"
              :is-first="currentQuestionIndex === 0"
              :is-last="currentQuestionIndex === totalQuestions - 1"
              @toggle-flag="toggleFlag(currentQuestion.id)"
              @single-choice="setSingleChoice(currentQuestion.id, $event)"
              @multiple-choice="toggleMultipleChoice(currentQuestion.id, $event)"
              @input-text="setInputText(currentQuestion.id, $event)"
              @chronology-move="
                (from, to) => currentQuestion && moveChronologyItem(currentQuestion.id, from, to)
              "
              @prev="currentQuestionIndex--"
              @next="currentQuestionIndex++"
            />
          </div>

          <ExamPalette
            :questions="paper.questions || []"
            :current-index="currentQuestionIndex"
            :open="isPaletteDrawerOpen"
            :answered-count="answeredCount"
            :flagged-count="flaggedCount"
            :total="totalQuestions"
            :is-answered="isQuestionAnswered"
            :is-flagged="isFlagged"
            @jump="jumpToQuestion"
            @close="isPaletteDrawerOpen = false"
          />
        </div>
      </template>

      <!-- Submitted: score & review -->
      <template v-else-if="attemptRecord">
        <div class="flex-1 overflow-y-auto overscroll-contain p-3 sm:p-6 lg:p-8">
          <div class="mx-auto max-w-4xl space-y-5 sm:space-y-6">
            <ExamScoreSummary :record="attemptRecord" :title="paper.title" />

            <ExamReviewList
              :paper="paper"
              :record="attemptRecord"
              :answers="answers"
              :is-flagged="isFlagged"
            />

            <div class="flex flex-col gap-2 pt-2 sm:flex-row sm:justify-end sm:gap-3">
              <Button variant="secondary" size="sm" class="w-full sm:w-auto" @click="retakeExam">
                <span class="inline-flex items-center gap-1.5">
                  <RotateCcw class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
                  Retake exam
                </span>
              </Button>
              <Button
                variant="gradient-violet"
                size="sm"
                class="w-full sm:w-auto"
                @click="handleClose"
              >
                <span class="inline-flex items-center gap-1.5">
                  <Check class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
                  Done
                </span>
              </Button>
            </div>
          </div>
        </div>
      </template>
    </div>

    <ExamSubmitConfirm
      :open="showSubmitConfirm"
      :answered="answeredCount"
      :total="totalQuestions"
      @confirm="confirmSubmit"
      @cancel="showSubmitConfirm = false"
    />
  </Modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { Check, RotateCcw } from '@lucide/vue';
import type { QuestionPaper, QuestionResponse } from '../../../types';
import { usePaperAttempt } from '../composables/usePaperAttempt';
import Button from '../../ui/Button.vue';
import Modal from '../../ui/Modal.vue';
import ExamTopBar from './runner/ExamTopBar.vue';
import ExamQuestionCard from './runner/ExamQuestionCard.vue';
import ExamPalette from './runner/ExamPalette.vue';
import ExamSubmitConfirm from './runner/ExamSubmitConfirm.vue';
import ExamScoreSummary from './runner/ExamScoreSummary.vue';
import ExamReviewList from './runner/ExamReviewList.vue';

const props = defineProps<{
  open: boolean;
  paper: QuestionPaper | null;
}>();

const emit = defineEmits<{
  close: [];
}>();

const {
  isExamSubmitted,
  currentQuestionIndex,
  answers,
  flagged,
  elapsedTime,
  attemptRecord,
  startExam,
  closeExam,
  submitExam,
  setSingleChoice,
  toggleMultipleChoice,
  setInputText,
  moveChronologyItem,
  toggleFlag,
  isQuestionAnswered,
  answeredCount,
  flaggedCount,
} = usePaperAttempt();

const showSubmitConfirm = ref(false);
const isPaletteDrawerOpen = ref(false);

watch(
  () => [props.open, props.paper?.id] as const,
  ([isOpen]) => {
    if (isOpen && props.paper?.questions?.length) {
      startExam(props.paper);
      isPaletteDrawerOpen.value = false;
      showSubmitConfirm.value = false;
    }
  },
  { immediate: true },
);

const totalQuestions = computed(() => props.paper?.questions?.length || 0);

const currentQuestion = computed<QuestionResponse | null>(() => {
  if (!props.paper?.questions?.length) return null;
  return props.paper.questions[currentQuestionIndex.value] || null;
});

const modalDescription = computed(() => {
  if (isExamSubmitted.value) return 'Performance summary and answer key review.';
  return `${totalQuestions.value} questions · Online Exam Mode`;
});

const isFlagged = (qId: number) => Boolean(flagged.value[qId]);

const jumpToQuestion = (idx: number) => {
  currentQuestionIndex.value = idx;
  isPaletteDrawerOpen.value = false;
};

const confirmSubmit = () => {
  showSubmitConfirm.value = false;
  submitExam();
};

const retakeExam = () => {
  if (props.paper) startExam(props.paper);
};

const handleClose = () => {
  closeExam();
  emit('close');
};
</script>
