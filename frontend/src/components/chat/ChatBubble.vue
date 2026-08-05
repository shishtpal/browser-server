<template>
  <!-- User message -->
  <article v-if="message.role === 'user'"
    class="group relative ml-auto max-w-[82%] rounded-xl rounded-br-sm bg-indigo-600 px-3.5 py-2.5 text-white shadow-sm ring-1 ring-inset ring-indigo-500/40 dark:bg-indigo-600 dark:ring-indigo-400/30">
    <!-- Image attachments -->
    <div v-if="hasImages" class="mb-2 flex flex-wrap gap-2">
      <button
        v-for="att in imageAttachments"
        :key="att.id"
        type="button"
        class="relative overflow-hidden rounded-lg border border-white/20 bg-white/10 p-0.5 hover:bg-white/20"
        @click="openPreview(att)"
      >
        <img
          :src="imageUrl(att)"
          :alt="att.filename"
          class="h-16 w-16 object-cover sm:h-20 sm:w-20"
          loading="lazy"
          @error="onImageError(att.id)"
        />
      </button>
    </div>

    <pre class="whitespace-pre-wrap break-words font-sans text-[0.92em] leading-[1.6]">{{ message.content }}</pre>
    <div class="absolute -bottom-3 right-1 hidden items-center gap-0.5 rounded-lg border border-slate-200 bg-white p-0.5 shadow-sm group-hover:flex dark:border-white/10 dark:bg-slate-900">
      <button
        class="rounded-md p-1.5 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-slate-200"
        title="Copy" type="button" @click="$emit('copy', message.content)">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
        </svg>
      </button>
      <button
        class="rounded-md p-1.5 text-slate-400 transition hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400"
        title="Branch a new conversation from here" type="button" @click="$emit('branch', message.id)">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 3v12m0 0a3 3 0 103 3M6 15a3 3 0 013-3h6a3 3 0 003-3V6m0 0a3 3 0 10-3-3 3 3 0 003 3z" />
        </svg>
      </button>
      <button
        class="rounded-md p-1.5 text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-500/10 dark:hover:text-red-400"
        title="Delete message" type="button" @click="$emit('delete', message.id)">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
      </button>
    </div>
  </article>

  <!-- Assistant message -->
  <article v-else-if="message.role === 'assistant'"
    class="group relative max-w-[92%] rounded-xl rounded-bl-sm border border-slate-200/80 bg-white px-4 py-3 shadow-sm dark:border-white/10 dark:bg-slate-900/70">
    <div v-if="message.status === 'pending' && !message.content"
      class="flex items-center gap-2 text-[0.82em] font-medium text-slate-400">
      <span class="inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-indigo-500"></span>
      Thinking…
    </div>
    <div v-else
      class="prose prose-slate max-w-none break-words dark:prose-invert prose-p:text-[0.92em] prose-p:leading-[1.65] prose-li:text-[0.92em] prose-headings:font-semibold prose-headings:tracking-tight prose-h1:text-[1.2em] prose-h2:text-[1.1em] prose-h3:text-[1em] prose-pre:my-2 prose-pre:rounded-lg"
      v-html="renderedContent" @click="copyCodeBlock"></div>
    <div v-if="message.status === 'error'" class="mt-2 flex items-center gap-1.5 text-[0.82em] font-medium text-red-500">
      <span class="inline-block h-1.5 w-1.5 rounded-full bg-red-500"></span> Generation failed
    </div>
    <div v-if="message.status === 'cancelled'" class="mt-2 flex items-center gap-1.5 text-[0.82em] font-medium text-amber-500">
      <span class="inline-block h-1.5 w-1.5 rounded-full bg-amber-500"></span> Stopped
    </div>

    <div class="absolute -bottom-3 right-2 hidden items-center gap-0.5 rounded-lg border border-slate-200 bg-white p-0.5 shadow-sm group-hover:flex dark:border-white/10 dark:bg-slate-900">
      <button
        class="rounded-md p-1.5 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-slate-200"
        title="Copy" type="button" @click="$emit('copy', message.content)">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
        </svg>
      </button>
      <button
        class="rounded-md p-1.5 text-slate-400 transition hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400"
        title="Branch a new conversation from here" type="button" @click="$emit('branch', message.id)">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 3v12m0 0a3 3 0 103 3M6 15a3 3 0 013-3h6a3 3 0 003-3V6m0 0a3 3 0 10-3-3 3 3 0 003 3z" />
        </svg>
      </button>
      <button
        class="rounded-md p-1.5 text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-500/10 dark:hover:text-red-400"
        title="Delete message" type="button" @click="$emit('delete', message.id)">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
      </button>
    </div>
  </article>

  <!-- Tool message -->
  <article v-else-if="message.role === 'tool'"
    class="group relative w-full max-w-[92%] pr-8 text-slate-700 dark:text-slate-300">
    <button
      class="absolute right-0 top-0 hidden rounded-md p-1.5 text-slate-400 transition hover:bg-red-50 hover:text-red-600 group-hover:block dark:hover:bg-red-500/10 dark:hover:text-red-400"
      title="Delete message" type="button" @click="$emit('delete', message.id)">
      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
      </svg>
    </button>
    <button class="group flex w-full items-center gap-1.5 rounded-md py-1 text-left text-[0.8em]" type="button"
      :aria-expanded="expanded" @click="expanded = !expanded">
      <svg class="h-3.5 w-3.5 shrink-0 text-slate-400 dark:text-slate-500" fill="none" stroke="currentColor"
        viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
          d="M8 9h8M8 15h5M5 4h14a1 1 0 011 1v14a1 1 0 01-1 1H5a1 1 0 01-1-1V5a1 1 0 011-1z" />
      </svg>
      <span class="text-slate-400 dark:text-slate-500">{{ isRetryCall ? 'requested' : 'used' }}</span>
      <span class="font-mono font-medium text-slate-700 dark:text-slate-200">{{ toolLabel }}</span>
      <span class="ml-0.5 font-medium" :class="toolStatus.className">{{ toolStatus.icon }} {{ toolStatus.label }}</span>
      <svg
        class="ml-0.5 h-3.5 w-3.5 text-slate-400 transition-transform group-hover:text-slate-600 dark:group-hover:text-slate-200"
        :class="expanded ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m19 9-7 7-7-7" />
      </svg>
    </button>

    <div v-show="expanded" class="mt-2 space-y-2">
      <ChatQuestionForm v-if="isQuestionCall && message.status === 'pending' && !toolData.decision"
        :context="questionRequest?.context" :questions="questionRequest?.questions ?? []"
        @submit="submitQuestionAnswers" />
      <div v-else-if="message.status === 'pending' && !toolData.decision"
        class="rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-900/40 dark:bg-amber-950/20">
        <p class="mb-2 text-[0.85em] text-amber-800 dark:text-amber-200">{{ isRetryCall
          ? (retryMessage || 'The AI provider failed while continuing after a tool call. Resume without the last tool-call turn?')
          : 'Review the command or arguments before allowing this tool.' }}</p>
        <div class="flex flex-wrap items-center gap-2">
          <button
            class="rounded-md bg-emerald-600 px-3 py-1.5 text-[0.85em] font-semibold text-white transition hover:bg-emerald-700"
            type="button" @click="emit('tool-decision', message.tool_call_id || '', true, '')">{{ isRetryCall ? 'Resume' : 'Allow' }}</button>
          <button
            class="rounded-md border border-red-200 bg-white px-3 py-1.5 text-[0.85em] font-semibold text-red-600 transition hover:bg-red-50 dark:border-red-900/60 dark:bg-slate-950 dark:hover:bg-red-950/30"
            type="button" @click="emit('tool-decision', message.tool_call_id || '', false, '')">{{ isRetryCall ? 'Stop' : 'Reject' }}</button>
          <input v-model="commentDraft"
            class="min-w-48 flex-1 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-[0.85em] outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
            :placeholder="isRetryCall ? 'Or add instructions before resuming…' : 'Or send feedback instead…'" @keydown.enter.prevent="submitComment" />
          <button
            class="rounded-md bg-slate-700 px-3 py-1.5 text-[0.85em] font-semibold text-white transition hover:bg-slate-800 disabled:opacity-40 dark:bg-slate-200 dark:text-slate-900 dark:hover:bg-white"
            type="button" :disabled="!commentDraft.trim()" @click="submitComment">Send</button>
        </div>
      </div>

      <div v-if="toolData.decision === 'commented'"
        class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-[0.85em] text-amber-800 dark:border-amber-800/40 dark:bg-amber-900/20 dark:text-amber-200">
        <span class="font-semibold">Your feedback:</span> {{ feedbackComment }}
      </div>

      <section v-for="section in toolSections" :key="section.label"
        class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-white/10 dark:bg-slate-900">
        <header
          class="flex h-7 items-center justify-between border-b border-slate-200 bg-slate-50 px-2.5 dark:border-white/10 dark:bg-slate-800/60">
          <span class="text-[0.65em] font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">{{
            section.label }}</span>
          <button
            class="rounded p-1 text-slate-400 transition hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-slate-200"
            type="button" :title="`Copy ${section.label.toLowerCase()}`"
            :aria-label="`Copy ${section.label.toLowerCase()}`" @click="emit('copy', section.copyValue)">
            <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                d="M8 16H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v2m-6 12h8a2 2 0 0 0 2-2v-8a2 2 0 0 0-2-2h-8a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2Z" />
            </svg>
          </button>
        </header>
        <pre
          class="max-h-64 overflow-auto whitespace-pre-wrap break-words px-2.5 py-2 font-mono text-[0.78em] leading-[1.55] text-slate-800 dark:text-slate-200">{{ section.content }}</pre>
      </section>
    </div>
  </article>

  <!-- Image preview modal -->
  <Teleport v-if="previewAttachment" to="body">
    <div
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
      @click="closePreview"
    >
      <div class="relative max-h-[90vh] max-w-[90vw] overflow-hidden rounded-lg bg-white shadow-2xl dark:bg-slate-900">
        <img
          :src="imageUrl(previewAttachment)"
          :alt="previewAttachment.filename"
          class="max-h-[80vh] max-w-[85vw] object-contain"
          @error="onImageError(previewAttachment.id)"
        />
        <div class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/70 to-transparent px-4 py-3">
          <p class="truncate text-sm font-medium text-white">{{ previewAttachment.filename }}</p>
          <p class="text-xs text-white/80">{{ formatBytes(previewAttachment.size_bytes) }}</p>
        </div>
        <button
          type="button"
          class="absolute right-2 top-2 rounded-full bg-black/50 p-1.5 text-white hover:bg-black/70"
          aria-label="Close preview"
          @click.stop="closePreview"
        >
          <svg class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { AIImageAttachment, AIMessage, ChatQuestion } from '@browser-server/shared-types'
import { computed, ref } from 'vue'
import { renderMarkdown } from './markdown'
import ChatQuestionForm from './ChatQuestionForm.vue'
import { getAIImageAttachmentUrl } from '../../lib/api'

const props = defineProps<{
  message: AIMessage
}>()

const emit = defineEmits<{
  copy: [content: string]
  delete: [messageId: string]
  branch: [messageId: string]
  'tool-decision': [callId: string, approved: boolean, comment: string]
}>()

const commentDraft = ref('')
const expanded = ref(true)
const previewAttachment = ref<AIImageAttachment | null>(null)
const brokenImageIds = ref<Set<string>>(new Set())

const imageAttachments = computed<AIImageAttachment[]>(() =>
  (props.message.attachments ?? []).filter((a) => !brokenImageIds.value.has(a.id))
)
const hasImages = computed(() => imageAttachments.value.length > 0)

function imageUrl(att: AIImageAttachment): string {
  return getAIImageAttachmentUrl(props.message.conversation_id, att.id)
}

function onImageError(id: string) {
  brokenImageIds.value.add(id)
}

function openPreview(att: AIImageAttachment) {
  previewAttachment.value = att
}

function closePreview() {
  previewAttachment.value = null
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function submitComment() {
  const text = commentDraft.value.trim()
  if (!text) return
  emit('tool-decision', props.message.tool_call_id || '', false, text)
  commentDraft.value = ''
}

const isQuestionCall = computed(() => toolData.value.name === 'ask_questions')
const isRetryCall = computed(() => toolData.value.name === 'retry_tool_call')
const retryMessage = computed(() => {
  if (!isRetryCall.value || !isRecord(toolData.value.args)) return ''
  return typeof toolData.value.args.message === 'string' ? toolData.value.args.message : ''
})
const questionRequest = computed(() => {
  if (!isQuestionCall.value || !isRecord(toolData.value.args)) return null
  const questions = Array.isArray(toolData.value.args.questions) ? toolData.value.args.questions : []
  return {
    context: typeof toolData.value.args.context === 'string' ? toolData.value.args.context : '',
    questions: questions.filter(isQuestion).map((question) => ({
      id: question.id,
      prompt: question.prompt,
      kind: question.kind,
      options: question.options,
      default: question.default,
      required: question.required,
    })),
  }
})

function submitQuestionAnswers(answers: Array<{ id: string; prompt: string; answer: unknown; skipped: boolean }>) {
  emit('tool-decision', props.message.tool_call_id || '', false, JSON.stringify({ answers }))
}

function copyCodeBlock(event: MouseEvent) {
  if (!(event.target instanceof Element)) return
  const button = event.target.closest<HTMLButtonElement>('[data-copy-code]')
  if (!button) return
  const code = button.parentElement?.querySelector<HTMLElement>('code')
  if (code) emit('copy', code.innerText)
}

const renderedContent = computed(() => renderMarkdown(props.message.content))

interface ToolData {
  name: string
  args: unknown
  result: unknown
  decision: 'approved' | 'rejected' | 'commented' | 'answered' | null
}

const toolData = computed<ToolData>(() => {
  if (props.message.role !== 'tool') return { name: '', args: null, result: null, decision: null }
  try {
    const parsed = JSON.parse(props.message.content)
    return {
      name: parsed.tool || '',
      args: parsed.args ?? null,
      result: parsed.result ?? parsed,
      decision: parsed.decision ?? null,
    }
  } catch {
    return { name: '', args: null, result: props.message.content, decision: null }
  }
})

interface ToolSection {
  label: string
  content: string
  copyValue: string
}

const resultRecord = computed<Record<string, unknown> | null>(() =>
  isRecord(toolData.value.result) ? toolData.value.result : null
)

const toolLabel = computed(() => {
  if (toolData.value.name === 'execute_command') return 'Shell'
  if (toolData.value.name === 'retry_tool_call') return 'tool-call recovery'
  if (!toolData.value.name) return 'Tool'
  return toolData.value.name
    .split('_')
    .filter(Boolean)
    .map((word) => word[0].toUpperCase() + word.slice(1))
    .join(' ')
})

const feedbackComment = computed(() => {
  const comment = resultRecord.value?.comment
  return typeof comment === 'string' ? comment : ''
})

const messageIsFinished = computed(() => props.message.status !== 'pending' || toolData.value.decision === 'commented')

const toolSections = computed<ToolSection[]>(() => {
  if (toolData.value.name === 'execute_command' && isRecord(toolData.value.args)) {
    const command = typeof toolData.value.args.command === 'string' ? toolData.value.args.command : ''
    const stdout = resultRecord.value?.stdout
    const stderr = resultRecord.value?.stderr
    const error = resultRecord.value?.error
    const stderrText = [
      typeof stderr === 'string' ? stderr.trimEnd() : '',
      typeof error === 'string' ? error : '',
    ].filter(Boolean).join('\n')
    const sections: ToolSection[] = []

    if (command) sections.push({ label: 'Command', content: `$ ${command}`, copyValue: command })
    if (messageIsFinished.value && typeof stdout === 'string') {
      sections.push({ label: 'Stdout', content: stdout || '(no output)', copyValue: stdout })
    }
    if (messageIsFinished.value && (typeof stderr === 'string' || typeof error === 'string')) {
      sections.push({ label: 'Stderr', content: stderrText || '(no output)', copyValue: stderrText })
    }
    return sections
  }

  const sections: ToolSection[] = []
  if (toolData.value.args !== null && toolData.value.args !== undefined) {
    const args = formatJson(toolData.value.args)
    sections.push({ label: 'Arguments', content: args, copyValue: args })
  }
  if (messageIsFinished.value && toolData.value.result !== null && toolData.value.result !== undefined) {
    const result = formatJson(toolData.value.result)
    sections.push({ label: 'Result', content: result || '(no output)', copyValue: result })
  }
  return sections
})

const toolStatus = computed(() => {
  if (props.message.status === 'pending' && !toolData.value.decision) {
    return {
      label: isRetryCall.value ? 'resume required' : 'approval required',
      icon: '!',
      className: 'text-amber-600 dark:text-amber-400',
    }
  }
  if (toolData.value.decision === 'commented') {
    return { label: 'commented', icon: '•', className: 'text-amber-600 dark:text-amber-400' }
  }
  if (toolData.value.decision === 'answered') {
    return { label: 'answered', icon: '✓', className: 'text-emerald-600 dark:text-emerald-400' }
  }
  if (props.message.status === 'pending') {
    return { label: isRetryCall.value ? 'resuming' : 'running', icon: '•', className: 'text-blue-600 dark:text-blue-400' }
  }
  const exitCode = resultRecord.value?.exit_code
  const failed = props.message.status === 'error' || resultRecord.value?.error || (typeof exitCode === 'number' && exitCode !== 0)
  if (failed) {
    const rejected = resultRecord.value?.error === 'rejected by user'
    return { label: rejected ? 'rejected' : 'failed', icon: '×', className: 'text-red-600 dark:text-red-400' }
  }
  return { label: '', icon: '✓', className: 'text-emerald-600 dark:text-emerald-400' }
})

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isQuestion(value: unknown): value is ChatQuestion {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.prompt !== 'string') return false
  return value.kind === undefined || ['text', 'choice', 'multi_choice', 'multiple_choice', 'confirm'].includes(value.kind as string)
}

function formatJson(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}
</script>
