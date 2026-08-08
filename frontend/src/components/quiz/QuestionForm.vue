<template>
  <form class="space-y-3" @submit.prevent="submit">
    <div class="grid gap-3 sm:grid-cols-3">
      <label class="block">
        <span class="mb-1 block text-[11px] font-bold text-slate-600 dark:text-slate-400"
          >Type</span
        >
        <select
          v-model="form.type"
          :disabled="isEdit"
          class="w-full rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        >
          <option value="single_choice">Single choice</option>
          <option value="multiple_choice">Multiple choice</option>
          <option value="input">Input (text answer)</option>
          <option value="chronology">Chronology (ordering)</option>
        </select>
      </label>
      <label class="block">
        <span class="mb-1 block text-[11px] font-bold text-slate-600 dark:text-slate-400"
          >Difficulty</span
        >
        <select
          v-model="form.difficulty"
          class="w-full rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        >
          <option value="easy">Easy</option>
          <option value="medium">Medium</option>
          <option value="hard">Hard</option>
        </select>
      </label>
      <label class="block">
        <span class="mb-1 block text-[11px] font-bold text-slate-600 dark:text-slate-400"
          >Source</span
        >
        <input
          v-model="form.source"
          type="text"
          placeholder="e.g. SSC CGL 2023"
          class="w-full rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        />
      </label>
    </div>

    <label class="block">
      <span class="mb-1 block text-[11px] font-bold text-slate-600 dark:text-slate-400"
        >Question</span
      >
      <textarea
        v-model="form.question"
        rows="3"
        required
        placeholder="Enter the question text"
        class="w-full rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
      ></textarea>
    </label>

    <!-- Options editor for choice types -->
    <div v-if="isChoice">
      <div class="mb-1 flex items-center justify-between">
        <span class="text-[11px] font-bold text-slate-600 dark:text-slate-400">
          Options (mark {{ form.type === 'single_choice' ? 'exactly one' : 'one or more' }} correct)
        </span>
        <button
          type="button"
          class="rounded-md bg-indigo-100 px-2 py-0.5 text-[11px] font-bold text-indigo-700 transition hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-300"
          @click="addOption"
        >
          + Add option
        </button>
      </div>
      <div class="space-y-1.5">
        <div v-for="(opt, i) in form.options" :key="i" class="flex items-center gap-2">
          <input
            :type="form.type === 'single_choice' ? 'radio' : 'checkbox'"
            :name="radioGroup"
            :checked="opt.correct"
            class="h-3.5 w-3.5 accent-indigo-600"
            @change="toggleCorrect(i)"
          />
          <input
            v-model="opt.text"
            type="text"
            :placeholder="`Option ${String.fromCharCode(65 + i)}`"
            class="flex-1 rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          />
          <button
            type="button"
            class="text-slate-400 transition hover:text-rose-500"
            :disabled="form.options.length <= 2"
            @click="removeOption(i)"
          >
            ✕
          </button>
        </div>
      </div>
    </div>

    <!-- Expected text for input type -->
    <label v-else-if="form.type === 'input'" class="block">
      <span class="mb-1 block text-[11px] font-bold text-slate-600 dark:text-slate-400"
        >Expected answer</span
      >
      <input
        v-model="form.expectedText"
        type="text"
        required
        placeholder="The correct answer"
        class="w-full rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
      />
    </label>

    <!-- Chronology editor -->
    <div v-else-if="form.type === 'chronology'">
      <div class="mb-1 flex items-center justify-between">
        <span class="text-[11px] font-bold text-slate-600 dark:text-slate-400"
          >Items (set the correct order)</span
        >
        <button
          type="button"
          class="rounded-md bg-indigo-100 px-2 py-0.5 text-[11px] font-bold text-indigo-700 transition hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-300"
          @click="addChronologyItem"
        >
          + Add item
        </button>
      </div>
      <div class="space-y-1.5">
        <div v-for="(item, i) in form.chronologyItems" :key="i" class="flex items-center gap-2">
          <input
            v-model.number="item.correct_order"
            type="number"
            min="1"
            :max="form.chronologyItems.length"
            class="w-14 rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-center text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          />
          <input
            v-model="item.text"
            type="text"
            :placeholder="`Item ${i + 1}`"
            class="flex-1 rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          />
          <button
            type="button"
            class="text-slate-400 transition hover:text-rose-500"
            :disabled="form.chronologyItems.length <= 2"
            @click="removeChronologyItem(i)"
          >
            ✕
          </button>
        </div>
      </div>
    </div>

    <label class="block">
      <span class="mb-1 block text-[11px] font-bold text-slate-600 dark:text-slate-400"
        >Explanation (markdown)</span
      >
      <textarea
        v-model="form.explanation"
        rows="3"
        placeholder="Optional explanation shown after answering"
        class="w-full rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
      ></textarea>
    </label>

    <div class="grid gap-3 sm:grid-cols-4">
      <label class="block">
        <span class="mb-1 block text-[11px] font-bold text-slate-600 dark:text-slate-400"
          >Tags</span
        >
        <div
          class="flex flex-wrap items-center gap-1 rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        >
          <span
            v-for="tag in form.tags"
            :key="tag"
            class="flex items-center gap-1 rounded bg-violet-100 px-1.5 py-0.5 text-[10px] font-semibold text-violet-800 dark:bg-violet-900/40 dark:text-violet-200"
          >
            {{ tag }}
            <button
              type="button"
              class="text-violet-600 hover:text-violet-800 dark:text-violet-300"
              @click="removeTag(tag)"
            >
              ×
            </button>
          </span>
          <input
            v-model="tagDraft"
            type="text"
            list="quiz-tags-vocab"
            placeholder="Type tag, press Enter…"
            class="min-w-[8ch] flex-1 border-0 bg-transparent p-0 text-xs focus:outline-none dark:text-slate-200"
            @keydown.enter.prevent="commitTagDraft"
            @keydown.,.prevent="commitTagDraft"
            @blur="commitTagDraft"
          />
          <datalist id="quiz-tags-vocab">
            <option v-for="v in vocabulary?.tags ?? []" :key="v" :value="v" />
          </datalist>
        </div>
        <p class="mt-1 text-[10px] text-slate-400">
          e.g. SSC, RRB, Banking. A question can carry any number of tags.
        </p>
      </label>
      <label v-for="field in tagFields" :key="field.key" class="block">
        <span class="mb-1 block text-[11px] font-bold text-slate-600 dark:text-slate-400">{{
          field.label
        }}</span>
        <input
          v-model="form[field.key]"
          type="text"
          :list="`quiz-${field.key}`"
          :placeholder="field.placeholder"
          class="w-full rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        />
        <datalist :id="`quiz-${field.key}`">
          <option v-for="v in vocabulary?.[field.vocabKey] ?? []" :key="v" :value="v" />
        </datalist>
      </label>
    </div>

    <label class="block">
      <span class="mb-1 block text-[11px] font-bold text-slate-600 dark:text-slate-400"
        >Image (optional)</span
      >
      <input
        type="file"
        accept="image/*"
        class="w-full text-xs text-slate-500 file:mr-2 file:rounded-lg file:border-0 file:bg-indigo-100 file:px-3 file:py-1.5 file:text-xs file:font-bold file:text-indigo-700 dark:file:bg-indigo-900/30 dark:file:text-indigo-300"
        @change="onFile"
      />
    </label>

    <p
      v-if="formError"
      class="rounded-lg bg-rose-50 px-3 py-2 text-xs font-semibold text-rose-700 dark:bg-rose-900/20 dark:text-rose-300"
    >
      {{ formError }}
    </p>

    <div class="flex justify-end gap-2">
      <button
        v-if="isEdit"
        type="button"
        class="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-bold text-slate-600 transition hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
        @click="$emit('cancel')"
      >
        Cancel
      </button>
      <button
        type="submit"
        :disabled="isSaving"
        class="rounded-lg bg-indigo-600 px-4 py-1.5 text-xs font-bold text-white transition hover:bg-indigo-700 disabled:opacity-50"
      >
        {{ isSaving ? 'Saving…' : isEdit ? 'Save changes' : 'Add question' }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { reactive, ref, computed, watch } from 'vue'
import type { QuestionOption, QuestionResponse, QuestionType, TagVocabulary } from '../../types'

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
const radioGroup = `quiz-correct-${Math.random().toString(36).slice(2)}`

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

const onFile = (e: Event) => {
  const target = e.target as HTMLInputElement
  imageFile.value = target.files?.[0] ?? null
}

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
