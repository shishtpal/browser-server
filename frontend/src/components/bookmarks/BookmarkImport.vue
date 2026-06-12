<template>
  <div class="rounded-xl border border-dashed border-gray-200 bg-white/60 p-3 shadow-sm transition-colors dark:border-white/5 dark:bg-slate-800/50">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex items-center gap-3">
        <div class="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400">
          <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
          </svg>
        </div>
        <div>
          <p class="text-xs font-black text-slate-700 dark:text-slate-200">Import from Chrome</p>
          <p class="text-[10px] text-slate-500 dark:text-slate-400">Upload a bookmarks HTML export file</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <input
          ref="fileInputRef"
          type="file"
          accept=".html,.htm"
          @change="onFileChange"
          class="block w-full min-w-0 max-w-40 rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs file:mr-2 file:rounded-md file:border-0 file:bg-amber-50 file:px-2 file:py-0.5 file:text-[10px] file:font-black file:text-amber-700 file:transition hover:file:bg-amber-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:file:bg-amber-900/20 dark:file:text-amber-400"
        />
        <button
          type="button"
          @click="doImport"
          :disabled="!importFile || importing"
          class="shrink-0 rounded-lg px-3 py-1.5 text-xs font-black text-white shadow-sm transition disabled:cursor-not-allowed disabled:opacity-40"
          :class="importing ? 'bg-slate-400' : 'bg-gradient-to-r from-amber-500 to-orange-600 hover:-translate-y-0.5 hover:shadow-md'"
        >
          <span v-if="importing" class="flex items-center gap-1.5">
            <span class="inline-block h-3 w-3 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
            Importing...
          </span>
          <span v-else>Import</span>
        </button>
      </div>
    </div>
    <div v-if="importResult" class="mt-2 rounded-lg px-3 py-2 text-xs font-bold" :class="importResult.skipped > 0 ? 'bg-amber-50 text-amber-800 dark:bg-amber-900/20 dark:text-amber-300' : 'bg-emerald-50 text-emerald-800 dark:bg-emerald-900/20 dark:text-emerald-300'">
      Imported {{ importResult.imported }} bookmark{{ importResult.imported !== 1 ? 's' : '' }}<span v-if="importResult.skipped > 0">, {{ importResult.skipped }} duplicate{{ importResult.skipped !== 1 ? 's' : '' }} skipped</span>
    </div>
    <div v-if="importError" class="mt-2 rounded-lg bg-red-50 px-3 py-2 text-xs font-bold text-red-700 dark:bg-red-900/20 dark:text-red-400">
      {{ importError }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { importBookmarks } from '../../lib/api'
import type { ImportResult } from '../../types'

interface Props {
  selectedUserId: number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  imported: []
}>()

const fileInputRef = ref<HTMLInputElement | null>(null)
const importFile = ref<File | null>(null)
const importing = ref(false)
const importResult = ref<ImportResult | null>(null)
const importError = ref<string | null>(null)

const onFileChange = (e: Event) => {
  const input = e.target as HTMLInputElement
  importFile.value = input.files?.[0] || null
  importResult.value = null
  importError.value = null
}

const doImport = async () => {
  if (!importFile.value) return
  importing.value = true
  importResult.value = null
  importError.value = null
  try {
    const result = await importBookmarks(props.selectedUserId, importFile.value)
    importResult.value = result
    importFile.value = null
    if (fileInputRef.value) fileInputRef.value.value = ''
    emit('imported')
  } catch (e) {
    importError.value = e instanceof Error ? e.message : 'Import failed'
  } finally {
    importing.value = false
  }
}
</script>
