<template>
  <div class="border-t border-slate-200 bg-white/80 px-4 py-3 backdrop-blur-sm dark:border-white/10 dark:bg-slate-950/80">
    <form class="mx-auto w-full lg:px-4" @submit.prevent="submit">
      <div class="relative">
        <textarea
          ref="textareaRef"
          v-model="localValue"
          class="max-h-48 min-h-[52px] w-full resize-none rounded-xl border border-slate-200 bg-white py-3 pl-4 pr-24 text-[1em] leading-relaxed outline-none transition-colors placeholder:text-slate-400 focus:border-slate-400 dark:border-white/10 dark:bg-slate-900 dark:placeholder:text-slate-500 dark:focus:border-white/20"
          :disabled="disabled"
          :placeholder="promptMode ? 'Search prompts…' : 'Message the assistant…'"
          rows="1"
          @input="onInput"
          @keydown="onKeydown"
          @blur="onBlur"
        ></textarea>
        <PromptSearchDropdown
          v-if="promptMode"
          ref="dropdownRef"
          :visible="promptMode"
          :results="promptResults"
          :loading="promptLoading"
          @select="onPromptSelect"
          @close="closePromptMode"
        />
        <div class="absolute bottom-2 right-2 flex items-center gap-1.5">
          <button
            v-if="busy"
            class="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-[0.85em] font-bold text-slate-700 transition hover:bg-slate-100 dark:border-white/10 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
            type="button"
            @click="$emit('stop')"
          >
            Stop
          </button>
          <button
            class="grid h-8 w-8 place-items-center rounded-lg bg-slate-900 text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-40 dark:bg-white dark:text-slate-900 dark:hover:bg-gray-100"
            :disabled="!canSend"
            type="submit"
            title="Send message"
          >
            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19V5m0 0l-7 7m7-7l7 7" />
            </svg>
          </button>
        </div>
      </div>
      <p class="mt-2 text-center text-[0.8em] text-slate-400 dark:text-slate-500">
        <kbd class="rounded border border-slate-200 px-1 py-0.5 text-[0.7em] dark:border-white/10">Enter</kbd> to send ·
        <kbd class="rounded border border-slate-200 px-1 py-0.5 text-[0.7em] dark:border-white/10">Shift+Enter</kbd> for new line ·
        <kbd class="rounded border border-slate-200 px-1 py-0.5 text-[0.7em] dark:border-white/10">/</kbd> to search prompts
      </p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import PromptSearchDropdown from '../prompts/PromptSearchDropdown.vue'
import { searchPrompts } from '../../lib/api'
import type { PromptResponse } from '../types'

const props = defineProps<{
  modelValue: string
  disabled: boolean
  busy: boolean
  userId?: number | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  send: [content: string]
  stop: []
  selectPrompt: [prompt: PromptResponse]
}>()

const textareaRef = ref<HTMLTextAreaElement | null>(null)
const dropdownRef = ref<InstanceType<typeof PromptSearchDropdown> | null>(null)
const localValue = ref(props.modelValue)

watch(() => props.modelValue, (v) => {
  if (v !== localValue.value) localValue.value = v
})

const canSend = computed(() => !props.disabled && !props.busy && localValue.value.trim().length > 0)

const promptMode = ref(false)
const promptQuery = ref('')
const promptResults = ref<PromptResponse[]>([])
const promptLoading = ref(false)
let promptDebounce: ReturnType<typeof setTimeout> | null = null

function onInput() {
  const el = textareaRef.value
  if (el) {
    el.style.height = 'auto'
    el.style.height = Math.min(el.scrollHeight, 192) + 'px'
  }

  const value = localValue.value
  if (!promptMode.value) {
    if (value.startsWith('/')) {
      enterPromptMode(value.slice(1))
      return
    }
    return
  }

  const query = value.startsWith('/') ? value.slice(1) : value
  promptQuery.value = query
  if (!query.trim()) {
    promptResults.value = []
    promptLoading.value = false
    return
  }

  if (promptDebounce) clearTimeout(promptDebounce)
  promptDebounce = setTimeout(() => runPromptSearch(query), 180)
}

function enterPromptMode(query = '') {
  promptMode.value = true
  promptQuery.value = query
  localValue.value = '/' + query
  if (query.trim()) {
    runPromptSearch(query)
  } else {
    promptResults.value = []
    promptLoading.value = false
  }
}

function runPromptSearch(query: string) {
  if (!props.userId || props.userId <= 0) return
  const trimmedQuery = query.trim()
  if (!trimmedQuery) {
    promptResults.value = []
    promptLoading.value = false
    return
  }

  promptLoading.value = true
  searchPrompts(props.userId, trimmedQuery, 20)
    .then((results) => {
      promptResults.value = results
    })
    .catch(() => {
      promptResults.value = []
    })
    .finally(() => {
      promptLoading.value = false
    })
}

function closePromptMode() {
  promptMode.value = false
  promptQuery.value = ''
  promptResults.value = []
  localValue.value = ''
  emit('update:modelValue', '')
}

function onPromptSelect(prompt: PromptResponse) {
  localValue.value = prompt.content
  emit('update:modelValue', prompt.content)
  emit('selectPrompt', prompt)
  closePromptMode()
  nextTick(() => textareaRef.value?.focus())
}

function onKeydown(event: KeyboardEvent) {
  if (!promptMode.value) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault()
      submit()
      return
    }
    if (event.key === 'Enter' && event.shiftKey) {
      return
    }
    return
  }

  if (event.key === 'ArrowDown') {
    event.preventDefault()
    dropdownRef.value?.move(1)
    return
  }
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    dropdownRef.value?.move(-1)
    return
  }
  if (event.key === 'Enter') {
    event.preventDefault()
    dropdownRef.value?.activate()
    return
  }
  if (event.key === 'Escape') {
    event.preventDefault()
    closePromptMode()
    return
  }
}

function onBlur() {
  // Small delay to allow dropdown click to register
  setTimeout(() => {
    if (promptMode.value) closePromptMode()
  }, 120)
}

function submit() {
  if (!canSend.value) return
  if (promptMode.value) {
    // allow selecting via enter only
    return
  }
  const content = localValue.value.trim()
  localValue.value = ''
  emit('update:modelValue', '')
  emit('send', content)
  if (textareaRef.value) textareaRef.value.style.height = 'auto'
}

function focus() {
  textareaRef.value?.focus()
}

defineExpose({ focus })
</script>
