<script setup lang="ts">
import type { BookmarkResponse } from '@browser-server/shared-client'
import { faviconUrl, timeAgo } from '@browser-server/shared-utils'
import { getBrowserApi } from '../browserApi'

defineProps<{
  bookmark: BookmarkResponse | null
}>()

const emit = defineEmits<{
  edit: []
  delete: []
  move: []
  close: []
}>()

function openUrl(url: string) {
  void getBrowserApi().tabs.create({ url, active: true })
}
</script>

<template>
  <aside class="flex w-80 shrink-0 flex-col overflow-hidden border-l border-slate-800/80 bg-slate-900/70 backdrop-blur-sm">
    <!-- Header -->
    <div class="flex h-14 shrink-0 items-center justify-between border-b border-slate-800/60 px-4">
      <h2 class="text-xs font-semibold uppercase tracking-wider text-slate-400">Inspector</h2>
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

    <!-- Empty state -->
    <div v-if="!bookmark" class="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
      <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-800/80 text-slate-600">
        <svg class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v16z" />
        </svg>
      </div>
      <p class="text-sm font-medium text-slate-300">Select a bookmark</p>
      <p class="max-w-[200px] text-xs leading-5 text-slate-600">Click or double-click a bookmark node to view its details.</p>
    </div>

    <!-- Bookmark details -->
    <div v-else class="flex min-h-0 flex-1 flex-col overflow-y-auto p-4">
      <!-- Header with favicon -->
      <div class="flex items-start gap-3">
        <img
          :src="faviconUrl(bookmark.url)"
          alt=""
          class="mt-0.5 h-6 w-6 shrink-0 rounded-md"
          @error="($event.target as HTMLImageElement).style.visibility = 'hidden'"
        />
        <div class="min-w-0">
          <p class="break-words text-sm font-semibold leading-tight text-slate-100">{{ bookmark.title }}</p>
          <a
            :href="bookmark.url"
            class="mt-1 block break-all text-[11px] text-sky-400/80 transition hover:text-sky-300"
            @click.prevent="openUrl(bookmark.url)"
          >{{ bookmark.url }}</a>
        </div>
      </div>

      <!-- Details -->
      <dl class="mt-5 space-y-3 text-xs">
        <div v-if="bookmark.description">
          <dt class="font-medium text-slate-500">Description</dt>
          <dd class="mt-1 text-slate-300">{{ bookmark.description }}</dd>
        </div>
        <div v-if="bookmark.folder_path">
          <dt class="font-medium text-slate-500">Folder</dt>
          <dd class="mt-1 rounded-md bg-slate-800/60 px-2 py-1 font-mono text-slate-300">{{ bookmark.folder_path }}</dd>
        </div>
        <div v-if="bookmark.tags.length">
          <dt class="font-medium text-slate-500">Tags</dt>
          <dd class="mt-1.5 flex flex-wrap gap-1">
            <span
              v-for="tag in bookmark.tags"
              :key="tag"
              class="rounded-full bg-rose-500/10 px-2 py-0.5 text-[10px] font-medium text-rose-300 ring-1 ring-inset ring-rose-500/20"
            >#{{ tag }}</span>
          </dd>
        </div>
        <div>
          <dt class="font-medium text-slate-500">Saved</dt>
          <dd class="mt-1 text-slate-400">{{ timeAgo(bookmark.created_at) }}</dd>
        </div>
      </dl>

      <!-- Actions -->
      <div class="mt-6 grid grid-cols-2 gap-2">
        <button
          type="button"
          class="col-span-2 rounded-xl bg-rose-500 px-3 py-2.5 text-xs font-semibold text-white shadow-md shadow-rose-500/20 transition hover:bg-rose-400"
          @click="openUrl(bookmark.url)"
        >
          Open in new tab
        </button>
        <button
          type="button"
          class="rounded-xl border border-slate-700/80 px-3 py-2 text-xs font-medium text-slate-300 transition hover:bg-slate-800"
          @click="emit('edit')"
        >
          Edit
        </button>
        <button
          type="button"
          class="rounded-xl border border-slate-700/80 px-3 py-2 text-xs font-medium text-slate-300 transition hover:bg-slate-800"
          @click="emit('move')"
        >
          Move
        </button>
        <button
          type="button"
          class="col-span-2 rounded-xl border border-rose-800/40 px-3 py-2 text-xs font-medium text-rose-300 transition hover:bg-rose-500/10"
          @click="emit('delete')"
        >
          Delete bookmark
        </button>
      </div>
    </div>
  </aside>
</template>
