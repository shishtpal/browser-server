<template>
  <div
    class="border-t border-slate-200 bg-white/80 px-3 py-3 backdrop-blur-sm sm:px-4 dark:border-white/10 dark:bg-slate-950/80"
  >
    <form class="mx-auto w-full max-w-4xl" @submit.prevent="submit">
      <!-- Prompt-applied indicator -->
      <Transition name="toast">
        <div
          v-if="appliedVisible"
          class="mb-2 flex items-center gap-2 rounded-lg bg-emerald-50 px-3 py-2 dark:bg-emerald-500/10"
          role="status"
        >
          <CircleCheck class="h-4 w-4 shrink-0 text-emerald-600" aria-hidden="true" />
          <span class="text-[0.8rem] font-medium text-emerald-700 dark:text-emerald-300">
            Prompt applied
          </span>
          <span class="text-[0.8rem] text-emerald-500 dark:text-emerald-400">
            — press Enter to send
          </span>
        </div>
      </Transition>

      <div class="relative">
        <!-- Prompt search dropdown (above input) -->
        <PromptSearchDropdown
          v-if="promptMode"
          ref="dropdownRef"
          :results="promptResults"
          :loading="promptLoading"
          :query="promptQuery"
          @select="onPromptSelect"
        />

        <!-- Staged image previews -->
        <StagedImageStrip :items="stagedImages" @remove="$emit('remove-image', $event)" />

        <!-- Input row -->
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
          <div v-if="promptMode" class="flex items-center gap-1.5 pt-2.5 pl-3">
            <span
              class="grid h-6 w-6 place-items-center rounded-md bg-violet-100 text-violet-600 dark:bg-violet-500/20 dark:text-violet-300"
            >
              <Search class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
            </span>
          </div>

          <textarea
            ref="textareaRef"
            v-model="localValue"
            class="max-h-48 min-h-[48px] w-full flex-1 resize-none bg-transparent py-3 pr-2 pl-3.5 text-[0.92em] leading-relaxed outline-none placeholder:text-slate-400 dark:placeholder:text-slate-500"
            :class="promptMode ? 'pl-2' : ''"
            :disabled="disabled"
            :placeholder="promptMode ? 'Search prompts…' : 'Message the assistant…'"
            rows="1"
            @input="onInput"
            @keydown="onKeydown"
            @paste="onPaste"
          />

          <!-- Action buttons -->
          <div class="flex items-center gap-1 px-2 pb-2 sm:gap-1.5 sm:px-2.5">
            <input
              ref="fileInputRef"
              type="file"
              :accept="ALLOWED_IMAGE_TYPES.join(',')"
              multiple
              class="hidden"
              @change="onFileSelected"
            />
            <button
              class="grid h-9 w-9 place-items-center rounded-lg text-slate-500 transition hover:bg-slate-100 hover:text-indigo-600 disabled:cursor-not-allowed disabled:opacity-40 sm:h-8 sm:w-8 dark:text-slate-400 dark:hover:bg-white/10"
              :disabled="!attachmentsEnabled || disabled"
              type="button"
              aria-label="Attach images"
              :title="attachmentsEnabled ? 'Attach images' : attachmentsDisabledReason"
              @click="fileInputRef?.click()"
            >
              <ImagePlus class="h-4 w-4" :stroke-width="2" aria-hidden="true" />
            </button>
            <button
              class="grid h-9 w-9 place-items-center rounded-lg text-slate-500 transition hover:bg-slate-100 hover:text-indigo-600 disabled:cursor-not-allowed disabled:opacity-40 sm:h-8 sm:w-8 dark:text-slate-400 dark:hover:bg-white/10"
              :disabled="disabled"
              type="button"
              aria-label="Voice typing"
              :title="
                disabled ? 'Voice typing is unavailable while AI chat is disabled' : 'Voice typing'
              "
              @click="$emit('voice')"
            >
              <Mic class="h-4 w-4" :stroke-width="2" aria-hidden="true" />
            </button>
            <button
              v-if="busy && canAppend"
              class="rounded-lg bg-indigo-600 px-3 py-2 text-[0.8em] font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 sm:py-1.5"
              :disabled="!canSubmit || isAppending"
              type="submit"
            >
              {{ isAppending ? 'Appending…' : 'Append' }}
            </button>
            <button
              v-if="busy"
              class="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-[0.8em] font-semibold text-slate-600 transition hover:bg-slate-50 sm:py-1.5 dark:border-white/10 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
              type="button"
              @click="$emit('stop')"
            >
              <Square class="h-3 w-3 fill-current" aria-hidden="true" />
              Stop
            </button>
            <button
              v-if="!busy"
              class="grid h-9 w-9 place-items-center rounded-lg transition-all duration-200 sm:h-8 sm:w-8"
              :class="
                canSubmit
                  ? 'bg-indigo-600 text-white shadow-sm hover:bg-indigo-700'
                  : 'bg-slate-100 text-slate-400 dark:bg-slate-800 dark:text-slate-600'
              "
              :disabled="!canSubmit"
              type="submit"
              title="Send message"
              aria-label="Send message"
            >
              <SendHorizontal class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>

      <!-- Keyboard hints -->
      <div
        class="mt-2 hidden items-center justify-center gap-3 text-[0.7rem] text-slate-400 min-[420px]:flex dark:text-slate-500"
      >
        <span>
          <kbd
            class="rounded border border-slate-200 px-1.5 py-0.5 font-mono text-[0.66rem] dark:border-white/10"
            >↵</kbd
          >
          send
        </span>
        <span aria-hidden="true" class="text-slate-300 dark:text-slate-600">·</span>
        <span>
          <kbd
            class="rounded border border-slate-200 px-1.5 py-0.5 font-mono text-[0.66rem] dark:border-white/10"
            >⇧↵</kbd
          >
          new line
        </span>
        <span aria-hidden="true" class="text-slate-300 dark:text-slate-600">·</span>
        <span>
          <kbd
            class="rounded border border-slate-200 px-1.5 py-0.5 font-mono text-[0.66rem] dark:border-white/10"
            >/</kbd
          >
          prompts
        </span>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, toRef, watch } from 'vue';
import { CircleCheck, ImagePlus, Mic, Search, SendHorizontal, Square } from '@lucide/vue';
import type { AIChatAttachmentsConfig } from '@browser-server/shared-types';
import type { PromptResponse } from '../../types';
import PromptSearchDropdown from '../prompts/PromptSearchDropdown.vue';
import StagedImageStrip from './input/StagedImageStrip.vue';
import {
  ALLOWED_IMAGE_TYPES,
  isAllowedImage,
  validateImageFiles,
  type StagedImageInput,
} from './input/stagedImages';
import { usePromptMode, type PromptDropdownApi } from './composables/usePromptMode';
import { formatBytes, normalizePromptContent } from './chatFormat';

export type { StagedImageInput } from './input/stagedImages';

const props = defineProps<{
  modelValue: string;
  disabled: boolean;
  busy: boolean;
  canAppend: boolean;
  isAppending: boolean;
  userId?: number | null;
  conversationId?: string;
  attachmentsConfig?: AIChatAttachmentsConfig | null;
  supportsVision?: boolean;
  stagedImages?: StagedImageInput[];
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
  send: [content: string];
  append: [content: string];
  stop: [];
  voice: [];
  'select-prompt': [prompt: PromptResponse];
  'add-images': [files: File[]];
  'remove-image': [id: string];
}>();

/* ------------------------------- textarea ---------------------------------- */

const textareaRef = ref<HTMLTextAreaElement | null>(null);
const dropdownRef = ref<PromptDropdownApi | null>(null);
const fileInputRef = ref<HTMLInputElement | null>(null);

const localValue = ref<string>(normalizePromptContent(props.modelValue));

watch(
  () => props.modelValue,
  (v) => {
    const newVal = normalizePromptContent(v);
    if (newVal !== localValue.value) {
      localValue.value = newVal;
      // Programmatic sets (prompt manager) never fire @input — resize manually.
      nextTick(() => autoResize());
    }
  },
);

function setModelValue(v: string) {
  localValue.value = v;
  emit('update:modelValue', v);
}

function autoResize() {
  const el = textareaRef.value;
  if (!el) return;
  el.style.height = 'auto';
  el.style.height = Math.min(el.scrollHeight, 192) + 'px';
}

/* ------------------------------ attachments -------------------------------- */

const attachmentsEnabled = computed(() => {
  if (props.disabled) return false;
  // Disable while a message is in flight: covers normal send + append window,
  // so attachment ordering never collides with a tool continuation.
  if (props.busy) return false;
  const cfg = props.attachmentsConfig;
  if (!cfg?.enabled || !props.supportsVision) return false;
  return true;
});

const attachmentsDisabledReason = computed(() => {
  if (props.disabled) return 'Chat is disabled';
  if (!props.attachmentsConfig?.enabled) return 'Image attachments are disabled on this server';
  if (!props.supportsVision) return 'The selected model does not support image attachments';
  return 'Image attachments are unavailable';
});

const stagedImages = computed(() => props.stagedImages ?? []);

function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement;
  const files = Array.from(input.files || []);
  input.value = '';
  if (!files.length || !attachmentsEnabled.value) return;
  const { valid, rejected } = validateImageFiles(files, props.attachmentsConfig, formatBytes);
  if (valid.length > 0) emit('add-images', valid);
  if (rejected.length > 0) {
    console.warn('[ChatInput] rejected attachments:', rejected);
  }
}

function onPaste(event: ClipboardEvent) {
  if (!attachmentsEnabled.value) return;
  const items = event.clipboardData?.items;
  if (!items) return;
  const files: File[] = [];
  for (const item of Array.from(items)) {
    if (item.kind === 'file') {
      const file = item.getAsFile();
      if (file && isAllowedImage(file)) files.push(file);
    }
  }
  if (files.length === 0) return;
  const { valid, rejected } = validateImageFiles(files, props.attachmentsConfig, formatBytes);
  if (valid.length > 0) {
    event.preventDefault();
    emit('add-images', valid);
  }
  if (rejected.length > 0) {
    console.warn('[ChatInput] rejected pasted attachments:', rejected);
  }
}

/* ------------------------------- prompt mode ------------------------------- */

const prompt = usePromptMode(toRef(props, 'userId'), setModelValue, () => localValue.value);
const { promptMode, promptQuery, promptResults, promptLoading } = prompt;

const appliedVisible = ref(false);
let appliedTimer: ReturnType<typeof setTimeout> | null = null;

function showAppliedIndicator() {
  appliedVisible.value = true;
  if (appliedTimer) clearTimeout(appliedTimer);
  appliedTimer = setTimeout(() => {
    appliedVisible.value = false;
  }, 2500);
}

function onPromptSelect(selected: PromptResponse) {
  prompt.exitPromptMode();
  setModelValue(normalizePromptContent(selected));
  emit('select-prompt', selected);
  showAppliedIndicator();
  nextTick(() => {
    textareaRef.value?.focus();
    autoResize();
  });
}

/* --------------------------------- submit ---------------------------------- */

const canSubmit = computed(
  () =>
    !props.disabled &&
    (localValue.value.trim().length > 0 ||
      stagedImages.value.some((i) => !i.uploading && !i.error)) &&
    (!props.busy || (props.canAppend && !props.isAppending)),
);

function onInput() {
  autoResize();
  emit('update:modelValue', localValue.value);
  prompt.onTextChanged();
}

function onKeydown(event: KeyboardEvent) {
  if (prompt.onKeydown(event, dropdownRef.value)) return;
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault();
    submit();
  }
}

function submit() {
  if (!canSubmit.value || promptMode.value) return;
  const content = localValue.value.trim();
  if (props.busy) {
    emit('append', content);
    return;
  }
  setModelValue('');
  emit('send', content);
  if (textareaRef.value) textareaRef.value.style.height = 'auto';
}

function focus() {
  textareaRef.value?.focus();
}

defineExpose({ focus });
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
