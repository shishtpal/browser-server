<template>
  <Modal
    :open="open"
    :title="question ? 'Edit question' : 'Add question'"
    :description="question ? 'Update the question, its answers, and tags.' : 'Create a new question for the question bank.'"
    fullscreen
    @close="$emit('close')"
  >
    <div class="h-full overflow-y-auto overscroll-contain">
      <div class="mx-auto w-full max-w-5xl px-3 pb-6 pt-1 sm:px-6 lg:px-8">
        <QuestionForm
          :question="question"
          :vocabulary="vocabulary"
          :is-saving="isSaving"
          @save="onSave"
          @cancel="$emit('close')"
        />
      </div>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import type { QuestionResponse, TagVocabulary } from '../../types'
import Modal from '../ui/Modal.vue'
import QuestionForm from './QuestionForm.vue'

const props = defineProps<{
  open: boolean
  question?: QuestionResponse | null
  vocabulary?: TagVocabulary | null
  isSaving?: boolean
}>()

const emit = defineEmits<{
  close: []
  save: [id: number | null, payload: Record<string, unknown>, image: File | null]
}>()

function onSave(payload: Record<string, unknown>, image: File | null) {
  emit('save', props.question?.id ?? null, payload, image)
}
</script>
