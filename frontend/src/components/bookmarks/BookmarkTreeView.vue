<template>
  <div
    class="overflow-hidden rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90"
  >
    <BookmarkTreeNode
      v-for="node in nodes"
      :key="node.key"
      :node="node"
      @toggle-folder="$emit('toggleFolder', $event)"
      @edit="$emit('edit', $event)"
      @delete="$emit('delete', $event)"
      @filter-tag="$emit('filterTag', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import type { BookmarkResponse } from '../../types';
import type { FlatTreeEntry } from './composables/useBookmarkTree';
import BookmarkTreeNode from './BookmarkTreeNode.vue';

defineProps<{ nodes: FlatTreeEntry[] }>();

defineEmits<{
  toggleFolder: [key: string];
  edit: [bookmark: BookmarkResponse];
  delete: [id: number];
  filterTag: [tag: string];
}>();
</script>
