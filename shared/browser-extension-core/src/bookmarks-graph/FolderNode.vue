<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import type { FolderNodeData } from './buildGraph'

const props = defineProps<{
  id: string
  data: FolderNodeData
}>()

const emit = defineEmits<{
  toggle: [folderId: string]
}>()
</script>

<template>
  <div
    class="group relative flex min-w-[160px] cursor-pointer items-center gap-2.5 rounded-xl border border-slate-700/80 bg-gradient-to-br from-slate-800 to-slate-850 px-4 py-3 shadow-lg transition-all duration-200 hover:border-amber-500/50 hover:shadow-amber-500/10"
    :class="data.expanded ? 'ring-2 ring-amber-500/30 border-amber-500/40' : ''"
    @dblclick="emit('toggle', id)"
  >
    <Handle type="target" :position="Position.Left" class="!-left-1 !h-2 !w-2 !border-slate-600 !bg-slate-500" />

    <!-- Folder icon -->
    <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg" :class="data.expanded ? 'bg-amber-500/20' : 'bg-slate-700/60'">
      <svg class="h-4.5 w-4.5" :class="data.expanded ? 'text-amber-400' : 'text-amber-500/70'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
        <path v-if="data.expanded" d="M5 19a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4l2 2h4a2 2 0 0 1 2 2v1M5 19h14a2 2 0 0 0 2-2v-5a2 2 0 0 0-2-2H9a2 2 0 0 0-2 2v5a2 2 0 0 1-2 2Z" />
        <path v-else d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z" />
      </svg>
    </div>

    <!-- Label + count -->
    <div class="min-w-0 flex-1">
      <p class="truncate text-sm font-semibold text-slate-100">{{ data.label }}</p>
      <p class="text-xs text-slate-500">{{ data.leafCount }} bookmark{{ data.leafCount === 1 ? '' : 's' }}</p>
    </div>

    <!-- Expand indicator -->
    <div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full transition-transform duration-200" :class="data.expanded ? 'rotate-90' : ''">
      <svg class="h-3.5 w-3.5 text-slate-500 group-hover:text-slate-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="m9 18 6-6-6-6" />
      </svg>
    </div>

    <Handle type="source" :position="Position.Right" class="!-right-1 !h-2 !w-2 !border-slate-600 !bg-slate-500" />
  </div>
</template>
