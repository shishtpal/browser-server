<template>
  <div
    ref="container"
    class="chat-scroll min-h-0 flex-1 overflow-y-auto bg-slate-50/40 dark:bg-slate-950"
    @scroll="handleScroll"
  >
    <div class="mx-auto w-full space-y-5 px-4 py-6 lg:px-6">
      <!-- Empty state -->
      <div
        v-if="messages.length === 0 && !loading"
        class="flex min-h-[56vh] flex-col items-center justify-center text-center"
      >
        <div
          class="mb-5 grid h-14 w-14 place-items-center rounded-2xl bg-gradient-to-br from-indigo-500 to-violet-600 shadow-lg shadow-indigo-500/25"
        >
          <svg class="h-7 w-7 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="1.5"
              d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z"
            />
          </svg>
        </div>
        <h2 class="text-lg font-bold tracking-tight">Start a conversation</h2>
        <p class="mt-1.5 max-w-sm text-[0.85em] text-slate-500 dark:text-slate-400">
          Choose a model and send a message, or try one of these:
        </p>
        <div class="mt-5 grid w-full max-w-lg gap-2 sm:grid-cols-2">
          <button
            v-for="suggestion in suggestions"
            :key="suggestion"
            class="rounded-lg border border-slate-200 bg-white px-3.5 py-2.5 text-left text-[0.82em] font-medium text-slate-700 shadow-sm transition hover:-translate-y-0.5 hover:border-indigo-300 hover:text-indigo-700 hover:shadow-md dark:border-white/10 dark:bg-slate-900 dark:text-slate-300 dark:hover:border-indigo-500/40 dark:hover:text-indigo-300"
            type="button"
            @click="$emit('suggestion', suggestion)"
          >
            {{ suggestion }}
          </button>
        </div>
      </div>

      <!-- Messages -->
      <template v-for="message in messages" :key="message.id">
        <ChatBubble
          :message="message"
          :show-thinking="showThinking"
          @copy="$emit('copy', $event)"
          @delete="$emit('delete', $event)"
          @branch="$emit('branch', $event)"
          @tool-decision="
            (callId, approved, comment) => $emit('tool-decision', callId, approved, comment)
          "
        />
      </template>

      <!-- Typing indicator -->
      <div
        v-if="loading && messages.length > 0"
        class="flex items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 shadow-sm dark:border-white/10 dark:bg-slate-900"
        style="width: fit-content"
      >
        <span
          class="h-1.5 w-1.5 animate-bounce rounded-full bg-indigo-400 [animation-delay:0ms]"
        ></span>
        <span
          class="h-1.5 w-1.5 animate-bounce rounded-full bg-indigo-400 [animation-delay:150ms]"
        ></span>
        <span
          class="h-1.5 w-1.5 animate-bounce rounded-full bg-indigo-400 [animation-delay:300ms]"
        ></span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { AIMessage } from '@browser-server/shared-types'
import ChatBubble from './ChatBubble.vue'

const props = withDefaults(
  defineProps<{
    messages: AIMessage[]
    loading: boolean
    showThinking?: boolean
  }>(),
  { showThinking: true },
)

defineEmits<{
  suggestion: [text: string]
  copy: [content: string]
  delete: [messageId: string]
  branch: [messageId: string]
  'tool-decision': [callId: string, approved: boolean, comment: string]
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

watch(
  () => props.messages,
  () => {
    if (!userScrolledUp.value) nextTick(scrollToBottom)
  },
  { deep: true },
)

watch(
  () => props.loading,
  (val) => {
    if (val && !userScrolledUp.value) nextTick(scrollToBottom)
  },
)

defineExpose({ scrollToBottom })
</script>
