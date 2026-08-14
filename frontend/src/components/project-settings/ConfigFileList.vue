<script setup lang="ts">
import { FileJson, RefreshCw } from '@lucide/vue';
import type { AdminConfigFile } from '../../lib/api';

defineProps<{
  files: AdminConfigFile[];
  selected: AdminConfigFile | null;
  loading: boolean;
}>();

const emit = defineEmits<{
  select: [file: AdminConfigFile];
  refresh: [];
}>();

function formatBytes(bytes: number): string {
  if (!bytes) return '—';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatModified(value?: string): string {
  if (!value) return 'Not created';
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(value),
  );
}
</script>

<template>
  <section
    class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-white/10 dark:bg-slate-900"
  >
    <header
      class="flex items-center justify-between border-b border-slate-100 px-4 py-3 dark:border-white/10"
    >
      <div>
        <h2 class="text-sm font-extrabold text-slate-900 dark:text-white">Configuration files</h2>
        <p class="text-[11px] text-slate-500 dark:text-slate-400">Explicit server-side whitelist</p>
      </div>
      <button
        type="button"
        class="grid h-8 w-8 place-items-center rounded-lg text-slate-500 transition hover:bg-slate-100 dark:hover:bg-white/10"
        title="Refresh file list"
        :disabled="loading"
        @click="emit('refresh')"
      >
        <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" aria-hidden="true" />
      </button>
    </header>

    <div class="divide-y divide-slate-100 dark:divide-white/5">
      <button
        v-for="file in files"
        :key="file.name"
        type="button"
        class="flex w-full items-start gap-3 px-4 py-3 text-left transition"
        :class="
          selected?.name === file.name
            ? 'bg-violet-50/80 dark:bg-violet-500/10'
            : 'hover:bg-slate-50 dark:hover:bg-white/5'
        "
        @click="emit('select', file)"
      >
        <FileJson
          class="mt-0.5 h-4 w-4 shrink-0"
          :class="selected?.name === file.name ? 'text-violet-600' : 'text-slate-400'"
          aria-hidden="true"
        />
        <span class="min-w-0 flex-1">
          <span
            class="block truncate font-mono text-[11px] font-bold text-slate-800 dark:text-slate-200"
          >
            {{ file.name }}
          </span>
          <span class="mt-1 flex flex-wrap items-center gap-1.5">
            <span
              class="rounded-full px-1.5 py-0.5 text-[9px] font-bold tracking-wide uppercase"
              :class="
                file.class === 'leaf'
                  ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
                  : 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
              "
            >
              {{ file.class === 'leaf' ? 'hot reload' : 'restart' }}
            </span>
            <span v-if="!file.exists" class="text-[9px] font-bold text-rose-500">missing</span>
            <span class="text-[10px] text-slate-400">{{ formatBytes(file.size) }}</span>
          </span>
          <span class="mt-1 block text-[10px] text-slate-400">{{
            formatModified(file.modified_at)
          }}</span>
        </span>
      </button>
    </div>
  </section>
</template>
