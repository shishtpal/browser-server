<template>
  <Modal
    :open="open"
    title="Voice typing"
    description="Record, review, and insert a transcript into your message."
    @close="close"
  >
    <div class="space-y-4">
      <div v-if="config?.enabled" class="grid gap-3 sm:grid-cols-3">
        <label class="text-xs font-semibold text-slate-600 dark:text-slate-300">
          Provider
          <select
            v-model="provider"
            :disabled="isActive"
            class="mt-1 w-full rounded-lg border border-slate-200 bg-white px-2 py-2 text-sm dark:border-white/10 dark:bg-slate-900"
          >
            <option v-for="[id] in providers" :key="id" :value="id">{{ id }}</option>
          </select>
        </label>
        <label class="text-xs font-semibold text-slate-600 dark:text-slate-300">
          Model
          <select
            v-model="model"
            :disabled="isActive"
            class="mt-1 w-full rounded-lg border border-slate-200 bg-white px-2 py-2 text-sm dark:border-white/10 dark:bg-slate-900"
          >
            <option v-for="item in models" :key="item.id" :value="item.id">{{ item.label }}</option>
          </select>
        </label>
        <label class="text-xs font-semibold text-slate-600 dark:text-slate-300">
          Language
          <select
            v-model="language"
            :disabled="isActive"
            class="mt-1 w-full rounded-lg border border-slate-200 bg-white px-2 py-2 text-sm dark:border-white/10 dark:bg-slate-900"
          >
            <option v-for="item in config.languages" :key="item.code" :value="item.code">
              {{ item.label }}
            </option>
          </select>
        </label>
      </div>

      <div
        class="rounded-lg bg-slate-50 p-3 text-sm dark:bg-white/5"
        role="status"
        aria-live="polite"
      >
        <span v-if="state === 'requesting'">Requesting microphone permission…</span>
        <span v-else-if="state === 'listening'" class="font-semibold text-red-600"
          >● Listening — {{ elapsedSeconds }}s</span
        >
        <span v-else-if="state === 'transcribing'">Transcribing…</span>
        <span v-else-if="state === 'denied'">Microphone permission is blocked.</span>
        <span v-else-if="state === 'unsupported'">Voice capture is unsupported here.</span>
        <span v-else>Ready to record.</span>
      </div>
      <p
        v-if="error"
        class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-500/10 dark:text-red-300"
      >
        {{ error }}
      </p>

      <label
        v-if="transcript || state === 'idle'"
        class="block text-xs font-semibold text-slate-600 dark:text-slate-300"
      >
        Transcript
        <textarea
          v-model="transcript"
          rows="5"
          placeholder="Your transcript will appear here…"
          class="mt-1 w-full resize-y rounded-lg border border-slate-200 bg-white p-3 text-sm outline-none focus:border-indigo-400 dark:border-white/10 dark:bg-slate-900"
        />
      </label>

      <div class="flex flex-wrap justify-end gap-2">
        <button
          v-if="state === 'listening' || state === 'requesting' || state === 'transcribing'"
          type="button"
          class="rounded-lg bg-red-600 px-4 py-2 text-sm font-bold text-white"
          @click="stop"
        >
          Stop
        </button>
        <button
          v-else-if="transcript"
          type="button"
          class="rounded-lg border border-slate-200 px-4 py-2 text-sm font-bold dark:border-white/10"
          @click="recordAgain"
        >
          Record again
        </button>
        <button
          v-else
          type="button"
          :disabled="!config?.enabled"
          class="rounded-lg border border-slate-200 px-4 py-2 text-sm font-bold disabled:opacity-50 dark:border-white/10"
          @click="start"
        >
          {{ state === 'denied' || state === 'error' ? 'Retry' : 'Start recording' }}
        </button>
        <button
          type="button"
          :disabled="!transcript.trim()"
          class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-bold text-white disabled:opacity-50"
          @click="useText"
        >
          Use text
        </button>
      </div>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { toRef, watch } from 'vue';
import Modal from '../ui/Modal.vue';
import { useVoiceTyping } from './composables/useVoiceTyping';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: []; use: [text: string] }>();
const {
  config,
  state,
  error,
  transcript,
  provider,
  model,
  language,
  elapsedSeconds,
  providers,
  models,
  isActive,
  openSession,
  start,
  stop,
  cleanup,
  recordAgain,
} = useVoiceTyping(toRef(props, 'open'));

watch(
  () => props.open,
  (value) => {
    if (value) void openSession();
    else cleanup();
  },
);

function close() {
  cleanup();
  emit('close');
}
function useText() {
  const text = transcript.value.trim();
  if (!text) return;
  emit('use', text);
  close();
}
</script>
