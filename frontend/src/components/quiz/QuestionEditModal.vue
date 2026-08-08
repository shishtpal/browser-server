<template>
  <Modal
    :open="!!question"
    title="Edit question"
    description="Update the question, its answers, and tags."
    @close="$emit('close')"
  >
    <div v-if="question" class="max-h-[70vh] overflow-y-auto pr-1">
      <QuestionForm
        :question="question"
        :vocabulary="vocabulary"
        :is-saving="isSaving"
        @save="onSave"
        @cancel="$emit('close')"
      />
    </div>
  </Modal>
</template>

<script setup lang="ts">
import type { QuestionResponse, TagVocabulary } from "../../types";
import Modal from "../ui/Modal.vue";
import QuestionForm from "./QuestionForm.vue";

const props = defineProps<{
  question: QuestionResponse | null;
  vocabulary?: TagVocabulary | null;
  isSaving?: boolean;
}>();

const emit = defineEmits<{
  close: [];
  save: [id: number, payload: Record<string, unknown>, image: File | null];
}>();

function onSave(payload: Record<string, unknown>, image: File | null) {
  if (!props.question) return;
  emit("save", props.question.id, payload, image);
}
</script>
