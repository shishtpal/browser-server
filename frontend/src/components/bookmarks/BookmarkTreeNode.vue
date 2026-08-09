<template>
  <!-- Folder row -->
  <button
    v-if="node.type === 'folder'"
    type="button"
    class="flex w-full cursor-pointer items-center gap-2 border-b border-gray-100 px-3 py-2.5 text-left text-xs font-black text-slate-600 transition hover:bg-slate-50 dark:border-slate-700/50 dark:text-slate-300 dark:hover:bg-slate-800/50"
    :style="{ paddingLeft: 12 + node.depth * 20 + 'px' }"
    :aria-expanded="node.expanded"
    @click="$emit('toggle-folder', node.key)"
  >
    <ChevronRight
      class="h-3.5 w-3.5 shrink-0 text-slate-400 transition-transform"
      :class="{ 'rotate-90': node.expanded }"
      :stroke-width="2.5"
      aria-hidden="true"
    />
    <FolderOpen
      v-if="node.expanded"
      class="h-4 w-4 shrink-0 text-amber-500"
      :stroke-width="2"
      aria-hidden="true"
    />
    <Folder v-else class="h-4 w-4 shrink-0 text-amber-500" :stroke-width="2" aria-hidden="true" />
    <span class="truncate font-black">{{ node.name }}</span>
    <span
      class="shrink-0 text-[10px] font-semibold text-slate-400 tabular-nums dark:text-slate-500"
    >
      ({{ node.count }})
    </span>
  </button>

  <!-- Bookmark row -->
  <div
    v-else-if="node.bookmark"
    class="group flex items-center gap-2 border-b border-gray-100 px-3 py-2 transition last:border-b-0 hover:bg-cyan-50/60 dark:border-slate-700/50 dark:hover:bg-cyan-900/20"
    :style="{ paddingLeft: 12 + node.depth * 20 + 'px' }"
  >
    <div
      class="grid h-7 w-7 shrink-0 place-items-center rounded-md bg-gradient-to-br from-slate-900 to-slate-800 text-[10px] font-black text-white dark:from-slate-950 dark:to-slate-900"
      aria-hidden="true"
    >
      {{ getInitial(node.bookmark.title) }}
    </div>

    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <span
          class="truncate text-sm font-black text-slate-900 dark:text-white"
          :title="node.bookmark.title"
        >
          {{ node.bookmark.title }}
        </span>
        <span class="shrink-0 text-[10px] text-cyan-600 dark:text-cyan-400">
          {{ formatHost(node.bookmark.url) }}
        </span>
      </div>
      <div class="flex min-w-0 items-center gap-2">
        <a
          :href="node.bookmark.url"
          target="_blank"
          rel="noopener"
          class="truncate text-xs font-semibold text-blue-600 hover:underline dark:text-blue-400"
        >
          {{ node.bookmark.url }}
        </a>
        <BookmarkTag
          v-for="tag in node.bookmark.tags"
          :key="tag"
          :tag="tag"
          @filter="$emit('filter-tag', $event)"
        />
      </div>
    </div>

    <div
      class="flex shrink-0 gap-0.5 transition sm:opacity-0 sm:group-focus-within:opacity-100 sm:group-hover:opacity-100"
    >
      <button
        type="button"
        title="Edit bookmark"
        aria-label="Edit bookmark"
        class="grid h-7 w-7 place-items-center rounded-lg text-slate-400 transition hover:bg-cyan-50 hover:text-cyan-700 dark:hover:bg-cyan-900/10 dark:hover:text-cyan-400"
        @click="$emit('edit', node.bookmark)"
      >
        <Pencil class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
      </button>
      <button
        type="button"
        title="Delete bookmark"
        aria-label="Delete bookmark"
        class="grid h-7 w-7 place-items-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
        @click="$emit('delete', node.bookmark.id)"
      >
        <Trash2 class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { BookmarkResponse } from '../../types';
import type { FlatTreeEntry } from './composables/useBookmarkTree';
import { ChevronRight, Folder, FolderOpen, Pencil, Trash2 } from '@lucide/vue';
import BookmarkTag from './BookmarkTag.vue';
import { formatHost, getInitial } from './bookmarkFormat';

defineProps<{ node: FlatTreeEntry }>();

defineEmits<{
  'toggle-folder': [key: string];
  edit: [bookmark: BookmarkResponse];
  delete: [id: number];
  'filter-tag': [tag: string];
}>();
</script>
