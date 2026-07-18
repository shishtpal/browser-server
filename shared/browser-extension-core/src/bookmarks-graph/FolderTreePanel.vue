<script setup lang="ts">
import type { FolderTreeNode } from './buildGraph'

defineProps<{
  folders: FolderTreeNode[]
  expandedFolders: Set<string>
}>()

const emit = defineEmits<{
  toggle: [folderId: string]
}>()
</script>

<template>
  <aside class="flex w-56 shrink-0 flex-col overflow-hidden border-r border-slate-800/80 bg-slate-900/50">
    <!-- Header -->
    <div class="flex h-14 shrink-0 items-center border-b border-slate-800/60 px-4">
      <h2 class="text-xs font-semibold uppercase tracking-wider text-slate-400">Folders</h2>
    </div>

    <!-- Tree -->
    <div class="min-h-0 flex-1 overflow-y-auto px-2 py-2">
      <div v-if="folders.length === 0" class="px-2 py-4 text-center text-xs text-slate-600">
        No folders
      </div>
      <FolderTreeItem
        v-for="folder in folders"
        :key="folder.id"
        :folder="folder"
        :expanded-folders="expandedFolders"
        :depth="0"
        @toggle="emit('toggle', $event)"
      />
    </div>
  </aside>
</template>

<script lang="ts">
import FolderTreeItem from './FolderTreeItem.vue'
</script>
