<script setup lang="ts">
import type { BookmarkResponse } from '@browser-server/shared-client'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useBookmarksGraph } from './useBookmarksGraph'
import BookmarksGraphCanvas from './BookmarksGraphCanvas.vue'
import BookmarkInspector from './BookmarkInspector.vue'
import BookmarkEditDialog from './BookmarkEditDialog.vue'

const {
  isReady,
  isLoading,
  errorMessage,
  bookmarks,
  allTags,
  layout,
  selection,
  searchQuery,
  activeTag,
  load,
  refresh,
  select,
  clearSelection,
  updateBookmark,
  moveBookmark,
  deleteBookmark,
  cleanup,
} = useBookmarksGraph()

const canvas = ref<InstanceType<typeof BookmarksGraphCanvas> | null>(null)
const editing = ref<BookmarkResponse | null>(null)

const selectedNode = computed(() => selection.value.node)
const selectedId = computed(() => selection.value.node?.node.id ?? null)
const nodeCount = computed(() => layout.value.nodes.filter((n) => n.node.type === 'bookmark').length)

function onSelect(node: typeof selectedNode.value) {
  if (node) select(node)
}

function onBackground() {
  clearSelection()
}

async function onDropBookmark(bookmarkId: number, folderPath: string) {
  await moveBookmark(bookmarkId, folderPath)
}

function requestEdit() {
  const b = selectedNode.value?.node.bookmark
  if (b) editing.value = b
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
  const b = selectedNode.value?.node.bookmark
  if (!b) return
  if (!confirm('Delete this bookmark?')) return
  await deleteBookmark(b.id)
  clearSelection()
}

async function onMoveFromInspector() {
  const sel = selection.value
  if (sel.kind === 'bookmark' && sel.node?.node.bookmark) {
    const target = prompt('Move bookmark to folder path (leave blank for Unfiled):', sel.node.node.path || '')
    if (target === null) return
    await moveBookmark(sel.node.node.bookmark.id, target.trim())
  } else if (sel.kind === 'folder' && sel.node) {
    const target = prompt('Enter bookmark id to move into this folder:', '')
    if (!target) return
    const id = Number.parseInt(target, 10)
    if (Number.isNaN(id)) return
    await moveBookmark(id, sel.node.node.path)
  }
}

function fit() {
  canvas.value?.fit()
}
function zoomIn() {
  canvas.value?.zoomIn()
}
function zoomOut() {
  canvas.value?.zoomOut()
}

onMounted(() => {
  if (isReady.value) void load().then(() => nextTick(fit))
})

watch(isReady, (ready) => {
  if (ready) void load().then(() => nextTick(fit))
})

watch(
  () => bookmarks.value.length,
  (count, prev) => {
    if (count > 0 && (prev === 0 || prev === undefined)) void nextTick(fit)
  },
)

onBeforeUnmount(() => {
  cleanup()
})
</script>

<template>
  <main class="flex h-screen min-w-[760px] flex-col overflow-hidden bg-slate-950 text-slate-100">
    <header class="flex h-14 shrink-0 items-center justify-between border-b border-slate-800 px-5">
      <div class="min-w-0">
        <h1 class="text-base font-semibold">Bookmarks Graph</h1>
        <p class="text-xs text-slate-500">Mind-map view of bookmarks grouped by folder</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="hidden items-center gap-1 rounded-lg border border-slate-700 px-1 sm:flex">
          <button type="button" class="flex h-7 w-7 items-center justify-center rounded text-slate-400 hover:bg-slate-800 hover:text-white" title="Zoom out" @click="zoomOut">
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14" /></svg>
          </button>
          <button type="button" class="flex h-7 w-7 items-center justify-center rounded text-slate-400 hover:bg-slate-800 hover:text-white" title="Zoom in" @click="zoomIn">
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5v14M5 12h14" /></svg>
          </button>
          <button type="button" class="flex h-7 w-7 items-center justify-center rounded text-slate-400 hover:bg-slate-800 hover:text-white" title="Fit to screen" @click="fit">
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 9V5a1 1 0 0 1 1-1h4M20 9V5a1 1 0 0 0-1-1h-4M4 15v4a1 1 0 0 0 1 1h4M20 15v4a1 1 0 0 1-1 1h-4" /></svg>
          </button>
        </div>
        <button type="button" class="rounded-lg border border-slate-700 px-3 py-1.5 text-xs font-medium text-slate-300 transition hover:border-slate-500 hover:text-white" @click="refresh">
          Refresh
        </button>
      </div>
    </header>

    <div v-if="!isReady" class="flex flex-1 items-center justify-center text-sm text-slate-400">
      Configure a server URL, API token, and user ID in extension settings first.
    </div>

    <div v-else class="flex min-h-0 flex-1 overflow-hidden">
      <div class="relative min-w-0 flex-1 overflow-hidden">
        <div class="pointer-events-none absolute left-4 top-4 z-10 flex flex-col gap-2">
          <div class="pointer-events-auto flex w-64 items-center gap-2 rounded-lg border border-slate-700 bg-slate-900/90 px-2 py-1.5 backdrop-blur">
            <svg class="h-3.5 w-3.5 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              v-model="searchQuery"
              type="search"
              placeholder="Filter bookmarks…"
              class="w-full bg-transparent text-xs text-slate-100 outline-none placeholder:text-slate-600"
            />
          </div>
          <div v-if="allTags.length" class="pointer-events-auto flex max-w-64 flex-wrap gap-1 rounded-lg border border-slate-700 bg-slate-900/90 px-2 py-1.5 backdrop-blur">
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

        <div v-if="isLoading && bookmarks.length === 0" class="absolute inset-0 flex items-center justify-center text-sm text-slate-500">
          Loading bookmarks…
        </div>
        <div v-else-if="bookmarks.length === 0" class="absolute inset-0 flex flex-col items-center justify-center gap-2 text-sm text-slate-500">
          <svg class="h-8 w-8 text-slate-700" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v16z" /></svg>
          <p>No bookmarks yet.</p>
        </div>
        <div v-else-if="nodeCount === 0" class="absolute inset-0 flex items-center justify-center text-sm text-slate-500">
          No bookmarks match the current filter.
        </div>

        <BookmarksGraphCanvas
          v-else
          ref="canvas"
          :layout="layout"
          :selected-id="selectedId"
          @select="onSelect"
          @background="onBackground"
          @drop-bookmark="onDropBookmark"
        />

        <div class="pointer-events-none absolute bottom-3 left-3 rounded-lg bg-slate-900/80 px-2.5 py-1 text-[10px] text-slate-500 backdrop-blur">
          Drag background to pan · wheel to zoom · drag a bookmark onto a folder to move it
        </div>
      </div>

      <BookmarkInspector
        :node="selectedNode"
        @edit="requestEdit"
        @delete="onDelete"
        @move="onMoveFromInspector"
        @close="clearSelection"
      />
    </div>

    <div v-if="errorMessage" class="absolute bottom-4 left-1/2 -translate-x-1/2 rounded-lg border border-rose-500/30 bg-rose-950 px-4 py-2 text-xs text-rose-200 shadow-xl">
      {{ errorMessage }}
    </div>

    <BookmarkEditDialog :bookmark="editing" @close="editing = null" @save="onSaveEdit" />
  </main>
</template>
