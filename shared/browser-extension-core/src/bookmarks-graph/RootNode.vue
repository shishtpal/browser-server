<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import type { RootNodeData } from './buildGraph'

defineProps<{
  data: RootNodeData
}>()
</script>

<template>
  <div class="flex items-center gap-3 rounded-2xl border-2 border-rose-500/50 bg-gradient-to-br from-slate-800 to-slate-900 px-5 py-3.5 shadow-xl shadow-rose-500/5">
    <!-- Bookmark icon -->
    <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-rose-500/15">
      <svg class="h-5 w-5 text-rose-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v16z" />
      </svg>
    </div>

    <div>
      <p class="text-sm font-bold text-white">{{ data.label }}</p>
      <p class="text-xs text-slate-400">{{ data.leafCount }} total</p>
    </div>

    <Handle type="source" :position="Position.Right" class="!-right-1 !h-3 !w-3 !border-rose-500 !bg-rose-500/80" />
  </div>
</template>
