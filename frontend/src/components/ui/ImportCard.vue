<template>
  <div
    class="rounded-xl border border-dashed border-gray-200 bg-white/60 p-3 shadow-sm transition-colors dark:border-white/5 dark:bg-slate-800/50"
  >
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex items-center gap-3">
        <div class="grid h-9 w-9 shrink-0 place-items-center rounded-lg" :class="palette.iconBg">
          <Upload class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </div>
        <div>
          <p class="text-xs font-black text-slate-700 dark:text-slate-200">{{ title }}</p>
          <p class="text-[10px] text-slate-500 dark:text-slate-400">{{ description }}</p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <label
          class="flex min-w-0 flex-1 cursor-pointer items-center gap-2 rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-xs font-semibold transition sm:flex-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          :class="importFile ? palette.filePicked : 'text-slate-400'"
        >
          <FileUp class="h-3.5 w-3.5 shrink-0" :stroke-width="2.25" aria-hidden="true" />
          <span class="truncate">{{ fileName }}</span>
          <input
            ref="fileInput"
            type="file"
            :accept="accept"
            class="sr-only"
            @change="onFileChange"
          />
        </label>
        <button
          type="button"
          :disabled="!importFile || importing"
          class="inline-flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-black text-white shadow-sm transition hover:-translate-y-0.5 hover:shadow-md disabled:translate-y-0 disabled:cursor-not-allowed disabled:opacity-40"
          :class="palette.button"
          @click="doImport"
        >
          <LoaderCircle v-if="importing" class="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
          <Upload v-else class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          {{ importing ? 'Importing…' : 'Import' }}
        </button>
      </div>
    </div>

    <div
      v-if="importResult"
      class="mt-2 flex items-center gap-1.5 rounded-lg px-3 py-2 text-xs font-bold"
      :class="
        importResult.skipped > 0
          ? 'bg-amber-50 text-amber-800 dark:bg-amber-900/20 dark:text-amber-300'
          : 'bg-emerald-50 text-emerald-800 dark:bg-emerald-900/20 dark:text-emerald-300'
      "
      role="status"
    >
      <component
        :is="importResult.skipped > 0 ? TriangleAlert : CircleCheck"
        class="h-3.5 w-3.5 shrink-0"
        :stroke-width="2.5"
        aria-hidden="true"
      />
      Imported {{ importResult.imported }} {{ resultNoun
      }}<span v-if="importResult.skipped > 0">
        , {{ importResult.skipped }} duplicate{{
          importResult.skipped !== 1 ? 's' : ''
        }}
        skipped</span
      >
    </div>

    <div
      v-if="importError"
      class="mt-2 flex items-center gap-1.5 rounded-lg bg-red-50 px-3 py-2 text-xs font-bold text-red-700 dark:bg-red-900/20 dark:text-red-400"
      role="alert"
    >
      <CircleAlert class="h-3.5 w-3.5 shrink-0" :stroke-width="2.25" aria-hidden="true" />
      {{ importError }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { CircleAlert, CircleCheck, FileUp, LoaderCircle, TriangleAlert, Upload } from '@lucide/vue';

export interface ImportSummary {
  imported: number;
  skipped: number;
}

const props = withDefaults(
  defineProps<{
    title: string;
    description: string;
    /** File extensions for the picker, e.g. ".html,.htm" or ".csv" — empty accepts any. */
    accept?: string;
    /** Result message nouns: "bookmark" → "bookmarks", "history entry" → "history entries". */
    noun: string;
    nounPlural?: string;
    color?: 'amber' | 'violet' | 'cyan' | 'indigo' | 'emerald';
    /** Performs the upload; must resolve with the import summary. */
    onImport: (file: File) => Promise<ImportSummary>;
  }>(),
  {
    accept: '',
    nounPlural: undefined,
    color: 'amber',
  },
);

const emit = defineEmits<{ imported: [result: ImportSummary] }>();

const fileInput = ref<HTMLInputElement | null>(null);
const importFile = ref<File | null>(null);
const importing = ref(false);
const importResult = ref<ImportSummary | null>(null);
const importError = ref<string | null>(null);

const fileName = computed(() => importFile.value?.name || 'Choose file…');

const palettes = {
  amber: {
    iconBg: 'bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400',
    button: 'bg-gradient-to-r from-amber-500 to-orange-600',
    filePicked: 'border-amber-300 text-amber-700 dark:border-amber-700 dark:text-amber-300',
  },
  violet: {
    iconBg: 'bg-violet-50 text-violet-600 dark:bg-violet-900/20 dark:text-violet-400',
    button: 'bg-gradient-to-r from-violet-500 to-purple-600',
    filePicked: 'border-violet-300 text-violet-700 dark:border-violet-700 dark:text-violet-300',
  },
  cyan: {
    iconBg: 'bg-cyan-50 text-cyan-600 dark:bg-cyan-900/20 dark:text-cyan-400',
    button: 'bg-gradient-to-r from-cyan-500 to-blue-600',
    filePicked: 'border-cyan-300 text-cyan-700 dark:border-cyan-700 dark:text-cyan-300',
  },
  indigo: {
    iconBg: 'bg-indigo-50 text-indigo-600 dark:bg-indigo-900/20 dark:text-indigo-400',
    button: 'bg-gradient-to-r from-indigo-500 to-blue-600',
    filePicked: 'border-indigo-300 text-indigo-700 dark:border-indigo-700 dark:text-indigo-300',
  },
  emerald: {
    iconBg: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-400',
    button: 'bg-gradient-to-r from-emerald-500 to-teal-600',
    filePicked: 'border-emerald-300 text-emerald-700 dark:border-emerald-700 dark:text-emerald-300',
  },
} as const;

const palette = computed(() => palettes[props.color]);

const resultNoun = computed(() =>
  (importResult.value?.imported ?? 0) === 1 ? props.noun : (props.nounPlural ?? `${props.noun}s`),
);

const onFileChange = (e: Event) => {
  importFile.value = (e.target as HTMLInputElement).files?.[0] || null;
  importResult.value = null;
  importError.value = null;
};

const doImport = async () => {
  if (!importFile.value) return;
  importing.value = true;
  importResult.value = null;
  importError.value = null;
  try {
    const result = await props.onImport(importFile.value);
    importResult.value = result;
    importFile.value = null;
    if (fileInput.value) fileInput.value.value = '';
    emit('imported', result);
  } catch (e) {
    importError.value = e instanceof Error ? e.message : 'Import failed';
  } finally {
    importing.value = false;
  }
};
</script>
