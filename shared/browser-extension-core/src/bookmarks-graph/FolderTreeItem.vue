<script setup lang="ts">
import type { FolderTreeNode } from './buildGraph'

const props = defineProps<{
  folder: FolderTreeNode
  expandedFolders: Set<string>
  depth: number
}>()

const emit = defineEmits<{
  toggle: [folderId: string]
}>()

function isExpanded(): boolean {
  return props.expandedFolders.has(props.folder.id)
}
</script>

<template>
  <div>
    <!-- Folder row -->
    <button
      type="button"
      class="group flex w-full items-center gap-1.5 rounded-lg px-2 py-1.5 text-left transition hover:bg-slate-800/70"
      :class="isExpanded() ? 'bg-slate-800/40' : ''"
      :style="{ paddingLeft: `${depth * 12 + 8}px` }"
      @click="emit('toggle', folder.id)"
    >
      <!-- Chevron -->
      <svg
        class="h-3 w-3 shrink-0 text-slate-500 transition-transform duration-150"
        :class="isExpanded() ? 'rotate-90' : ''"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="m9 18 6-6-6-6" />
      </svg>

      <!-- Folder icon -->
      <svg
        class="h-3.5 w-3.5 shrink-0"
        :class="isExpanded() ? 'text-amber-400' : 'text-amber-500/60'"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z" />
      </svg>

      <!-- Name -->
      <span class="min-w-0 flex-1 truncate text-xs font-medium text-slate-300 group-hover:text-slate-100">
        {{ folder.name }}
      </span>

      <!-- Count badge -->
      <span class="shrink-0 rounded-full bg-slate-800 px-1.5 py-0.5 text-[9px] font-medium text-slate-500">
        {{ folder.leafCount }}
      </span>
    </button>

    <!-- Nested children (only shown when expanded in sidebar for navigation clarity) -->
    <div v-if="folder.children.length > 0" v-show="isExpanded()">
      <FolderTreeItem
        v-for="child in folder.children"
        :key="child.id"
        :folder="child"
        :expanded-folders="expandedFolders"
        :depth="depth + 1"
        @toggle="emit('toggle', $event)"
      />
    </div>
  </div>
</template>
