<script setup lang="ts">
import type { BookmarkResponse } from '@browser-server/shared-client'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { useBookmarksGraph } from './useBookmarksGraph'
import FolderNode from './FolderNode.vue'
import BookmarkNode from './BookmarkNode.vue'
import RootNode from './RootNode.vue'
import BookmarkInspector from './BookmarkInspector.vue'
import BookmarkEditDialog from './BookmarkEditDialog.vue'
import FolderTreePanel from './FolderTreePanel.vue'

const {
  isReady,
  isLoading,
  errorMessage,
  bookmarks,
  allTags,
  nodes,
  edges,
  folderTree,
  expandedFolders,
  searchQuery,
  activeTag,
  selectedBookmark,
  load,
  refresh,
  toggleFolder,
  expandAll,
  collapseAll,
  selectBookmark,
  updateBookmark,
  moveBookmark,
  deleteBookmark,
  cleanup,
} = useBookmarksGraph()

const { fitView, onNodesInitialized } = useVueFlow()

const editing = ref<BookmarkResponse | null>(null)
const showInspector = ref(false)
const hasFittedOnce = ref(false)

// Fit the view only on the very first render — never again after that
onNodesInitialized(() => {
  if (!hasFittedOnce.value) {
    hasFittedOnce.value = true
    fitView({ padding: 0.2 })
  }
})

function onNodeDoubleClick(event: { node: { id: string; type: string; data: unknown } }) {
  const node = event.node
  if (node.type === 'folder') {
    toggleFolder(node.id)
  } else if (node.type === 'bookmark') {
    const data = node.data as { bookmark: BookmarkResponse }
    selectBookmark(data.bookmark)
    showInspector.value = true
  }
}

function onNodeClick(event: { node: { id: string; type: string; data: unknown } }) {
  const node = event.node
  if (node.type === 'bookmark') {
    const data = node.data as { bookmark: BookmarkResponse }
    selectBookmark(data.bookmark)
    showInspector.value = true
  } else if (node.type === 'folder') {
    // Single-click on folder also toggles
    toggleFolder(node.id)
  }
}

function requestEdit() {
  if (selectedBookmark.value) {
    editing.value = selectedBookmark.value
  }
}

async function onSaveEdit(bookmark: BookmarkResponse, payload: {
  title: string
  url: string
  description: string
  tags: string
  folderPath: string
}) {
  const tagList = payload.tags
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean)
  await updateBookmark(bookmark.id, {
    user_id: bookmark.user_id,
    title: payload.title.trim(),
    url: payload.url.trim(),
    description: payload.description.trim(),
    tags: tagList,
    folder_path: payload.folderPath.trim(),
  })
  editing.value = null
}

async function onDelete() {
  const b = selectedBookmark.value
  if (!b) return
  if (!confirm('Delete this bookmark?')) return
  await deleteBookmark(b.id)
  selectBookmark(null)
  showInspector.value = false
}

async function onMove() {
  const b = selectedBookmark.value
  if (!b) return
  const target = prompt('Move bookmark to folder (leave blank for Unfiled):', b.folder_path || '')
  if (target === null) return
  await moveBookmark(b.id, target.trim())
}

function closeInspector() {
  showInspector.value = false
  selectBookmark(null)
}

onMounted(() => {
  if (isReady.value) void load()
})

watch(isReady, (ready) => {
  if (ready) void load()
})

onBeforeUnmount(() => {
  cleanup()
})
</script>

<template>
  <main class="flex h-screen min-w-[760px] flex-col overflow-hidden bg-slate-950 text-slate-100">
    <!-- Header -->
    <header class="flex h-14 shrink-0 items-center justify-between border-b border-slate-800/80 bg-slate-900/50 px-5 backdrop-blur">
      <div class="min-w-0">
        <h1 class="text-base font-semibold tracking-tight">Bookmarks Graph</h1>
        <p class="text-[11px] text-slate-500">Interactive mind-map · double-click folders to expand</p>
      </div>
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="rounded-lg border border-slate-700/80 px-3 py-1.5 text-xs font-medium text-slate-400 transition hover:border-slate-500 hover:text-white"
          @click="expandAll"
        >
          Expand All
        </button>
        <button
          type="button"
          class="rounded-lg border border-slate-700/80 px-3 py-1.5 text-xs font-medium text-slate-400 transition hover:border-slate-500 hover:text-white"
          @click="collapseAll"
        >
          Collapse All
        </button>
        <button
          type="button"
          class="rounded-lg border border-rose-500/30 bg-rose-500/10 px-3 py-1.5 text-xs font-semibold text-rose-300 transition hover:bg-rose-500/20"
          @click="refresh"
        >
          Refresh
        </button>
      </div>
    </header>

    <!-- Not configured -->
    <div v-if="!isReady" class="flex flex-1 items-center justify-center text-sm text-slate-400">
      Configure a server URL, API token, and user ID in extension settings first.
    </div>

    <!-- Main content -->
    <div v-else class="flex min-h-0 flex-1 overflow-hidden">
      <!-- Folder tree sidebar -->
      <FolderTreePanel
        :folders="folderTree"
        :expanded-folders="expandedFolders"
        @toggle="toggleFolder"
      />

      <!-- Graph panel -->
      <div class="relative min-w-0 flex-1 overflow-hidden">
        <!-- Filter bar floating overlay -->
        <div class="pointer-events-none absolute left-4 top-4 z-10 flex flex-col gap-2">
          <!-- Search -->
          <div class="pointer-events-auto flex w-72 items-center gap-2 rounded-xl border border-slate-700/80 bg-slate-900/95 px-3 py-2 shadow-lg backdrop-blur">
            <svg class="h-3.5 w-3.5 shrink-0 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              v-model="searchQuery"
              type="search"
              placeholder="Search bookmarks…"
              class="w-full bg-transparent text-xs text-slate-100 outline-none placeholder:text-slate-600"
            />
          </div>

          <!-- Tags -->
          <div v-if="allTags.length" class="pointer-events-auto flex max-w-72 flex-wrap gap-1 rounded-xl border border-slate-700/80 bg-slate-900/95 px-3 py-2 shadow-lg backdrop-blur">
            <button
              type="button"
              class="rounded-full px-2 py-0.5 text-[10px] font-semibold transition"
              :class="!activeTag ? 'bg-rose-500/15 text-rose-300 ring-1 ring-inset ring-rose-500/30' : 'text-slate-400 hover:bg-slate-800 hover:text-slate-300'"
              @click="activeTag = ''"
            >All</button>
            <button
              v-for="tag in allTags"
              :key="tag"
              type="button"
              class="rounded-full px-2 py-0.5 text-[10px] font-medium transition"
              :class="activeTag === tag ? 'bg-rose-500/15 text-rose-300 ring-1 ring-inset ring-rose-500/30' : 'text-slate-400 hover:bg-slate-800 hover:text-slate-300'"
              @click="activeTag = activeTag === tag ? '' : tag"
            >#{{ tag }}</button>
          </div>
        </div>

        <!-- Loading state -->
        <div v-if="isLoading && bookmarks.length === 0" class="absolute inset-0 flex items-center justify-center text-sm text-slate-500">
          <div class="flex flex-col items-center gap-3">
            <div class="h-8 w-8 animate-spin rounded-full border-2 border-slate-700 border-t-rose-500"></div>
            <span>Loading bookmarks…</span>
          </div>
        </div>

        <!-- Empty state -->
        <div v-else-if="bookmarks.length === 0" class="absolute inset-0 flex flex-col items-center justify-center gap-3 text-sm text-slate-500">
          <svg class="h-12 w-12 text-slate-700" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v16z" />
          </svg>
          <p class="font-medium text-slate-400">No bookmarks yet</p>
          <p class="text-xs text-slate-600">Start saving bookmarks using the extension toolbar.</p>
        </div>

        <!-- No filter results -->
        <div v-else-if="nodes.length === 0" class="absolute inset-0 flex items-center justify-center text-sm text-slate-500">
          No bookmarks match the current filter.
        </div>

        <!-- Vue Flow graph -->
        <VueFlow
          v-else
          :nodes="nodes"
          :edges="edges"
          :default-viewport="{ zoom: 0.8, x: 50, y: 50 }"
          :min-zoom="0.2"
          :max-zoom="3"
          :pan-on-scroll="true"
          :zoom-on-scroll="true"
          :nodes-draggable="true"
          :nodes-connectable="false"
          :edges-updatable="false"
          class="h-full w-full"
          @node-double-click="onNodeDoubleClick"
          @node-click="onNodeClick"
        >
          <!-- Custom node types -->
          <template #node-root="rootProps">
            <RootNode :data="rootProps.data" />
          </template>
          <template #node-folder="folderProps">
            <FolderNode :id="folderProps.id" :data="folderProps.data" @toggle="toggleFolder" />
          </template>
          <template #node-bookmark="bookmarkProps">
            <BookmarkNode :data="bookmarkProps.data" @select="(b) => { selectBookmark(b); showInspector = true }" />
          </template>

          <!-- Background pattern -->
          <Background :gap="24" :size="1" pattern-color="rgba(51, 65, 85, 0.3)" />

          <!-- Controls -->
          <Controls position="bottom-right" />

          <!-- MiniMap -->
          <MiniMap position="bottom-left" pannable zoomable />
        </VueFlow>

        <!-- Hint -->
        <div class="pointer-events-none absolute bottom-3 right-32 rounded-lg bg-slate-900/80 px-2.5 py-1 text-[10px] text-slate-500 backdrop-blur">
          Double-click folder to expand · Click bookmark to inspect · Scroll to zoom
        </div>
      </div>

      <!-- Inspector sidebar -->
      <BookmarkInspector
        v-if="showInspector"
        :bookmark="selectedBookmark"
        @edit="requestEdit"
        @delete="onDelete"
        @move="onMove"
        @close="closeInspector"
      />
    </div>

    <!-- Error toast -->
    <div v-if="errorMessage" class="absolute bottom-4 left-1/2 -translate-x-1/2 rounded-lg border border-rose-500/30 bg-rose-950 px-4 py-2 text-xs text-rose-200 shadow-xl">
      {{ errorMessage }}
    </div>

    <!-- Edit modal -->
    <BookmarkEditDialog :bookmark="editing" @close="editing = null" @save="onSaveEdit" />
  </main>
</template>

<style>
@import '@vue-flow/core/dist/style.css';
@import '@vue-flow/core/dist/theme-default.css';
@import '@vue-flow/controls/dist/style.css';
@import '@vue-flow/minimap/dist/style.css';

/* Override Vue Flow defaults for dark theme */
.vue-flow {
  --vf-node-bg: transparent;
  --vf-node-text: #e2e8f0;
  --vf-connection-stroke: rgb(244 63 94 / 0.5);
  --vf-handle: #64748b;
}

.vue-flow__edge-path {
  stroke: rgb(71 85 105 / 0.6);
  stroke-width: 2;
}

.vue-flow__edge.animated .vue-flow__edge-path {
  stroke: rgb(251 191 36 / 0.4);
}

.vue-flow__minimap {
  background: rgb(15 23 42 / 0.9);
  border: 1px solid rgb(51 65 85 / 0.5);
  border-radius: 0.75rem;
}

.vue-flow__controls {
  background: rgb(15 23 42 / 0.9);
  border: 1px solid rgb(51 65 85 / 0.5);
  border-radius: 0.75rem;
  overflow: hidden;
}

.vue-flow__controls-button {
  background: transparent;
  border-bottom-color: rgb(51 65 85 / 0.3);
  fill: #94a3b8;
}

.vue-flow__controls-button:hover {
  background: rgb(30 41 59);
  fill: #f1f5f9;
}

.vue-flow__background pattern circle {
  fill: rgb(51 65 85 / 0.4);
}
</style>
