<template>
  <form class="space-y-5 sm:space-y-6" @submit.prevent="submit">
    <!-- Basics -->
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <FormField label="Type">
        <SelectField v-model="form.type" :disabled="isEdit">
          <option value="single_choice">Single choice</option>
          <option value="multiple_choice">Multiple choice</option>
          <option value="input">Input (text answer)</option>
          <option value="chronology">Chronology (ordering)</option>
        </SelectField>
      </FormField>

      <FormField label="Difficulty">
        <SelectField v-model="form.difficulty">
          <option value="easy">Easy</option>
          <option value="medium">Medium</option>
          <option value="hard">Hard</option>
        </SelectField>
      </FormField>

      <FormField label="Source" class="sm:col-span-2 lg:col-span-1">
        <InputField v-model="form.source" placeholder="e.g. SSC CGL 2023" />
      </FormField>
    </div>

    <FormField label="Question" required>
      <TextAreaField
        v-model="form.question"
        placeholder="Enter the question text"
        required
        :rows="4"
      />
    </FormField>

    <!-- Answer editor, one per question type -->
    <OptionsEditor
      v-if="isChoice"
      :options="form.options"
      :single="form.type === 'single_choice'"
    />

    <FormField v-else-if="form.type === 'input'" label="Expected answer" required>
      <InputField v-model="form.expectedText" placeholder="The correct answer" required />
    </FormField>

    <ChronologyEditor v-else-if="form.type === 'chronology'" :items="form.chronologyItems" />

    <FormField label="Explanation (markdown)">
      <TextAreaField
        v-model="form.explanation"
        placeholder="Optional explanation shown after answering"
        :rows="3"
      />
    </FormField>

    <!-- Tags & taxonomy -->
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <FormField
        label="Tags"
        help-text="e.g. SSC, RRB. Press enter to add."
        class="sm:col-span-2 lg:col-span-1"
      >
        <TagInput
          ref="tagInput"
          v-model="form.tags"
          :suggestions="vocabulary?.tags ?? []"
          list-id="quiz-form-tags"
          placeholder="Type tag…"
        />
      </FormField>

      <FormField v-for="field in taxonomyFields" :key="field.key" :label="field.label">
        <InputField
          v-model="form[field.key]"
          :placeholder="field.placeholder"
          :list="`quiz-form-${field.key}`"
        />
        <datalist :id="`quiz-form-${field.key}`">
          <option v-for="v in vocabulary?.[field.vocabKey] ?? []" :key="v" :value="v" />
        </datalist>
      </FormField>
    </div>

    <!-- Image -->
    <FormField label="Image (optional)">
      <div class="space-y-3">
        <label
          class="flex cursor-pointer items-center justify-center gap-2 rounded-xl border border-dashed border-gray-300 bg-gray-50/60 px-4 py-5 text-xs font-semibold text-slate-500 transition hover:border-violet-400 hover:bg-violet-50/40 hover:text-violet-600 dark:border-slate-600 dark:bg-slate-800/40 dark:text-slate-400 dark:hover:border-violet-600 dark:hover:text-violet-400"
        >
          <ImagePlus class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
          {{ imageFile || imagePreview ? 'Replace image' : 'Choose an image to attach' }}
          <input ref="fileInput" type="file" accept="image/*" class="sr-only" @change="onFile" />
        </label>

        <p
          v-if="isEdit && question?.image_url && !imagePreview"
          class="text-[11px] text-slate-500 dark:text-slate-400"
        >
          This question already has an image attached — choosing a new one replaces it.
        </p>

        <div v-if="imagePreview || (isEdit && existingImageSrc)" class="relative inline-block">
          <img
            :src="imagePreview ?? existingImageSrc ?? undefined"
            alt="Selected image preview"
            class="max-h-40 rounded-xl border border-gray-200 object-contain shadow-sm dark:border-slate-700"
          />
          <button
            v-if="imagePreview"
            type="button"
            class="absolute -top-2 -right-2 flex h-7 w-7 items-center justify-center rounded-full bg-rose-500 text-white shadow transition hover:bg-rose-600"
            aria-label="Remove selected image"
            @click="clearImage"
          >
            <X class="h-3.5 w-3.5" :stroke-width="3" aria-hidden="true" />
          </button>
        </div>
      </div>
    </FormField>

    <!-- Validation error -->
    <div
      v-if="formError"
      class="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm font-semibold text-rose-700 dark:border-rose-800/40 dark:bg-rose-900/20 dark:text-rose-300"
      role="alert"
    >
      <CircleAlert class="mt-0.5 h-4 w-4 shrink-0" :stroke-width="2.25" aria-hidden="true" />
      {{ formError }}
    </div>

    <!-- Sticky action bar -->
    <div
      class="sticky bottom-0 z-10 -mx-3 flex flex-col-reverse gap-2 border-t border-gray-200 bg-white/95 px-3 py-3 backdrop-blur sm:-mx-6 sm:flex-row sm:justify-end sm:gap-3 sm:px-6 lg:-mx-8 lg:px-8 dark:border-slate-700 dark:bg-slate-900/95"
    >
      <Button v-if="isEdit" variant="ghost" class="w-full sm:w-auto" @click="$emit('cancel')">
        Cancel
      </Button>
      <Button
        type="submit"
        variant="gradient-violet"
        class="w-full sm:w-auto"
        :loading="isSaving"
        :loading-text="isEdit ? 'Saving…' : 'Adding…'"
      >
        <span class="inline-flex items-center gap-1.5">
          <Save class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          {{ isEdit ? 'Save changes' : 'Add question' }}
        </span>
      </Button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { CircleAlert, ImagePlus, Save, X } from '@lucide/vue';
import type {
  QuestionOption,
  QuestionResponse,
  QuestionType,
  TagVocabulary,
} from '../../../../types';
import { API_BASE } from '../../../../lib/api';
import FormField from '../../../ui/FormField.vue';
import InputField from '../../../ui/InputField.vue';
import SelectField from '../../../ui/SelectField.vue';
import TextAreaField from '../../../ui/TextAreaField.vue';
import Button from '../../../ui/Button.vue';
import TagInput from '../../ui/TagInput.vue';
import OptionsEditor from './OptionsEditor.vue';
import ChronologyEditor, { type ChronologyDraft } from './ChronologyEditor.vue';
import { questionImageSrc } from '../../quizFormat';

const props = defineProps<{
  question?: QuestionResponse | null;
  vocabulary?: TagVocabulary | null;
  isSaving?: boolean;
}>();

const emit = defineEmits<{
  save: [payload: Record<string, unknown>, image: File | null];
  cancel: [];
}>();

/* ------------------------------- state --------------------------------- */

const isEdit = computed(() => !!props.question);

interface FormState {
  type: QuestionType;
  difficulty: 'easy' | 'medium' | 'hard';
  question: string;
  explanation: string;
  source: string;
  tags: string[];
  subject: string;
  topic: string;
  sub_topic: string;
  options: QuestionOption[];
  chronologyItems: ChronologyDraft[];
  expectedText: string;
}

const blankOptions = (): QuestionOption[] => [
  { index: 0, text: '', correct: false },
  { index: 1, text: '', correct: false },
];
const blankChronology = (): ChronologyDraft[] => [
  { index: 0, text: '', correct_order: 1 },
  { index: 1, text: '', correct_order: 2 },
];

const form = reactive<FormState>({
  type: 'single_choice',
  difficulty: 'medium',
  question: '',
  explanation: '',
  source: '',
  tags: [],
  subject: '',
  topic: '',
  sub_topic: '',
  options: blankOptions(),
  chronologyItems: blankChronology(),
  expectedText: '',
});

const imageFile = ref<File | null>(null);
const imagePreview = ref<string | null>(null);
const fileInput = ref<HTMLInputElement | null>(null);
const tagInput = ref<{ flush: () => void } | null>(null);
const formError = ref<string | null>(null);

const isChoice = computed(() => form.type === 'single_choice' || form.type === 'multiple_choice');

const taxonomyFields = [
  { key: 'subject', label: 'Subject', placeholder: 'Math', vocabKey: 'subjects' },
  { key: 'topic', label: 'Topic', placeholder: 'Algebra', vocabKey: 'topics' },
  { key: 'sub_topic', label: 'Sub-topic', placeholder: 'Equations', vocabKey: 'sub_topics' },
] as const;

const existingImageSrc = computed(() =>
  props.question?.image_url ? questionImageSrc(props.question.image_url, API_BASE) : undefined,
);

/* ---------------------------- edit prefill ------------------------------ */

watch(
  () => props.question,
  (q) => {
    if (!q) return;
    form.type = q.type;
    form.difficulty = q.difficulty;
    form.question = q.question;
    form.explanation = q.explanation;
    form.source = q.source;
    form.tags = Array.isArray(q.tags) ? [...q.tags] : [];
    form.subject = q.subject;
    form.topic = q.topic;
    form.sub_topic = q.sub_topic;
    form.options = q.options?.length ? q.options.map((o) => ({ ...o })) : blankOptions();
    form.chronologyItems = q.chronology_items?.length
      ? q.chronology_items.map((c) => ({ ...c }))
      : blankChronology();
    form.expectedText = q.expected_text ?? '';
  },
  { immediate: true },
);

/* -------------------------------- image --------------------------------- */

const revokePreview = () => {
  if (imagePreview.value) {
    URL.revokeObjectURL(imagePreview.value);
    imagePreview.value = null;
  }
};

const onFile = (e: Event) => {
  const target = e.target as HTMLInputElement;
  revokePreview();
  imageFile.value = target.files?.[0] ?? null;
  if (imageFile.value) {
    imagePreview.value = URL.createObjectURL(imageFile.value);
  }
};

const clearImage = () => {
  revokePreview();
  imageFile.value = null;
  if (fileInput.value) fileInput.value.value = '';
};

onBeforeUnmount(revokePreview);

/* ------------------------------- submit --------------------------------- */

function buildPayload(): Record<string, unknown> | null {
  const payload: Record<string, unknown> = {
    type: form.type,
    difficulty: form.difficulty,
    question: form.question.trim(),
    explanation: form.explanation,
    source: form.source,
    tags: form.tags,
    subject: form.subject.trim(),
    topic: form.topic.trim(),
    sub_topic: form.sub_topic.trim(),
  };

  if (!payload.question) {
    formError.value = 'Question text is required.';
    return null;
  }

  if (isChoice.value) {
    const options = form.options.filter((o) => o.text.trim());
    if (options.length < 2) {
      formError.value = 'At least two options are required.';
      return null;
    }
    if (!options.some((o) => o.correct)) {
      formError.value = 'Mark at least one option as correct.';
      return null;
    }
    payload.options = options;
  } else if (form.type === 'input') {
    if (!form.expectedText.trim()) {
      formError.value = 'Expected answer is required for input questions.';
      return null;
    }
    payload.expected_text = form.expectedText.trim();
  } else if (form.type === 'chronology') {
    const items = form.chronologyItems.filter((c) => c.text.trim());
    if (items.length < 2) {
      formError.value = 'At least two chronology items are required.';
      return null;
    }
    payload.chronology_items = items;
  }

  return payload;
}

const submit = () => {
  formError.value = null;
  // Pick up any tag the user typed but didn't commit with Enter.
  tagInput.value?.flush();
  const payload = buildPayload();
  if (!payload) return;
  emit('save', payload, imageFile.value);
};
</script>
