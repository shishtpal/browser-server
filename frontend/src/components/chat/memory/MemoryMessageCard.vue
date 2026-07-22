<template>
  <div
    class="group relative rounded-lg border px-3 py-2 transition"
    :class="messageBorderClass(message, isDeleteTarget ? message.id : null)"
  >
    <!-- Role badge + timestamp -->
    <div class="mb-1 flex items-center gap-2">
      <span
        class="rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide"
        :class="roleBadgeClass(message.role)"
      >{{ message.role }}</span>
      <span v-if="message.role === 'tool'" class="text-[10px] font-medium text-slate-600 dark:text-slate-300">{{ toolName }}</span>
      <span class="text-[10px] text-slate-400">{{ formatTime(message.created_at) }}</span>
      <span v-if="message.status !== 'completed'" class="text-[10px] italic text-slate-400">({{ message.status }})</span>
    </div>

    <!-- Edit mode -->
    <div v-if="editing" class="space-y-2">
      <textarea
        ref="editTextarea"
        :value="editContent"
        class="w-full rounded border border-slate-300 bg-white px-2 py-1.5 font-mono text-xs leading-relaxed outline-none focus:border-slate-500 dark:border-white/20 dark:bg-slate-800 dark:text-white"
        rows="6"
        @input="$emit('editInput', ($event.target as HTMLTextAreaElement).value)"
      ></textarea>
      <div class="flex gap-2">
        <Button variant="primary" size="sm" :loading="saving" loading-text="Saving…" @click="$emit('save')">Save</Button>
        <Button variant="ghost" size="sm" @click="$emit('cancelEdit')">Cancel</Button>
      </div>
    </div>

    <!-- Tool message display -->
    <MemoryToolContent v-else-if="message.role === 'tool'" :message="message" />

    <!-- Regular message display -->
    <div v-else>
      <pre class="max-h-40 overflow-auto whitespace-pre-wrap break-words text-xs leading-relaxed text-slate-700 dark:text-slate-300">{{ truncateContent(message.content) }}</pre>
    </div>

    <!-- Action buttons (shown on hover) -->
    <div class="absolute right-2 top-2 hidden gap-1 group-hover:flex">
      <button
        class="rounded p-1 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-white"
        title="Edit message"
        type="button"
        @click="$emit('edit')"
      >
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
      </button>
      <button
        v-if="!isDeleteTarget"
        class="rounded p-1 text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/30 dark:hover:text-red-400"
        title="Delete message"
        type="button"
        @click="$emit('confirmDelete')"
      >
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
      </button>
    </div>

    <!-- Inline delete confirmation -->
    <div v-if="isDeleteTarget" class="mt-2 flex items-center gap-2 rounded border border-red-200 bg-red-50 px-2.5 py-1.5 dark:border-red-900/40 dark:bg-red-950/20">
      <span class="flex-1 text-[11px] text-red-700 dark:text-red-300">Delete this {{ message.role }} message?</span>
      <Button variant="danger" size="sm" :loading="saving" loading-text="…" @click="$emit('delete')">Delete</Button>
      <Button variant="ghost" size="sm" @click="$emit('cancelDelete')">Cancel</Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AIMessage } from '@browser-server/shared-types'
import { computed } from 'vue'
import Button from '../../ui/Button.vue'
import MemoryToolContent from './MemoryToolContent.vue'
import { getToolName, formatTime, truncateContent, roleBadgeClass, messageBorderClass } from './memoryUtils'

const props = defineProps<{
  message: AIMessage
  editing: boolean
  editContent: string
  isDeleteTarget: boolean
  saving: boolean
}>()

defineEmits<{
  edit: []
  editInput: [value: string]
  save: []
  cancelEdit: []
  confirmDelete: []
  delete: []
  cancelDelete: []
}>()

const toolName = computed(() => getToolName(props.message))
</script>
