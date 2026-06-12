<template>
  <div class="overflow-hidden rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90">
    <BookmarkTreeNode
      v-for="n in nodes"
      :key="n.key"
      :node="n"
      @toggle-folder="(k) => emit('toggleFolder', k)"
      @edit="(b) => emit('edit', b)"
      @delete="(id) => emit('delete', id)"
      @filter-tag="(t) => emit('filterTag', t)"
    />
  </div>
</template>

<script setup lang="ts">
import type { FlatTreeEntry } from '../../composables/useBookmarkTree'
import type { BookmarkResponse } from '../../types'
import BookmarkTreeNode from './BookmarkTreeNode.vue'

interface Props {
  nodes: FlatTreeEntry[]
}

defineProps<Props>()

const emit = defineEmits<{
  toggleFolder: [key: string]
  edit: [bookmark: BookmarkResponse]
  delete: [id: number]
  filterTag: [tag: string]
}>()
</script>
