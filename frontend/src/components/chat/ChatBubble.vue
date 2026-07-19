<template>
  <!-- User message -->
  <article
    v-if="message.role === 'user'"
    class="group relative ml-auto max-w-[80%] rounded-2xl rounded-br-md bg-slate-900 px-4 py-3 text-white shadow-sm dark:bg-white dark:text-slate-900"
  >
    <pre class="whitespace-pre-wrap break-words font-sans text-sm leading-relaxed">{{ message.content }}</pre>
    <!-- Copy button -->
    <button
      class="absolute right-2 top-2 hidden rounded-md p-1.5 text-white/50 transition hover:bg-white/10 hover:text-white group-hover:block dark:text-slate-400 dark:hover:bg-slate-900/10 dark:hover:text-slate-900"
      title="Copy"
      type="button"
      @click="$emit('copy', message.content)"
    >
      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
    </button>
  </article>

  <!-- Assistant message -->
  <article
    v-else-if="message.role === 'assistant'"
    class="group relative max-w-[90%] rounded-2xl rounded-bl-md border border-slate-200 bg-white px-4 py-3 shadow-sm dark:border-white/10 dark:bg-slate-900"
  >
    <div v-if="message.status === 'pending' && !message.content" class="flex items-center gap-2 text-xs text-slate-400">
      <span class="inline-block h-2 w-2 animate-pulse rounded-full bg-indigo-400"></span>
      Thinking…
    </div>
    <div
      v-else
      class="prose prose-sm prose-slate max-w-none break-words dark:prose-invert"
      v-html="renderedContent"
    ></div>
    <div v-if="message.status === 'error'" class="mt-2 text-xs text-red-500">Generation failed</div>
    <div v-if="message.status === 'cancelled'" class="mt-2 text-xs text-amber-500">Stopped</div>

    <!-- Copy button -->
    <button
      class="absolute right-2 top-2 hidden rounded-md p-1.5 text-slate-300 transition hover:bg-slate-100 hover:text-slate-600 group-hover:block dark:text-slate-600 dark:hover:bg-white/10 dark:hover:text-slate-300"
      title="Copy"
      type="button"
      @click="$emit('copy', message.content)"
    >
      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
    </button>
  </article>

  <!-- Tool message -->
  <article
    v-else-if="message.role === 'tool'"
    class="max-w-[90%] rounded-lg border border-slate-200 bg-gradient-to-r from-slate-50 to-white px-4 py-3 shadow-sm dark:border-white/10 dark:from-slate-900/50 dark:to-slate-900"
  >
    <div class="flex items-center gap-2">
      <span class="grid h-6 w-6 place-items-center rounded-md bg-amber-100 text-xs dark:bg-amber-900/30">🔧</span>
      <span class="text-xs font-bold text-slate-700 dark:text-slate-300">{{ toolData.name || 'Tool call' }}</span>
      <span
        class="rounded-full px-2 py-0.5 text-[10px] font-semibold"
        :class="toolStatus.className"
      >{{ toolStatus.label }}</span>
    </div>
    <div v-if="message.status === 'pending' && !toolData.decision" class="mt-3 flex items-center gap-2">
      <button class="rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-bold text-white hover:bg-emerald-700" type="button" @click="$emit('tool-decision', message.tool_call_id || '', true)">Allow</button>
      <button class="rounded-md bg-red-600 px-3 py-1.5 text-xs font-bold text-white hover:bg-red-700" type="button" @click="$emit('tool-decision', message.tool_call_id || '', false)">Reject</button>
      <span class="text-xs text-slate-500">Review the arguments before allowing this tool.</span>
    </div>
    <details v-if="toolData.args" class="mt-2">
      <summary class="cursor-pointer text-xs font-medium text-slate-500 dark:text-slate-400">Arguments</summary>
      <pre class="mt-1 max-h-32 overflow-auto rounded-md bg-slate-100 p-2 text-xs dark:bg-slate-800">{{ formatJson(toolData.args) }}</pre>
    </details>
    <details v-if="message.status !== 'pending'" class="mt-2" open>
      <summary class="cursor-pointer text-xs font-medium text-slate-500 dark:text-slate-400">Result</summary>
      <pre class="mt-1 max-h-48 overflow-auto rounded-md bg-slate-100 p-2 text-xs dark:bg-slate-800">{{ formatJson(toolData.result) }}</pre>
    </details>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AIMessage } from '@browser-server/shared-types'
import { renderMarkdown } from './markdown'

const props = defineProps<{
  message: AIMessage
}>()

defineEmits<{
  copy: [content: string]
  toolDecision: [callId: string, approved: boolean]
}>()

const renderedContent = computed(() => renderMarkdown(props.message.content))

interface ToolData {
  name: string
  args: unknown
  result: unknown
  decision: 'approved' | 'rejected' | null
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

const toolStatus = computed(() => {
  if (props.message.status === 'pending' && !toolData.value.decision) {
    return { label: 'approval required', className: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300' }
  }
  if (props.message.status === 'pending' || toolData.value.decision === 'approved') {
    return { label: 'running', className: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300' }
  }
  if (props.message.status === 'error') {
    const result = toolData.value.result as { error?: string } | null
    return { label: result?.error === 'rejected by user' ? 'rejected' : 'failed', className: 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400' }
  }
  return { label: 'success', className: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400' }
})

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
