<script setup lang="ts">
import type { PositionedNode } from './buildGraph'
import { faviconUrl, timeAgo } from '@browser-server/shared-utils'
import { computed } from 'vue'
import { getBrowserApi } from '../browserApi'

const props = defineProps<{
  node: PositionedNode | null
}>()

const emit = defineEmits<{
  edit: []
  delete: []
  move: []
  close: []
}>()

const bookmark = computed(() => props.node?.node.bookmark ?? null)
const isFolder = computed(() => props.node?.node.type === 'folder')
const isBookmark = computed(() => props.node?.node.type === 'bookmark')
const folderLeafCount = computed(() => props.node?.node.leafCount ?? 0)

function openUrl(url: string) {
  void getBrowserApi().tabs.create({ url, active: true })
}
</script>

<template>
  <aside class="flex w-80 shrink-0 flex-col overflow-hidden border-l border-slate-800 bg-slate-900/60">
    <div class="flex h-14 shrink-0 items-center justify-between border-b border-slate-800 px-4">
      <h2 class="text-xs font-semibold uppercase tracking-wider text-slate-400">Details</h2>
      <button
        type="button"
        class="flex h-7 w-7 items-center justify-center rounded-lg text-slate-500 transition hover:bg-slate-800 hover:text-slate-200"
        title="Close"
        @click="emit('close')"
      >
        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M18 6 6 18M6 6l12 12" />
        </svg>
      </button>
    </div>

    <div v-if="!node" class="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
      <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-800 text-slate-600">
        <svg class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v16z" />
        </svg>
      </div>
      <p class="text-sm font-medium text-slate-300">Select a node</p>
      <p class="max-w-[220px] text-xs leading-5 text-slate-600">Click a bookmark or folder to inspect, edit, move, or delete it.</p>
    </div>

    <div v-else class="flex min-h-0 flex-1 flex-col overflow-y-auto p-4">
      <!-- Bookmark details -->
      <template v-if="isBookmark && bookmark">
        <div class="flex items-start gap-3">
          <img
            :src="faviconUrl(bookmark.url)"
            alt=""
            class="mt-0.5 h-5 w-5 shrink-0 rounded-sm"
            @error="($event.target as HTMLImageElement).style.visibility = 'hidden'"
          />
          <div class="min-w-0">
            <p class="break-words text-sm font-semibold text-slate-100">{{ bookmark.title }}</p>
            <p class="mt-0.5 break-all text-[11px] text-sky-400/80">{{ bookmark.url }}</p>
          </div>
        </div>

        <dl class="mt-4 space-y-3 text-xs">
          <div v-if="bookmark.description">
            <dt class="text-slate-500">Description</dt>
            <dd class="mt-1 text-slate-300">{{ bookmark.description }}</dd>
          </div>
          <div v-if="bookmark.folder_path">
            <dt class="text-slate-500">Folder</dt>
            <dd class="mt-1 font-mono text-slate-300">{{ bookmark.folder_path }}</dd>
          </div>
          <div v-if="bookmark.tags.length">
            <dt class="text-slate-500">Tags</dt>
            <dd class="mt-1 flex flex-wrap gap-1">
              <span
                v-for="tag in bookmark.tags"
                :key="tag"
                class="rounded-full bg-slate-800 px-2 py-0.5 text-[10px] font-medium text-slate-300"
              >#{{ tag }}</span>
            </dd>
          </div>
          <div>
            <dt class="text-slate-500">Saved</dt>
            <dd class="mt-1 text-slate-400">{{ timeAgo(bookmark.created_at) }}</dd>
          </div>
        </dl>

        <div class="mt-5 grid grid-cols-2 gap-2">
          <button
            type="button"
            class="col-span-2 rounded-lg bg-rose-500 px-3 py-2 text-xs font-semibold text-white transition hover:bg-rose-400"
            @click="openUrl(bookmark.url)"
          >Open in tab</button>
          <button
            type="button"
            class="rounded-lg border border-slate-700 px-3 py-2 text-xs font-medium text-slate-300 transition hover:bg-slate-800"
            @click="emit('edit')"
          >Edit</button>
          <button
            type="button"
            class="rounded-lg border border-slate-700 px-3 py-2 text-xs font-medium text-slate-300 transition hover:bg-slate-800"
            @click="emit('move')"
          >Move</button>
          <button
            type="button"
            class="col-span-2 rounded-lg border border-rose-800/60 px-3 py-2 text-xs font-medium text-rose-300 transition hover:bg-rose-500/10"
            @click="emit('delete')"
          >Delete bookmark</button>
        </div>
      </template>

      <!-- Folder details -->
      <template v-else-if="isFolder">
        <div class="flex items-center gap-2">
          <svg class="h-5 w-5 shrink-0 text-amber-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z" />
          </svg>
          <p class="truncate text-sm font-semibold text-slate-100">{{ node.node.name }}</p>
        </div>
        <p class="mt-3 text-xs text-slate-500">
          <span class="font-mono text-slate-300">{{ node.node.path || 'Unfiled' }}</span>
        </p>
        <p class="mt-2 text-xs text-slate-400">{{ folderLeafCount }} bookmark{{ folderLeafCount === 1 ? '' : 's' }}</p>

        <div class="mt-5 grid grid-cols-2 gap-2">
          <button
            type="button"
            class="col-span-2 rounded-lg border border-slate-700 px-3 py-2 text-xs font-medium text-slate-300 transition hover:bg-slate-800"
            @click="emit('move')"
          >Move bookmark here</button>
        </div>
      </template>

      <template v-else>
        <p class="text-sm text-slate-400">Root node — no details.</p>
      </template>
    </div>
  </aside>
</template>
