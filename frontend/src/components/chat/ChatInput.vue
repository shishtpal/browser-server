<template>
  <div class="border-t border-slate-200 bg-white/80 px-4 py-3 backdrop-blur-sm dark:border-white/10 dark:bg-slate-950/80">
    <form class="mx-auto w-full max-w-4xl" @submit.prevent="submit">
      <!-- ── Prompt-applied indicator ── -->
      <Transition name="toast">
        <div
          v-if="appliedVisible"
          class="mb-2 flex items-center gap-2 rounded-lg bg-emerald-50 px-3 py-2 dark:bg-emerald-500/10"
        >
          <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-[0.65rem] font-bold text-white">✓</span>
          <span class="text-[0.8rem] font-medium text-emerald-700 dark:text-emerald-300">Prompt applied</span>
          <span class="text-[0.8rem] text-emerald-500 dark:text-emerald-400">— press Enter to send</span>
        </div>
      </Transition>

      <div class="relative">
        <!-- ── Prompt search dropdown (above input) ── -->
        <PromptSearchDropdown
          v-if="promptMode"
          ref="dropdownRef"
          :results="promptResults"
          :loading="promptLoading"
          :query="promptQuery"
          @select="onPromptSelect"
        />

        <!-- ── Input row ── -->
        <div
          class="flex items-end rounded-xl border transition-all duration-200"
          :class="[
            promptMode
              ? 'border-violet-300 bg-violet-50/50 shadow-lg shadow-violet-500/10 dark:border-violet-500/30 dark:bg-violet-950/20 dark:shadow-violet-500/5'
              : 'border-slate-200 bg-white shadow-sm focus-within:border-indigo-400 focus-within:ring-2 focus-within:ring-indigo-500/15 hover:shadow-md dark:border-white/10 dark:bg-slate-900',
            disabled && 'pointer-events-none opacity-60',
          ]"
        >
          <!-- Prompt-mode badge -->
          <div v-if="promptMode" class="flex items-center gap-1.5 pl-3 pt-2.5">
            <span class="flex h-6 w-6 items-center justify-center rounded-md bg-violet-100 text-xs dark:bg-violet-500/20">🔍</span>
          </div>

          <textarea
            ref="textareaRef"
            v-model="localValue"
            class="max-h-48 min-h-[46px] w-full flex-1 resize-none bg-transparent py-3 pl-3.5 pr-2 text-[0.92em] leading-relaxed outline-none placeholder:text-slate-400 dark:placeholder:text-slate-500"
            :class="promptMode ? 'pl-2' : ''"
            :disabled="disabled"
            :placeholder="promptMode ? 'Search prompts…' : 'Message the assistant…'"
            rows="1"
            @input="onInput"
            @keydown="onKeydown"
          />

          <!-- Action buttons -->
          <div class="flex items-center gap-1.5 px-2.5 pb-2">
            <button
              v-if="busy && canAppend"
              class="rounded-lg bg-indigo-600 px-3 py-1.5 text-[0.8em] font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
              :disabled="!canSubmit || isAppending"
              type="submit"
            >
              {{ isAppending ? 'Appending…' : 'Append context' }}
            </button>
            <button
              v-if="busy"
              class="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-[0.8em] font-semibold text-slate-600 transition hover:bg-slate-50 dark:border-white/10 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
              type="button"
              @click="$emit('stop')"
            >
              Stop
            </button>
            <button
              v-if="!busy"
              class="grid h-8 w-8 place-items-center rounded-lg transition-all duration-200"
              :class="
                canSubmit
                  ? 'bg-indigo-600 text-white shadow-sm hover:bg-indigo-700'
                  : 'bg-slate-100 text-slate-400 dark:bg-slate-800 dark:text-slate-600'
              "
              :disabled="!canSubmit"
              type="submit"
              title="Send message"
            >
              <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5" />
              </svg>
            </button>
          </div>
        </div>
      </div>

      <!-- ── Keyboard hints ── -->
      <div class="mt-2 flex items-center justify-center gap-3 text-[0.7rem] text-slate-400 dark:text-slate-500">
        <span>
          <kbd class="rounded border border-slate-200 px-1.5 py-0.5 font-mono text-[0.66rem] dark:border-white/10">↵</kbd> send
        </span>
        <span class="text-slate-300 dark:text-slate-600">·</span>
        <span>
          <kbd class="rounded border border-slate-200 px-1.5 py-0.5 font-mono text-[0.66rem] dark:border-white/10">⇧↵</kbd> new line
        </span>
        <span class="text-slate-300 dark:text-slate-600">·</span>
        <span>
          <kbd class="rounded border border-slate-200 px-1.5 py-0.5 font-mono text-[0.66rem] dark:border-white/10">/</kbd> prompts
        </span>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import PromptSearchDropdown from '../prompts/PromptSearchDropdown.vue'
import { searchPrompts } from '../../lib/api'
import type { PromptResponse } from '../../types'

const props = defineProps<{
  modelValue: string
  disabled: boolean
  busy: boolean
  canAppend: boolean
  isAppending: boolean
  userId?: number | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  send: [content: string]
  append: [content: string]
  stop: []
  selectPrompt: [prompt: PromptResponse]
}>()

/* ───── refs ───── */
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const dropdownRef = ref<InstanceType<typeof PromptSearchDropdown> | null>(null)

function normalizeToString(v: any): string {
  if (typeof v === 'string') return v
  if (v == null) return ''
  const candidate = (v as any).content ?? (v as any).Content ?? (v as any).Prompt?.content ?? (v as any).prompt?.content
  if (typeof candidate === 'string') return candidate
  try {
    return JSON.stringify(v)
  } catch {
    return String(v)
  }
}

const localValue = ref<string>(normalizeToString(props.modelValue))

watch(() => props.modelValue, (v) => {
  const newVal = normalizeToString(v)
  if (newVal !== localValue.value) {
    localValue.value = newVal
    // modelValue can be set programmatically (e.g. inserting a prompt from the
    // manager); the @input handler never fires in that case, so resize manually.
    nextTick(() => autoResize())
  }
})

const canSubmit = computed(() =>
  !props.disabled
  && localValue.value.trim().length > 0
  && (!props.busy || (props.canAppend && !props.isAppending)),
)

/* ───── prompt-mode state ───── */
const promptMode = ref(false)
const promptQuery = ref('')
const promptResults = ref<PromptResponse[]>([])
const promptLoading = ref(false)
let promptDebounce: ReturnType<typeof setTimeout> | null = null

/* ───── applied indicator ───── */
const appliedVisible = ref(false)
let appliedTimer: ReturnType<typeof setTimeout> | null = null

function showAppliedIndicator() {
  appliedVisible.value = true
  if (appliedTimer) clearTimeout(appliedTimer)
  appliedTimer = setTimeout(() => { appliedVisible.value = false }, 2500)
}

/* ───── auto-resize ───── */
function autoResize() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 192) + 'px'
}

/* ───── input handler ───── */
function onInput() {
  autoResize()

  emit('update:modelValue', localValue.value)

  const value = localValue.value

  /* Already in prompt mode */
  if (promptMode.value) {
    if (!value.startsWith('/')) {
      // User backspaced over the "/" — exit prompt mode, keep remaining text
      exitPromptMode()
      return
    }
    const query = value.slice(1)
    promptQuery.value = query
    if (!query.trim()) {
      promptResults.value = []
      promptLoading.value = false
      return
    }
    debouncedSearch(query)
    return
  }

  /* Not in prompt mode — check if user just typed "/" */
  if (value.startsWith('/')) {
    enterPromptMode(value.slice(1))
  }
}

/* ───── prompt-mode helpers ───── */
function enterPromptMode(query = '') {
  promptMode.value = true
  promptQuery.value = query
  localValue.value = '/' + query
  emit('update:modelValue', localValue.value)
  if (query.trim()) {
    runPromptSearch(query)
  } else {
    promptResults.value = []
    promptLoading.value = false
  }
}

/**
 * Exit prompt mode but **preserve** whatever is currently in the input.
 * Used when the user backspaces over the "/" or when a prompt is selected.
 */
function exitPromptMode() {
  promptMode.value = false
  promptQuery.value = ''
  promptResults.value = []
  promptLoading.value = false
  if (promptDebounce) {
    clearTimeout(promptDebounce)
    promptDebounce = null
  }
}

/**
 * Exit prompt mode **and** clear the input (e.g. Escape).
 */
function clearPromptMode() {
  exitPromptMode()
  localValue.value = ''
  emit('update:modelValue', '')
}

function debouncedSearch(query: string) {
  if (promptDebounce) clearTimeout(promptDebounce)
  promptDebounce = setTimeout(() => runPromptSearch(query), 180)
}

function runPromptSearch(query: string) {
  if (!props.userId || props.userId <= 0) return
  const q = query.trim()
  if (!q) { promptResults.value = []; promptLoading.value = false; return }

  promptLoading.value = true
  searchPrompts(props.userId, q, 20)
    .then((r) => { promptResults.value = r })
    .catch(() => { promptResults.value = [] })
    .finally(() => { promptLoading.value = false })
}

/* ───── prompt selection ───── */
function onPromptSelect(prompt: PromptResponse) {
  // 1. Exit prompt mode WITHOUT clearing the input
  exitPromptMode()

  // 2. Fill the input with the prompt content
  // Some responses may contain the prompt content as an object (older shapes
  // or nested payloads). Normalize to a string so the textarea doesn't get
  // `[object Object]` inserted.
  let contentValue: string
  const raw: any = prompt as any
  const candidate = raw.content ?? raw.Content ?? raw.Prompt?.content ?? raw.prompt?.content
  if (typeof candidate === 'string') {
    contentValue = candidate
  } else if (candidate == null) {
    // Fallback to an empty string if nothing available
    contentValue = ''
  } else {
    try {
      contentValue = JSON.stringify(candidate)
    } catch {
      contentValue = String(candidate)
    }
  }

  localValue.value = contentValue
  emit('update:modelValue', contentValue)
  emit('selectPrompt', prompt)

  // 3. Show the "applied" indicator
  showAppliedIndicator()

  // 4. Refocus & resize
  nextTick(() => {
    textareaRef.value?.focus()
    autoResize()
  })
}

/* ───── keyboard ───── */
function onKeydown(event: KeyboardEvent) {
  /* ── Prompt-mode shortcuts ── */
  if (promptMode.value) {
    if (event.key === 'ArrowDown') { event.preventDefault(); dropdownRef.value?.move(1); return }
    if (event.key === 'ArrowUp')   { event.preventDefault(); dropdownRef.value?.move(-1); return }
    if (event.key === 'Enter')     { event.preventDefault(); dropdownRef.value?.activate(); return }
    if (event.key === 'Escape')    { event.preventDefault(); clearPromptMode(); return }
    return
  }

  /* ── Normal-mode shortcuts ── */
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    submit()
  }
}

/* ───── submit ───── */
function submit() {
  if (!canSubmit.value) return
  // In prompt mode, Enter selects from the dropdown — don't send
  if (promptMode.value) return

  const content = localValue.value.trim()
  if (props.busy) {
    emit('append', content)
    return
  }
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

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.25s ease;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
