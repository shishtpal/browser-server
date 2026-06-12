<template>
  <Modal :open="!!bookmark" title="Edit bookmark" description="Update the saved link details." @close="emit('close')">
    <div v-if="bookmark" :key="bookmark.id" class="grid gap-3">
      <input v-model="localForm.title" type="text" placeholder="Title" required class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
      <input v-model="localForm.url" type="url" placeholder="URL" required class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
      <input v-model="localForm.description" type="text" placeholder="Description" class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
      <input v-model="localForm.tagsStr" type="text" placeholder="Tags: comma, separated" class="rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30" />
    </div>
    <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
      <button type="button" @click="emit('close')" class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-black text-slate-700 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600">Cancel</button>
      <button type="button" @click="onSave" class="rounded-lg bg-gradient-to-r from-cyan-500 to-blue-600 px-4 py-2 text-sm font-black text-white shadow-lg shadow-cyan-500/20">Save changes</button>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import Modal from '../ui/Modal.vue'
import type { BookmarkResponse } from '../../types'

interface Props {
  bookmark: BookmarkResponse | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
  save: [data: { title: string; url: string; description: string; tagsStr: string }]
}>()

const localForm = ref({ title: '', url: '', description: '', tagsStr: '' })

watch(() => props.bookmark, (b) => {
  if (b) {
    localForm.value = {
      title: b.title,
      url: b.url,
      description: b.description,
      tagsStr: b.tags.join(', '),
    }
  }
}, { immediate: true })

const onSave = () => {
  emit('save', { ...localForm.value })
}
</script>
