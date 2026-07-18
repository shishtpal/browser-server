<script setup lang="ts">
import { ref, watch } from 'vue'
import type { BookmarkResponse } from '@browser-server/shared-client'

interface EditPayload {
  title: string
  url: string
  description: string
  tags: string
  folderPath: string
}

const props = defineProps<{ bookmark: BookmarkResponse | null }>()

const emit = defineEmits<{
  close: []
  save: [bookmark: BookmarkResponse, payload: EditPayload]
}>()

const form = ref<EditPayload>({ title: '', url: '', description: '', tags: '', folderPath: '' })

watch(
  () => props.bookmark,
  (b) => {
    if (b) {
      form.value = {
        title: b.title,
        url: b.url,
        description: b.description,
        tags: b.tags.join(', '),
        folderPath: b.folder_path,
      }
    }
  },
  { immediate: true },
)

function onSave() {
  if (!props.bookmark) return
  if (!form.value.title.trim() || !form.value.url.trim()) return
  emit('save', props.bookmark, { ...form.value })
}
</script>

<template>
  <div
    v-if="bookmark"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
    @click.self="emit('close')"
  >
    <div class="w-full max-w-md rounded-2xl border border-slate-700/80 bg-slate-900 p-6 shadow-2xl">
      <h2 class="text-sm font-semibold text-slate-100">Edit bookmark</h2>
      <p class="mb-5 mt-1 text-xs text-slate-500">Update the saved link details and folder.</p>

      <div class="grid gap-3.5">
        <label class="grid gap-1.5 text-xs font-medium text-slate-400">
          Title
          <input
            v-model="form.title"
            type="text"
            placeholder="Title"
            class="rounded-xl border border-slate-700/80 bg-slate-800/80 px-3 py-2.5 text-sm text-slate-100 outline-none transition focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
          />
        </label>
        <label class="grid gap-1.5 text-xs font-medium text-slate-400">
          URL
          <input
            v-model="form.url"
            type="url"
            placeholder="https://"
            class="rounded-xl border border-slate-700/80 bg-slate-800/80 px-3 py-2.5 text-sm text-slate-100 outline-none transition focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
          />
        </label>
        <label class="grid gap-1.5 text-xs font-medium text-slate-400">
          Description
          <input
            v-model="form.description"
            type="text"
            placeholder="Optional"
            class="rounded-xl border border-slate-700/80 bg-slate-800/80 px-3 py-2.5 text-sm text-slate-100 outline-none transition focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
          />
        </label>
        <div class="grid grid-cols-2 gap-3">
          <label class="grid gap-1.5 text-xs font-medium text-slate-400">
            Tags
            <input
              v-model="form.tags"
              type="text"
              placeholder="comma, separated"
              class="rounded-xl border border-slate-700/80 bg-slate-800/80 px-3 py-2.5 text-sm text-slate-100 outline-none transition focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
            />
          </label>
          <label class="grid gap-1.5 text-xs font-medium text-slate-400">
            Folder path
            <input
              v-model="form.folderPath"
              type="text"
              placeholder="e.g. Dev/Go"
              class="rounded-xl border border-slate-700/80 bg-slate-800/80 px-3 py-2.5 text-sm text-slate-100 outline-none transition focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
            />
          </label>
        </div>
      </div>

      <div class="mt-6 flex justify-end gap-2">
        <button
          type="button"
          class="rounded-xl border border-slate-700/80 px-4 py-2.5 text-sm font-medium text-slate-300 transition hover:bg-slate-800"
          @click="emit('close')"
        >
          Cancel
        </button>
        <button
          type="button"
          class="rounded-xl bg-rose-500 px-4 py-2.5 text-sm font-semibold text-white shadow-md shadow-rose-500/20 transition hover:bg-rose-400"
          @click="onSave"
        >
          Save changes
        </button>
      </div>
    </div>
  </div>
</template>
