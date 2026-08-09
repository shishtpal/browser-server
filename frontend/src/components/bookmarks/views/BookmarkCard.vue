<template>
  <article
    class="group flex flex-col overflow-hidden rounded-xl border border-gray-200/80 bg-white shadow-sm transition hover:-translate-y-0.5 hover:border-cyan-200 hover:shadow-md dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-cyan-500/30"
  >
    <div
      class="flex items-start justify-between gap-3 border-b border-gray-100 bg-gradient-to-br from-slate-900 to-slate-800 p-3 text-white transition-colors dark:border-slate-700/60 dark:from-slate-950 dark:to-slate-900"
    >
      <div
        class="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-white/10 text-sm font-black"
      >
        {{ getInitial(bookmark.title) }}
      </div>
      <div class="min-w-0 text-right">
        <h3 class="truncate text-sm font-black" :title="bookmark.title">{{ bookmark.title }}</h3>
        <a
          :href="bookmark.url"
          target="_blank"
          rel="noopener"
          class="inline-flex items-center gap-1 truncate text-[10px] text-cyan-200 transition hover:text-cyan-50"
        >
          <Globe class="h-2.5 w-2.5 shrink-0" :stroke-width="2.25" aria-hidden="true" />
          {{ formatHost(bookmark.url) }}
        </a>
      </div>
    </div>

    <div class="flex flex-1 flex-col p-3">
      <a
        :href="bookmark.url"
        target="_blank"
        rel="noopener"
        class="inline-flex items-center gap-1 text-xs font-bold break-all text-blue-600 transition-colors hover:underline dark:text-blue-400"
      >
        {{ bookmark.url }}
        <SquareArrowOutUpRight
          class="h-3 w-3 shrink-0 opacity-60"
          :stroke-width="2.5"
          aria-hidden="true"
        />
      </a>

      <p
        v-if="bookmark.folder_path"
        class="mt-1.5 inline-flex items-center gap-1 truncate text-[10px] font-semibold text-slate-400 dark:text-slate-500"
        :title="bookmark.folder_path"
      >
        <Folder class="h-3 w-3 shrink-0" :stroke-width="2.25" aria-hidden="true" />
        {{ bookmark.folder_path }}
      </p>

      <p
        v-if="bookmark.description"
        class="mt-1.5 line-clamp-2 text-xs leading-5 text-slate-500 transition-colors dark:text-slate-400"
      >
        {{ bookmark.description }}
      </p>

      <div v-if="bookmark.tags.length" class="mt-auto flex flex-wrap gap-1 pt-3">
        <BookmarkTag
          v-for="tag in bookmark.tags"
          :key="tag"
          :tag="tag"
          @filter="$emit('filter-tag', $event)"
        />
      </div>

      <div
        class="mt-3 flex gap-1.5 border-t border-gray-100 pt-3 transition-colors dark:border-slate-700/50"
      >
        <button
          type="button"
          class="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-gray-100 px-3 py-2 text-xs font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600"
          @click="$emit('edit', bookmark)"
        >
          <Pencil class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Edit
        </button>
        <button
          type="button"
          class="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-red-50 px-3 py-2 text-xs font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30"
          @click="$emit('delete', bookmark.id)"
        >
          <Trash2 class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Delete
        </button>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import type { BookmarkResponse } from '../../../types';
import { Folder, Globe, Pencil, SquareArrowOutUpRight, Trash2 } from '@lucide/vue';
import BookmarkTag from '../BookmarkTag.vue';
import { formatHost, getInitial } from '../bookmarkFormat';

defineProps<{ bookmark: BookmarkResponse }>();

defineEmits<{
  edit: [bookmark: BookmarkResponse];
  delete: [id: number];
  'filter-tag': [tag: string];
}>();
</script>
