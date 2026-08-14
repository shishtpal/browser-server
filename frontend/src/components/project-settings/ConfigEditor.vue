<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import {
  AlertTriangle,
  Maximize2,
  Minimize2,
  Minus,
  Plus,
  Power,
  RefreshCw,
  RotateCcw,
  Save,
} from '@lucide/vue';
import type { AdminConfigFile } from '../../lib/api';
import JsonCodeEditor from './JsonCodeEditor.vue';

const props = defineProps<{
  file: AdminConfigFile | null;
  dirty: boolean;
  loading: boolean;
  saving: boolean;
  reloading: boolean;
  restarting: boolean;
  managed: boolean;
  restartRequired: boolean;
  canRestart: boolean;
}>();

const emit = defineEmits<{
  save: [];
  reload: [];
  restart: [];
}>();

const draft = defineModel<string>({ required: true });
const FONT_SIZE_KEY = 'project-settings-editor-font-size';
const DEFAULT_FONT_SIZE = 13;
const MIN_FONT_SIZE = 10;
const MAX_FONT_SIZE = 24;
const fontSize = ref(DEFAULT_FONT_SIZE);
const fullscreen = ref(false);
let previousBodyOverflow = '';

const lineCount = computed(() => Math.max(1, draft.value.split('\n').length));
const byteCount = computed(() => new TextEncoder().encode(draft.value).length);
const jsonError = computed(() => {
  try {
    JSON.parse(draft.value);
    return '';
  } catch (caught) {
    return caught instanceof Error ? caught.message : 'Invalid JSON';
  }
});

function updateFontSize(value: number) {
  fontSize.value = Math.min(MAX_FONT_SIZE, Math.max(MIN_FONT_SIZE, value));
  localStorage.setItem(FONT_SIZE_KEY, String(fontSize.value));
}

function resetFontSize() {
  updateFontSize(DEFAULT_FONT_SIZE);
}

function setFullscreen(value: boolean) {
  if (fullscreen.value === value) return;
  fullscreen.value = value;
  if (value) {
    previousBodyOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
  } else {
    document.body.style.overflow = previousBodyOverflow;
  }
}

function handleWindowKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && fullscreen.value) setFullscreen(false);
}

function requestSave() {
  if (props.dirty && !props.loading && !props.saving && !jsonError.value) emit('save');
}

onMounted(() => {
  const stored = Number.parseInt(localStorage.getItem(FONT_SIZE_KEY) ?? '', 10);
  if (Number.isFinite(stored)) {
    fontSize.value = Math.min(MAX_FONT_SIZE, Math.max(MIN_FONT_SIZE, stored));
  }
  window.addEventListener('keydown', handleWindowKeydown);
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleWindowKeydown);
  if (fullscreen.value) document.body.style.overflow = previousBodyOverflow;
});
</script>

<template>
  <section
    class="flex flex-col overflow-hidden bg-white shadow-sm dark:bg-slate-900"
    :class="
      fullscreen
        ? 'fixed inset-0 z-[100] [height:100dvh] h-screen min-h-0 rounded-none border-0'
        : 'min-h-[560px] rounded-2xl border border-slate-200 dark:border-white/10'
    "
  >
    <header
      class="flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-slate-100 px-4 py-3 dark:border-white/10"
    >
      <div class="min-w-0">
        <h2 class="truncate font-mono text-sm font-extrabold text-slate-900 dark:text-white">
          {{ file?.name ?? 'Select a configuration file' }}
        </h2>
        <p v-if="file" class="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">
          {{
            file.class === 'leaf'
              ? 'Changes are hot-reloaded after save.'
              : 'Changes become active after restart.'
          }}
          <span v-if="dirty" class="font-bold text-amber-600 dark:text-amber-300">
            Unsaved changes</span
          >
        </p>
      </div>
      <div v-if="file" class="flex flex-wrap items-center gap-2">
        <div
          class="flex items-center overflow-hidden rounded-lg border border-slate-200 bg-slate-50 dark:border-white/10 dark:bg-slate-950"
          aria-label="Editor font size"
        >
          <button
            type="button"
            class="grid h-8 w-8 place-items-center text-slate-500 transition hover:bg-slate-200 disabled:opacity-30 dark:text-slate-300 dark:hover:bg-white/10"
            :disabled="fontSize <= MIN_FONT_SIZE"
            title="Decrease editor font size"
            aria-label="Decrease editor font size"
            @click="updateFontSize(fontSize - 1)"
          >
            <Minus class="h-3.5 w-3.5" aria-hidden="true" />
          </button>
          <span
            class="w-11 text-center font-mono text-[10px] font-bold text-slate-600 dark:text-slate-300"
          >
            {{ fontSize }}px
          </span>
          <button
            type="button"
            class="grid h-8 w-8 place-items-center text-slate-500 transition hover:bg-slate-200 disabled:opacity-30 dark:text-slate-300 dark:hover:bg-white/10"
            :disabled="fontSize >= MAX_FONT_SIZE"
            title="Increase editor font size"
            aria-label="Increase editor font size"
            @click="updateFontSize(fontSize + 1)"
          >
            <Plus class="h-3.5 w-3.5" aria-hidden="true" />
          </button>
          <button
            type="button"
            class="grid h-8 w-8 place-items-center border-l border-slate-200 text-slate-500 transition hover:bg-slate-200 dark:border-white/10 dark:text-slate-300 dark:hover:bg-white/10"
            title="Reset editor font size"
            aria-label="Reset editor font size"
            @click="resetFontSize"
          >
            <RotateCcw class="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        </div>
        <button
          type="button"
          class="grid h-8 w-8 place-items-center rounded-lg border border-slate-200 text-slate-500 transition hover:bg-slate-100 dark:border-white/10 dark:text-slate-300 dark:hover:bg-white/10"
          :title="fullscreen ? 'Exit full-screen editor' : 'Open full-screen editor'"
          :aria-label="fullscreen ? 'Exit full-screen editor' : 'Open full-screen editor'"
          @click="setFullscreen(!fullscreen)"
        >
          <Minimize2 v-if="fullscreen" class="h-3.5 w-3.5" aria-hidden="true" />
          <Maximize2 v-else class="h-3.5 w-3.5" aria-hidden="true" />
        </button>
        <button
          v-if="file.class === 'leaf'"
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-xs font-bold text-slate-600 transition hover:bg-slate-100 disabled:opacity-40 dark:border-white/10 dark:text-slate-300 dark:hover:bg-white/10"
          :disabled="reloading || loading || dirty"
          @click="emit('reload')"
        >
          <RefreshCw
            class="h-3.5 w-3.5"
            :class="{ 'animate-spin': reloading }"
            aria-hidden="true"
          />
          Reload
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg bg-violet-600 px-3 py-2 text-xs font-bold text-white transition hover:bg-violet-500 disabled:cursor-not-allowed disabled:opacity-40"
          :disabled="saving || loading || !dirty || jsonError !== ''"
          @click="requestSave"
        >
          <Save class="h-3.5 w-3.5" aria-hidden="true" />
          {{ saving ? 'Saving…' : 'Save' }}
        </button>
        <button
          v-if="canRestart"
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg bg-amber-500 px-3 py-2 text-xs font-bold text-white transition hover:bg-amber-400 disabled:opacity-40"
          :disabled="restarting"
          @click="emit('restart')"
        >
          <Power class="h-3.5 w-3.5" aria-hidden="true" />
          {{ restarting ? 'Restarting…' : 'Restart server' }}
        </button>
      </div>
    </header>

    <div
      v-if="(restartRequired || file?.class === 'core') && !managed"
      class="border-b border-amber-200 bg-amber-50 px-4 py-2.5 text-xs text-amber-800 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200"
    >
      <span class="inline-flex items-center gap-1.5">
        <AlertTriangle class="h-3.5 w-3.5" aria-hidden="true" />
        {{
          restartRequired
            ? 'These saved changes require a manual server restart.'
            : 'This server is not managed. Restart it manually after saving core configuration.'
        }}
      </span>
    </div>

    <div
      v-if="file"
      class="border-b border-slate-100 px-4 py-2 text-[10px] leading-relaxed text-slate-500 dark:border-white/10 dark:text-slate-400"
    >
      Literal secret values are returned as
      <code class="font-bold text-violet-600 dark:text-violet-300">"__KEEP__"</code> and restored
      from disk when you save. Environment references such as
      <code>env:OPENROUTER_API_KEY</code> remain visible.
    </div>

    <div v-if="loading" class="grid flex-1 place-items-center text-sm text-slate-500">
      Loading configuration…
    </div>
    <div
      v-else-if="!file"
      class="grid flex-1 place-items-center px-6 text-center text-sm text-slate-500"
    >
      Choose a whitelisted file to view or edit it.
    </div>
    <div v-else class="relative flex flex-1 flex-col bg-slate-950">
      <JsonCodeEditor
        v-model="draft"
        :aria-label="`JSON editor for ${file.name}`"
        :disabled="saving || restarting"
        :font-size="fontSize"
        class="flex-1"
        @save="requestSave"
      />
      <footer
        class="flex flex-wrap items-center justify-between gap-2 border-t border-white/10 px-4 py-2 font-mono text-[10px] text-slate-500"
      >
        <span class="flex min-w-0 items-center gap-2">
          <span>Fira Code · JSON · {{ lineCount }} lines</span>
          <span
            class="truncate"
            :class="jsonError ? 'text-rose-400' : 'text-emerald-400'"
            :title="jsonError || 'Valid JSON syntax'"
          >
            {{ jsonError || 'Valid JSON' }}
          </span>
        </span>
        <span>{{ byteCount }} bytes · Tab inserts 2 spaces · Ctrl/⌘+S saves</span>
      </footer>
    </div>
  </section>
</template>
