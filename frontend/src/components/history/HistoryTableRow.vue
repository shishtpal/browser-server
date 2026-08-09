<template>
  <tr class="group transition hover:bg-violet-50/60 dark:hover:bg-violet-900/20">
    <!-- Title + favicon-style glyph -->
    <td class="max-w-md truncate px-3 py-3">
      <div class="flex min-w-0 items-center gap-3">
        <div
          class="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-violet-50 text-violet-600 dark:bg-violet-900/20 dark:text-violet-400"
        >
          <Globe class="h-4 w-4" :stroke-width="2" aria-hidden="true" />
        </div>
        <span
          class="block truncate text-sm font-black text-slate-900 transition-colors dark:text-white"
          :title="entry.title"
        >
          {{ entry.title }}
        </span>
      </div>
    </td>

    <td class="max-w-md px-3 py-3">
      <a
        :href="entry.url"
        target="_blank"
        rel="noopener"
        class="group/link flex min-w-0 items-center gap-1.5"
        :title="entry.url"
      >
        <span
          class="truncate text-sm font-semibold text-blue-600 transition-colors group-hover/link:underline dark:text-blue-400"
        >
          {{ entry.url }}
        </span>
        <SquareArrowOutUpRight
          class="h-3 w-3 shrink-0 opacity-0 transition group-hover/link:opacity-100"
          :stroke-width="2.5"
          aria-hidden="true"
        />
      </a>
    </td>

    <td class="px-3 py-3">
      <span
        class="rounded-md bg-gray-100 px-2 py-1 text-[10px] font-bold whitespace-nowrap text-slate-500 transition-colors dark:bg-slate-700 dark:text-slate-400"
      >
        {{ formatDate(entry.visited_at) }}
      </span>
    </td>

    <td class="px-3 py-3">
      <span
        v-if="entry.duration > 0"
        class="inline-flex items-center gap-1 rounded-md bg-violet-50 px-2 py-1 text-[10px] font-bold whitespace-nowrap text-violet-700 transition-colors dark:bg-violet-900/20 dark:text-violet-400"
      >
        <Clock class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
        {{ formatDuration(entry.duration) }}
      </span>
      <span v-else class="text-[10px] text-slate-400 dark:text-slate-500">—</span>
    </td>

    <td class="px-3 py-3 text-right">
      <button
        type="button"
        title="Delete entry"
        aria-label="Delete history entry"
        class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
        @click="$emit('delete', entry)"
      >
        <Trash2 class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
      </button>
    </td>
  </tr>
</template>

<script setup lang="ts">
import type { History } from '../../types';
import { Clock, Globe, SquareArrowOutUpRight, Trash2 } from '@lucide/vue';
import { formatDate, formatDuration } from '../../lib/utils';

defineProps<{
  entry: History;
}>();

defineEmits<{ delete: [entry: History] }>();
</script>
