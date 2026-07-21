<template>
  <div ref="container" class="flex-1 overflow-y-auto" @scroll="handleScroll">
    <div class="mx-auto w-full space-y-6 px-4 py-6 lg:px-8">
      <!-- Empty state -->
      <div v-if="messages.length === 0 && !loading" class="flex min-h-[50vh] flex-col items-center justify-center text-center">
        <div class="mb-6 grid h-16 w-16 place-items-center rounded-2xl bg-gradient-to-br from-indigo-500 to-violet-500 shadow-lg shadow-indigo-500/20">
          <svg class="h-8 w-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z"/></svg>
        </div>
        <h2 class="text-xl font-black">Start a conversation</h2>
        <p class="mt-2 max-w-sm text-[0.9em] text-slate-500 dark:text-slate-400">Choose a model and send a message, or try one of these:</p>
        <div class="mt-4 grid gap-2 sm:grid-cols-2">
          <button
            v-for="suggestion in suggestions"
            :key="suggestion"
            class="rounded-lg border border-slate-200 px-4 py-3 text-left text-[0.9em] transition hover:border-slate-300 hover:bg-slate-50 dark:border-white/10 dark:hover:border-white/20 dark:hover:bg-white/5"
            type="button"
            @click="$emit('suggestion', suggestion)"
          >{{ suggestion }}</button>
        </div>
      </div>

      <!-- Messages -->
      <template v-for="message in messages" :key="message.id">
        <ChatBubble :message="message" @copy="$emit('copy', $event)" @tool-decision="(callId, approved, comment) => $emit('tool-decision', callId, approved, comment)" />
      </template>

      <!-- Typing indicator -->
      <div v-if="loading && messages.length > 0" class="flex items-center gap-1.5 rounded-2xl border border-slate-200 bg-white px-4 py-3 dark:border-white/10 dark:bg-slate-900" style="width: fit-content;">
        <span class="h-2 w-2 animate-bounce rounded-full bg-slate-400 [animation-delay:0ms]"></span>
        <span class="h-2 w-2 animate-bounce rounded-full bg-slate-400 [animation-delay:150ms]"></span>
        <span class="h-2 w-2 animate-bounce rounded-full bg-slate-400 [animation-delay:300ms]"></span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { AIMessage } from '@browser-server/shared-types'
import ChatBubble from './ChatBubble.vue'

const props = defineProps<{
  messages: AIMessage[]
  loading: boolean
}>()

defineEmits<{
  suggestion: [text: string]
  copy: [content: string]
  toolDecision: [callId: string, approved: boolean, comment: string]
}>()

const container = ref<HTMLElement | null>(null)
const userScrolledUp = ref(false)

const suggestions = [
  'Explain how this project works',
  'Help me debug an issue',
  'Write a function that…',
  'Summarize what I should do next',
]

function handleScroll() {
  const el = container.value
  if (!el) return
  const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
  userScrolledUp.value = distFromBottom > 100
}

function scrollToBottom() {
  const el = container.value
  if (el) el.scrollTop = el.scrollHeight
}

watch(() => props.messages, () => {
  if (!userScrolledUp.value) nextTick(scrollToBottom)
}, { deep: true })

watch(() => props.loading, (val) => {
  if (val && !userScrolledUp.value) nextTick(scrollToBottom)
})

defineExpose({ scrollToBottom })
</script>
