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
      <TextAreaField v-model="form.question" placeholder="Enter the question text" required :rows="4" />
    </FormField>

    <!-- Options editor for choice types -->
    <div
      v-if="isChoice"
      class="space-y-4 rounded-xl border border-gray-200 bg-gray-50/50 p-3 sm:rounded-2xl sm:p-5 dark:border-slate-700 dark:bg-slate-800/30"
    >
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="min-w-0">
          <span class="block text-sm font-bold text-slate-700 dark:text-slate-300">Options</span>
          <span class="block text-xs text-slate-500 dark:text-slate-400">
            Tap the letter to mark {{ form.type === 'single_choice' ? 'exactly one' : 'one or more' }} correct
          </span>
        </div>
        <Button variant="secondary" size="sm" class="shrink-0" @click="addOption" :disabled="form.options.length >= 10">
          + Add option
        </Button>
      </div>

      <TransitionGroup tag="div" name="row" class="space-y-2">
        <div v-for="(opt, i) in form.options" :key="i" class="flex items-center gap-2 sm:gap-3">
          <!-- Letter / correct toggle -->
          <button
            type="button"
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border-2 text-sm font-bold transition sm:h-10 sm:w-10"
            :class="opt.correct
              ? 'border-emerald-500 bg-emerald-500 text-white shadow-sm shadow-emerald-500/30'
              : 'border-gray-300 bg-white text-slate-500 hover:border-emerald-400 hover:text-emerald-600 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-400 dark:hover:border-emerald-500'"
            :title="opt.correct ? 'Correct answer' : 'Mark as correct'"
            :aria-pressed="opt.correct"
            @click="toggleCorrect(i)"
          >
            <svg v-if="opt.correct" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
            </svg>
            <span v-else>{{ String.fromCharCode(65 + i) }}</span>
          </button>

          <div class="min-w-0 flex-1">
            <InputField v-model="opt.text" :placeholder="`Option ${String.fromCharCode(65 + i)}`" flex />
          </div>

          <button
            type="button"
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-slate-400 transition hover:bg-rose-50 hover:text-rose-500 disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-slate-400 dark:hover:bg-rose-900/20"
            :disabled="form.options.length <= 2"
            aria-label="Remove option"
            @click="removeOption(i)"
          >
            <svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
            </svg>
          </button>
        </div>
      </TransitionGroup>
    </div>

    <!-- Expected text for input type -->
    <FormField v-else-if="form.type === 'input'" label="Expected answer" required>
      <InputField v-model="form.expectedText" placeholder="The correct answer" required />
    </FormField>

    <!-- Chronology editor -->
    <div
      v-else-if="form.type === 'chronology'"
      class="space-y-4 rounded-xl border border-gray-200 bg-gray-50/50 p-3 sm:rounded-2xl sm:p-5 dark:border-slate-700 dark:bg-slate-800/30"
    >
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="min-w-0">
          <span class="block text-sm font-bold text-slate-700 dark:text-slate-300">Items</span>
          <span class="block text-xs text-slate-500 dark:text-slate-400">Set the correct order number for each item</span>
        </div>
        <Button variant="secondary" size="sm" class="shrink-0" @click="addChronologyItem" :disabled="form.chronologyItems.length >= 20">
          + Add item
        </Button>
      </div>

      <TransitionGroup tag="div" name="row" class="space-y-2">
        <div v-for="(item, i) in form.chronologyItems" :key="i" class="flex items-center gap-2 sm:gap-3">
          <div class="w-14 shrink-0 sm:w-20">
            <InputField v-model.number="item.correct_order" type="number" aria-label="Correct order" />
          </div>
          <div class="min-w-0 flex-1">
            <InputField v-model="item.text" :placeholder="`Item ${i + 1}`" flex />
          </div>
          <button
            type="button"
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-slate-400 transition hover:bg-rose-50 hover:text-rose-500 disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-slate-400 dark:hover:bg-rose-900/20"
            :disabled="form.chronologyItems.length <= 2"
            aria-label="Remove item"
            @click="removeChronologyItem(i)"
          >
            <svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
          </svg>
          </button>
        </div>
      </TransitionGroup>
    </div>

    <FormField label="Explanation (markdown)">
      <TextAreaField v-model="form.explanation" placeholder="Optional explanation shown after answering" :rows="3" />
    </FormField>

    <!-- Tags & taxonomy -->
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <FormField label="Tags" help-text="e.g. SSC, RRB. Press enter to add." class="sm:col-span-2 lg:col-span-1">
        <div class="flex flex-wrap items-center gap-1.5 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 shadow-sm focus-within:border-indigo-400 focus-within:ring-4 focus-within:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:focus-within:ring-indigo-900/30">
          <span
            v-for="tag in form.tags"
            :key="tag"
            class="flex items-center gap-1 rounded bg-violet-100 px-2 py-0.5 text-xs font-semibold text-violet-800 dark:bg-violet-900/40 dark:text-violet-200"
          >
            {{ tag }}
            <button
              type="button"
              class="-mr-0.5 flex h-4 w-4 items-center justify-center rounded-full text-violet-600 transition hover:bg-violet-200 hover:text-violet-900 dark:text-violet-300 dark:hover:bg-violet-800"
              :aria-label="`Remove tag ${tag}`"
              @click="removeTag(tag)"
            >
              ×
            </button>
          </span>
          <input
            v-model="tagDraft"
            type="text"
            list="quiz-tags-vocab"
            placeholder="Type tag…"
            class="min-w-[8ch] flex-1 border-0 bg-transparent p-0 text-sm font-semibold text-slate-700 placeholder:font-normal focus:outline-none dark:text-slate-200"
            @keydown.enter.prevent="commitTagDraft"
            @keydown.,.prevent="commitTagDraft"
            @blur="commitTagDraft"
          />
          <datalist id="quiz-tags-vocab">
            <option v-for="v in vocabulary?.tags ?? []" :key="v" :value="v" />
          </datalist>
        </div>
      </FormField>

      <FormField v-for="field in tagFields" :key="field.key" :label="field.label">
        <InputField
          v-model="form[field.key]"
          :placeholder="field.placeholder"
          :list="`quiz-${field.key}`"
        />
        <datalist :id="`quiz-${field.key}`">
          <option v-for="v in vocabulary?.[field.vocabKey] ?? []" :key="v" :value="v" />
        </datalist>
      </FormField>
    </div>

    <!-- Image -->
    <FormField label="Image (optional)">
      <div class="space-y-3">
        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          class="w-full text-sm font-semibold text-slate-500 file:mr-3 file:rounded-lg file:border-0 file:bg-indigo-100 file:px-4 file:py-2 file:text-sm file:font-bold file:text-indigo-700 transition hover:file:bg-indigo-200 dark:file:bg-indigo-900/30 dark:file:text-indigo-300 dark:hover:file:bg-indigo-900/50"
          @change="onFile"
        />
        <div v-if="imagePreview" class="relative inline-block">
          <img
            :src="imagePreview"
            alt="Selected image preview"
            class="max-h-40 rounded-xl border border-gray-200 object-contain shadow-sm dark:border-slate-700"
          />
          <button
            type="button"
            class="absolute -right-2 -top-2 flex h-7 w-7 items-center justify-center rounded-full bg-rose-500 text-white shadow transition hover:bg-rose-600"
            aria-label="Remove image"
            @click="clearImage"
          >
            <svg class="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
            </svg>
          </button>
        </div>
      </div>
    </FormField>

    <div
      v-if="formError"
      class="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm font-semibold text-rose-700 dark:border-rose-800/40 dark:bg-rose-900/20 dark:text-rose-300"
      role="alert"
    >
      <svg class="mt-0.5 h-4 w-4 shrink-0" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm-1-4a1 1 0 112 0 1 1 0 01-2 0zm1-9a.75.75 0 00-.75.75v4.5a.75.75 0 001.5 0v-4.5A.75.75 0 0010 5z" clip-rule="evenodd" />
      </svg>
      {{ formError }}
    </div>

    <!-- Sticky action bar -->
    <div
      class="sticky bottom-0 z-10 -mx-3 flex flex-col-reverse gap-2 border-t border-gray-200 bg-white/95 px-3 py-3 backdrop-blur sm:-mx-6 sm:flex-row sm:justify-end sm:gap-3 sm:px-6 lg:-mx-8 lg:px-8 dark:border-slate-700 dark:bg-slate-900/95"
    >
      <Button
        v-if="isEdit"
        variant="ghost"
        class="w-full sm:w-auto"
        @click="$emit('cancel')"
      >
        Cancel
      </Button>
      <Button
        type="submit"
        class="w-full sm:w-auto"
        :loading="isSaving"
        :loading-text="isEdit ? 'Saving...' : 'Adding...'"
      >
        {{ isEdit ? 'Save changes' : 'Add question' }}
      </Button>
    </div>
  </form>
</template>

<script setup lang="ts">
import type { QuestionOption, QuestionResponse, QuestionType, TagVocabulary } from '../../types'
import { reactive, ref, computed, watch, onBeforeUnmount } from 'vue'
import FormField from '../ui/FormField.vue'
import InputField from '../ui/InputField.vue'
import SelectField from '../ui/SelectField.vue'
import TextAreaField from '../ui/TextAreaField.vue'
import Button from '../ui/Button.vue'

const props = defineProps<{
  question?: QuestionResponse | null
  vocabulary?: TagVocabulary | null
  isSaving?: boolean
}>()

const emit = defineEmits<{
  save: [payload: Record<string, unknown>, image: File | null]
  cancel: []
}>()

const isEdit = computed(() => !!props.question)

interface FormState {
  type: QuestionType
  difficulty: 'easy' | 'medium' | 'hard'
  question: string
  explanation: string
  source: string
  tags: string[]
  subject: string
  topic: string
  sub_topic: string
  options: QuestionOption[]
  chronologyItems: { index: number; text: string; correct_order: number }[]
  expectedText: string
}

const blankOptions = (): QuestionOption[] => [
  { index: 0, text: '', correct: false },
  { index: 1, text: '', correct: false },
]
const blankChronology = () => [
  { index: 0, text: '', correct_order: 1 },
  { index: 1, text: '', correct_order: 2 },
]

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
})

const imageFile = ref<File | null>(null)
const imagePreview = ref<string | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const formError = ref<string | null>(null)
const tagDraft = ref('')

const isChoice = computed(() => form.type === 'single_choice' || form.type === 'multiple_choice')

const tagFields = [
  { key: 'subject', label: 'Subject', placeholder: 'Math', vocabKey: 'subjects' },
  { key: 'topic', label: 'Topic', placeholder: 'Algebra', vocabKey: 'topics' },
  { key: 'sub_topic', label: 'Sub-topic', placeholder: 'Equations', vocabKey: 'sub_topics' },
] as const

function commitTagDraft() {
  const value = tagDraft.value.trim()
  if (!value) return
  if (!form.tags.includes(value)) form.tags.push(value)
  tagDraft.value = ''
}

function removeTag(tag: string) {
  form.tags = form.tags.filter((t) => t !== tag)
}

watch(
  () => props.question,
  (q) => {
    if (!q) return
    form.type = q.type
    form.difficulty = q.difficulty
    form.question = q.question
    form.explanation = q.explanation
    form.source = q.source
    form.tags = Array.isArray(q.tags) ? [...q.tags] : []
    form.subject = q.subject
    form.topic = q.topic
    form.sub_topic = q.sub_topic
    form.options = q.options?.length ? q.options.map((o) => ({ ...o })) : blankOptions()
    form.chronologyItems = q.chronology_items?.length
      ? q.chronology_items.map((c) => ({ ...c }))
      : blankChronology()
    form.expectedText = q.expected_text ?? ''
  },
  { immediate: true },
)

const addOption = () => {
  if (form.options.length >= 10) return
  form.options.push({ index: form.options.length, text: '', correct: false })
}
const removeOption = (i: number) => {
  form.options.splice(i, 1)
  form.options.forEach((o, idx) => (o.index = idx))
}
const toggleCorrect = (i: number) => {
  if (form.type === 'single_choice') {
    form.options.forEach((o, idx) => (o.correct = idx === i))
  } else {
    form.options[i].correct = !form.options[i].correct
  }
}

const addChronologyItem = () => {
  if (form.chronologyItems.length >= 20) return
  form.chronologyItems.push({
    index: form.chronologyItems.length,
    text: '',
    correct_order: form.chronologyItems.length + 1,
  })
}
const removeChronologyItem = (i: number) => {
  form.chronologyItems.splice(i, 1)
  form.chronologyItems.forEach((c, idx) => (c.index = idx))
}

const revokePreview = () => {
  if (imagePreview.value) {
    URL.revokeObjectURL(imagePreview.value)
    imagePreview.value = null
  }
}

const onFile = (e: Event) => {
  const target = e.target as HTMLInputElement
  revokePreview()
  imageFile.value = target.files?.[0] ?? null
  if (imageFile.value) {
    imagePreview.value = URL.createObjectURL(imageFile.value)
  }
}

const clearImage = () => {
  revokePreview()
  imageFile.value = null
  if (fileInput.value) fileInput.value.value = ''
}

onBeforeUnmount(revokePreview)

const submit = () => {
  formError.value = null
  // Pick up any tag the user typed but didn't commit with Enter.
  if (tagDraft.value.trim()) commitTagDraft()
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
  }
  if (!payload.question) {
    formError.value = 'Question text is required.'
    return
  }
  if (isChoice.value) {
    const options = form.options.filter((o) => o.text.trim())
    if (options.length < 2) {
      formError.value = 'At least two options are required.'
      return
    }
    if (!options.some((o) => o.correct)) {
      formError.value = 'Mark at least one option as correct.'
      return
    }
    payload.options = options
  } else if (form.type === 'input') {
    if (!form.expectedText.trim()) {
      formError.value = 'Expected answer is required for input questions.'
      return
    }
    payload.expected_text = form.expectedText.trim()
  } else if (form.type === 'chronology') {
    const items = form.chronologyItems.filter((c) => c.text.trim())
    if (items.length < 2) {
      formError.value = 'At least two chronology items are required.'
      return
    }
    payload.chronology_items = items
  }
  emit('save', payload, imageFile.value)
}
</script>

<style scoped>
.row-enter-active,
.row-leave-active {
  transition: all 0.2s ease;
}
.row-enter-from,
.row-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>