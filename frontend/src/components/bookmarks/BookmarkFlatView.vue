<template>
  <div>
    <div class="hidden overflow-x-auto rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90 md:block">
      <table class="w-full table-fixed divide-y divide-gray-200 transition-colors dark:divide-slate-700">
        <colgroup>
          <col class="w-10" />
          <col class="w-[22%]" />
          <col class="w-[24%]" />
          <col class="w-[16%]" />
          <col class="w-[12%]" />
          <col class="w-[16%]" />
          <col class="w-28" />
        </colgroup>
        <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
          <tr>
            <th class="px-3 py-3"></th>
            <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Title</th>
            <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">URL</th>
            <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Description</th>
            <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Folder</th>
            <th class="px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Tags</th>
            <th class="w-28 px-3 py-3 text-right text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
          <tr v-for="b in bookmarks" :key="b.id" class="transition hover:bg-cyan-50/60 dark:hover:bg-cyan-900/20">
            <td class="px-3 py-3">
              <div class="grid h-8 w-8 place-items-center rounded-lg bg-gradient-to-br from-slate-900 to-slate-800 text-xs font-black text-white dark:from-slate-950 dark:to-slate-900">{{ getInitial(b.title) }}</div>
            </td>
            <td class="truncate px-3 py-3">
              <span class="block truncate text-sm font-black text-slate-900 transition-colors dark:text-white" :title="b.title">{{ b.title }}</span>
              <span class="block truncate text-[10px] text-cyan-600 dark:text-cyan-400">{{ formatHost(b.url) }}</span>
            </td>
            <td class="truncate px-3 py-3">
              <a :href="b.url" target="_blank" rel="noopener" class="block truncate text-sm font-semibold text-blue-600 transition-colors hover:underline dark:text-blue-400" :title="b.url">{{ b.url }}</a>
            </td>
            <td class="truncate px-3 py-3">
              <span class="block truncate text-sm text-slate-500 transition-colors dark:text-slate-400" :title="b.description">{{ b.description || '—' }}</span>
            </td>
            <td class="truncate px-3 py-3">
              <span class="block truncate text-xs font-semibold text-slate-400 transition-colors dark:text-slate-500" :title="b.folder_path">{{ b.folder_path || '—' }}</span>
            </td>
            <td class="px-3 py-3">
              <div class="flex flex-wrap gap-1">
                <button
                  v-for="tag in b.tags"
                  :key="tag"
                  type="button"
                  @click="emit('filterTag', tag)"
                  class="rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-black text-slate-600 transition hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400"
                >#{{ tag }}</button>
                <span v-if="!b.tags.length" class="text-[10px] text-slate-400 dark:text-slate-500">—</span>
              </div>
            </td>
            <td class="px-3 py-3 text-right">
              <div class="flex justify-end gap-1">
                <button type="button" @click="emit('edit', b)" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-slate-500 transition hover:bg-cyan-50 hover:text-cyan-700 dark:hover:bg-cyan-900/10 dark:hover:text-cyan-400">Edit</button>
                <button type="button" @click="emit('delete', b.id)" class="rounded-lg px-2.5 py-1.5 text-xs font-black text-red-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">Delete</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 md:hidden">
      <article v-for="b in bookmarks" :key="b.id" class="group flex flex-col overflow-hidden rounded-xl border border-gray-200/80 bg-white shadow-sm transition hover:-translate-y-0.5 hover:border-cyan-200 hover:shadow-md dark:border-slate-700/80 dark:bg-slate-800/90 dark:hover:border-cyan-500/30">
        <div class="flex items-start justify-between gap-3 border-b border-gray-100 bg-gradient-to-br from-slate-900 to-slate-800 p-3 text-white transition-colors dark:from-slate-950 dark:to-slate-900">
          <div class="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-white/10 text-sm font-black">{{ getInitial(b.title) }}</div>
          <div class="text-right">
            <h3 class="line-clamp-1 text-sm font-black">{{ b.title }}</h3>
            <a :href="b.url" target="_blank" rel="noopener" class="line-clamp-1 text-[10px] text-cyan-200 hover:text-cyan-50">{{ formatHost(b.url) }}</a>
          </div>
        </div>
        <div class="flex flex-1 flex-col p-3">
          <a :href="b.url" target="_blank" rel="noopener" class="truncate text-xs font-bold text-blue-600 transition-colors hover:underline dark:text-blue-400">{{ b.url }}</a>
          <p v-if="b.folder_path" class="mt-1.5 text-[10px] font-semibold text-slate-400 dark:text-slate-500">{{ b.folder_path }}</p>
          <p v-if="b.description" class="mt-1.5 line-clamp-2 text-xs leading-5 text-slate-500 transition-colors dark:text-slate-400">{{ b.description }}</p>
          <div v-if="b.tags.length" class="mt-auto flex flex-wrap gap-1 pt-3">
            <button
              v-for="tag in b.tags"
              :key="tag"
              type="button"
              @click="emit('filterTag', tag)"
              class="rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-black text-slate-600 transition hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400"
            >#{{ tag }}</button>
          </div>
          <div class="mt-3 flex gap-2 border-t border-gray-100 pt-3 transition-colors dark:border-slate-700/50">
            <button type="button" @click="emit('edit', b)" class="flex-1 rounded-lg bg-gray-100 px-3 py-2 text-xs font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600">Edit</button>
            <button type="button" @click="emit('delete', b.id)" class="flex-1 rounded-lg bg-red-50 px-3 py-2 text-xs font-black text-red-700 transition hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30">Delete</button>
          </div>
        </div>
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { BookmarkResponse } from '../../types'
import { getInitial, formatHost } from '../../composables/useBookmarks'

interface Props {
  bookmarks: BookmarkResponse[]
}

defineProps<Props>()

const emit = defineEmits<{
  edit: [bookmark: BookmarkResponse]
  delete: [id: number]
  filterTag: [tag: string]
}>()
</script>
