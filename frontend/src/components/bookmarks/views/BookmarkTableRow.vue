<template>
  <tr class="group transition hover:bg-cyan-50/60 dark:hover:bg-cyan-900/20">
    <!-- Avatar -->
    <td class="px-3 py-3">
      <div
        class="grid h-8 w-8 place-items-center rounded-lg bg-gradient-to-br from-slate-900 to-slate-800 text-xs font-black text-white dark:from-slate-950 dark:to-slate-900"
      >
        {{ getInitial(bookmark.title) }}
      </div>
    </td>

    <!-- Title + host -->
    <td class="truncate px-3 py-3">
      <a
        :href="bookmark.url"
        target="_blank"
        rel="noopener"
        class="group/link flex min-w-0 items-center gap-1.5"
        :title="bookmark.title"
      >
        <span
          class="block truncate text-sm font-black text-slate-900 transition-colors group-hover/link:text-cyan-700 dark:text-white dark:group-hover/link:text-cyan-400"
        >
          {{ bookmark.title }}
        </span>
        <SquareArrowOutUpRight
          class="h-3 w-3 shrink-0 text-slate-300 opacity-0 transition group-hover/link:text-cyan-500 group-hover/link:opacity-100 dark:text-slate-600"
          :stroke-width="2.5"
          aria-hidden="true"
        />
      </a>
      <span class="block truncate text-[10px] text-cyan-600 dark:text-cyan-400">
        {{ formatHost(bookmark.url) }}
      </span>
    </td>

    <!-- URL -->
    <td class="truncate px-3 py-3">
      <a
        :href="bookmark.url"
        target="_blank"
        rel="noopener"
        class="block truncate text-sm font-semibold text-blue-600 transition-colors hover:underline dark:text-blue-400"
        :title="bookmark.url"
      >
        {{ bookmark.url }}
      </a>
    </td>

    <!-- Description -->
    <td class="truncate px-3 py-3">
      <span
        class="block truncate text-sm text-slate-500 transition-colors dark:text-slate-400"
        :title="bookmark.description"
      >
        {{ bookmark.description || '—' }}
      </span>
    </td>

    <!-- Folder -->
    <td class="truncate px-3 py-3">
      <span
        v-if="bookmark.folder_path"
        class="inline-flex max-w-full items-center gap-1 rounded-md bg-slate-50 px-1.5 py-0.5 text-[10px] font-semibold text-slate-500 dark:bg-slate-800 dark:text-slate-400"
        :title="bookmark.folder_path"
      >
        <Folder class="h-3 w-3 shrink-0" :stroke-width="2.25" aria-hidden="true" />
        <span class="truncate">{{ bookmark.folder_path }}</span>
      </span>
      <span v-else class="text-[10px] text-slate-400 dark:text-slate-500">—</span>
    </td>

    <!-- Tags -->
    <td class="px-3 py-3">
      <div class="flex flex-wrap gap-1">
        <BookmarkTag
          v-for="tag in bookmark.tags"
          :key="tag"
          :tag="tag"
          @filter="$emit('filter-tag', $event)"
        />
        <span v-if="!bookmark.tags.length" class="text-[10px] text-slate-400 dark:text-slate-500">
          —
        </span>
      </div>
    </td>

    <!-- Actions -->
    <td class="px-3 py-3 text-right">
      <div class="flex justify-end gap-0.5">
        <button
          type="button"
          title="Edit bookmark"
          aria-label="Edit bookmark"
          class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-cyan-50 hover:text-cyan-700 dark:hover:bg-cyan-900/10 dark:hover:text-cyan-400"
          @click="$emit('edit', bookmark)"
        >
          <Pencil class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
        <button
          type="button"
          title="Delete bookmark"
          aria-label="Delete bookmark"
          class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
          @click="$emit('delete', bookmark.id)"
        >
          <Trash2 class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
      </div>
    </td>
  </tr>
</template>

<script setup lang="ts">
import type { BookmarkResponse } from '../../../types';
import { Folder, Pencil, SquareArrowOutUpRight, Trash2 } from '@lucide/vue';
import BookmarkTag from '../BookmarkTag.vue';
import { formatHost, getInitial } from '../bookmarkFormat';

defineProps<{ bookmark: BookmarkResponse }>();

defineEmits<{
  edit: [bookmark: BookmarkResponse];
  delete: [id: number];
  'filter-tag': [tag: string];
}>();
</script>
