<template>
  <form class="space-y-4 rounded border border-indigo-200 bg-indigo-50 p-3 dark:border-indigo-900/60 dark:bg-indigo-950/20" @submit.prevent="submit">
    <div>
      <p class="text-[0.85em] font-semibold text-indigo-900 dark:text-indigo-200">A clarification is needed</p>
      <p v-if="context" class="mt-1 whitespace-pre-wrap text-[0.8em] text-indigo-800 dark:text-indigo-300">{{ context }}</p>
    </div>

    <fieldset v-for="question in questions" :key="question.id" class="space-y-1.5">
      <legend class="text-[0.85em] font-medium text-slate-800 dark:text-slate-200">
        {{ question.prompt }}<span v-if="question.required" class="text-red-500"> *</span>
      </legend>

      <input
        v-if="question.kind === 'text'"
        v-model="answers[question.id]"
        class="w-full rounded border border-slate-200 bg-white px-2.5 py-1.5 text-[0.85em] outline-none focus:border-indigo-400 dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
        type="text"
        :placeholder="question.default || 'Your answer'"
      />

      <div v-else-if="question.kind === 'choice'" class="space-y-1">
        <label v-for="option in question.options" :key="option" class="flex cursor-pointer items-center gap-2 text-[0.85em] text-slate-700 dark:text-slate-300">
          <input v-model="answers[question.id]" :name="`question-${question.id}`" type="radio" :value="option" />
          {{ option }}
        </label>
      </div>

      <div v-else-if="isMultipleChoice(question.kind)" class="space-y-1">
        <label v-for="option in question.options" :key="option" class="flex cursor-pointer items-center gap-2 text-[0.85em] text-slate-700 dark:text-slate-300">
          <input v-model="answers[question.id]" type="checkbox" :value="option" />
          {{ option }}
        </label>
      </div>

      <div v-else class="flex gap-4 text-[0.85em] text-slate-700 dark:text-slate-300">
        <label class="flex items-center gap-2"><input v-model="answers[question.id]" :name="`confirm-${question.id}`" type="radio" value="yes" /> Yes</label>
        <label class="flex items-center gap-2"><input v-model="answers[question.id]" :name="`confirm-${question.id}`" type="radio" value="no" /> No</label>
      </div>
    </fieldset>

    <p v-if="error" class="text-[0.8em] text-red-600 dark:text-red-400">{{ error }}</p>
    <button class="rounded bg-indigo-600 px-3 py-1.5 text-[0.85em] font-semibold text-white transition hover:bg-indigo-700 disabled:opacity-40" type="submit" :disabled="submitting">
      {{ submitting ? 'Sending…' : 'Submit answers' }}
    </button>
  </form>
</template>

<script setup lang="ts">
import type { ChatQuestion } from '@browser-server/shared-types'
import { reactive, ref } from 'vue'

const props = defineProps<{
  context?: string
  questions: ChatQuestion[]
}>()

const emit = defineEmits<{
  submit: [answers: Array<{ id: string; prompt: string; answer: unknown; skipped: boolean }>]
}>()

const answers = reactive<Record<string, string | string[]>>({})
const error = ref('')
const submitting = ref(false)

for (const question of props.questions) {
  answers[question.id] = isMultipleChoice(question.kind) ? [] : question.default || ''
}

function isMultipleChoice(kind: ChatQuestion['kind']) {
  return kind === 'multi_choice' || kind === 'multiple_choice'
}

function isEmpty(answer: string | string[]) {
  return Array.isArray(answer) ? answer.length === 0 : !answer.trim()
}

function submit() {
  for (const question of props.questions) {
    if (question.required && isEmpty(answers[question.id] || '')) {
      error.value = `Please answer: ${question.prompt}`
      return
    }
  }
  error.value = ''
  submitting.value = true
  emit('submit', props.questions.map((question) => {
    const answer = answers[question.id] || (isMultipleChoice(question.kind) ? [] : '')
    return { id: question.id, prompt: question.prompt, answer, skipped: isEmpty(answer) }
  }))
}
</script>
