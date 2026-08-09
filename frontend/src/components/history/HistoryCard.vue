<template>
  <article
    class="relative rounded-xl border border-gray-200/80 bg-white p-3 shadow-sm transition hover:-translate-y-0.5 hover:border-violet-200 hover:shadow-md dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-violet-500/30"
  >
    <!-- Timeline dot -->
    <div
      class="absolute top-4 left-3 grid h-3 w-3 place-items-center rounded-full border-3 border-white bg-violet-500 shadow-sm dark:border-slate-800"
      aria-hidden="true"
    ></div>

    <div class="pl-7">
      <div class="flex items-start justify-between gap-2">
        <div class="min-w-0 flex-1">
          <h3
            class="truncate text-sm font-black text-slate-900 transition-colors dark:text-white"
            :title="entry.title"
          >
            {{ entry.title }}
          </h3>
          <a
            :href="entry.url"
            target="_blank"
            rel="noopener"
            class="mt-0.5 block truncate text-xs font-semibold text-blue-600 transition-colors hover:underline dark:text-blue-400"
            :title="entry.url"
          >
            {{ entry.url }}
          </a>

          <div class="mt-2 flex flex-wrap gap-1.5">
            <span
              class="rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-bold text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400"
            >
              {{ formatDate(entry.visited_at) }}
            </span>
            <span
              v-if="entry.duration > 0"
              class="inline-flex items-center gap-1 rounded-md bg-violet-50 px-2 py-0.5 text-[10px] font-bold text-violet-700 transition-colors dark:bg-violet-900/20 dark:text-violet-400"
            >
              <Clock class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
              {{ formatDuration(entry.duration) }}
            </span>
          </div>
        </div>

        <button
          type="button"
          title="Delete entry"
          aria-label="Delete history entry"
          class="grid h-8 w-8 shrink-0 place-items-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
          @click="$emit('delete', entry)"
        >
          <Trash2 class="h-4 w-4" aria-hidden="true" />
        </button>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import type { History } from '../../types';
import { Clock, Trash2 } from '@lucide/vue';
import { formatDate, formatDuration } from '../../lib/utils';

defineProps<{
  entry: History;
}>();

defineEmits<{ delete: [entry: History] }>();
</script>
