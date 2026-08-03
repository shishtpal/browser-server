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

        <!-- ── Staged image previews ── -->
        <div
          v-if="stagedImages.length > 0"
          class="mb-2 flex flex-wrap gap-2"
        >
          <div
            v-for="item in stagedImages"
            :key="item.id"
            class="group relative flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-2 py-1.5 pr-8 dark:border-white/10 dark:bg-slate-900"
          >
            <img
              :src="item.previewUrl"
              alt=""
              class="h-10 w-10 rounded-md object-cover"
            />
            <div class="flex min-w-0 flex-col">
              <span class="max-w-[10rem] truncate text-[0.72rem] font-medium text-slate-700 dark:text-slate-200">{{ item.file.name }}</span>
              <span class="text-[0.65rem] text-slate-500 dark:text-slate-400">{{ formatBytes(item.file.size) }}</span>
            </div>
            <span
              v-if="item.uploading"
              class="absolute right-1.5 top-1/2 -translate-y-1/2"
            >
              <span class="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-slate-300 border-t-indigo-600" />
            </span>
            <button
              v-else
              type="button"
              class="absolute right-1 top-1/2 -translate-y-1/2 rounded p-1 text-slate-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-500/10 dark:hover:text-red-400"
              title="Remove image"
              aria-label="Remove image"
              @click="removeImage(item.id)"
            >
              <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
            <div
              v-if="item.error"
              class="absolute -bottom-1 left-0 right-0 translate-y-full rounded bg-red-100 px-1.5 py-0.5 text-[0.65rem] text-red-700 dark:bg-red-950/40 dark:text-red-200"
            >
              {{ item.error }}
            </div>
          </div>
        </div>

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
            @paste="onPaste"
          />

          <!-- Action buttons -->
          <div class="flex items-center gap-1.5 px-2.5 pb-2">
            <input
              ref="fileInputRef"
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif"
              multiple
              class="hidden"
              @change="onFileSelected"
            />
            <button
              class="grid h-8 w-8 place-items-center rounded-lg text-slate-500 transition hover:bg-slate-100 hover:text-indigo-600 disabled:cursor-not-allowed disabled:opacity-40 dark:text-slate-400 dark:hover:bg-white/10"
              :disabled="!attachmentsEnabled || disabled"
              type="button"
              aria-label="Attach images"
              :title="attachmentsEnabled ? 'Attach images' : attachmentsDisabledReason"
              @click="fileInputRef?.click()"
            >
              <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M15.172 7l-6.586 6.586a2 2 0 002.828 2.828l6.586-6.586A4 4 0 0010.172 3H6a4 4 0 00-4 4v10a4 4 0 004 4h10a4 4 0 004-4v-6" />
              </svg>
            </button>
            <button
              class="grid h-8 w-8 place-items-center rounded-lg text-slate-500 transition hover:bg-slate-100 hover:text-indigo-600 disabled:cursor-not-allowed disabled:opacity-40 dark:text-slate-400 dark:hover:bg-white/10"
              :disabled="disabled"
              type="button"
              aria-label="Voice typing"
              :title="disabled ? 'Voice typing is unavailable while AI chat is disabled' : 'Voice typing'"
              @click="$emit('voice')"
            >
              <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 2a3 3 0 00-3 3v7a3 3 0 006 0V5a3 3 0 00-3-3zM5 10v2a7 7 0 0014 0v-2M12 19v3m-4 0h8" />
              </svg>
            </button>
            <button
              v-if="busy && canAppend"
              class="rounded-lg bg-indigo-600 px-3 py-1.5 text-[0.8em] font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
              :disabled="!canSubmit || isAppending"
              type="submit"
            >
              {{ isAppending ? 'Appending…' : 'Append' }}
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
import type { AIChatAttachmentsConfig } from '@browser-server/shared-types'

export interface StagedImageInput {
  id: string
  file: File
  previewUrl: string
  uploading: boolean
  error?: string
}

const props = defineProps<{
  modelValue: string
  disabled: boolean
  busy: boolean
  canAppend: boolean
  isAppending: boolean
  userId?: number | null
  conversationId?: string
  attachmentsConfig?: AIChatAttachmentsConfig | null
  supportsVision?: boolean
  stagedImages?: StagedImageInput[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  send: [content: string]
  append: [content: string]
  stop: []
  voice: []
  selectPrompt: [prompt: PromptResponse]
  addImages: [files: File[]]
  removeImage: [id: string]
}>()

/* ───── refs ───── */
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const dropdownRef = ref<InstanceType<typeof PromptSearchDropdown> | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

const allowedMimeTypes = ['image/png', 'image/jpeg', 'image/webp', 'image/gif']

const attachmentsEnabled = computed(() => {
  if (props.disabled) return false
  // Disable while a message is in flight: that covers a normal send and an
  // open append window, so attachment ordering never collides with a tool
  // continuation (per the product spec, append stays text-only).
  if (props.busy) return false
  const cfg = props.attachmentsConfig
  if (!cfg?.enabled || !props.supportsVision) return false
  return true
})

const attachmentsDisabledReason = computed(() => {
  if (props.disabled) return 'Chat is disabled'
  if (!props.attachmentsConfig?.enabled) return 'Image attachments are disabled on this server'
  if (!props.supportsVision) return 'The selected model does not support image attachments'
  return 'Image attachments are unavailable'
})

const stagedImages = computed(() => props.stagedImages ?? [])

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function isAllowedImage(file: File): boolean {
  return allowedMimeTypes.includes(file.type)
}

function validateFiles(files: File[]): { valid: File[]; rejected: string[] } {
  const cfg = props.attachmentsConfig
  const valid: File[] = []
  const rejected: string[] = []
  const maxBytes = cfg?.max_image_bytes ?? 5 * 1024 * 1024
  const maxCount = cfg?.max_images ?? 5
  for (const file of files) {
    if (!isAllowedImage(file)) {
      rejected.push(`${file.name || 'file'} is not a supported image`)
      continue
    }
    if (file.size > maxBytes) {
      rejected.push(`${file.name || 'file'} exceeds the ${formatBytes(maxBytes)} image limit`)
      continue
    }
    if (valid.length >= maxCount) {
      rejected.push(`Only ${maxCount} images are allowed per message`)
      break
    }
    valid.push(file)
  }
  return { valid, rejected }
}

function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (!files.length) return
  if (!attachmentsEnabled.value) return
  const { valid, rejected } = validateFiles(files)
  if (valid.length > 0) emit('addImages', valid)
  if (rejected.length > 0) {
    // Surface first rejection to the user; the rest are logged only.
    console.warn('[ChatInput] rejected attachments:', rejected)
  }
}

function onPaste(event: ClipboardEvent) {
  if (!attachmentsEnabled.value) return
  const items = event.clipboardData?.items
  if (!items) return
  const files: File[] = []
  for (const item of Array.from(items)) {
    if (item.kind === 'file') {
      const file = item.getAsFile()
      if (file && isAllowedImage(file)) files.push(file)
    }
  }
  if (files.length === 0) return
  const { valid, rejected } = validateFiles(files)
  if (valid.length > 0) {
    event.preventDefault()
    emit('addImages', valid)
  }
  if (rejected.length > 0) {
    console.warn('[ChatInput] rejected pasted attachments:', rejected)
  }
}

function removeImage(id: string) {
  emit('removeImage', id)
}

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
  && (localValue.value.trim().length > 0 || stagedImages.value.some((i) => !i.uploading && !i.error))
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
