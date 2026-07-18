<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import type { BookmarkNodeData } from './buildGraph'

const props = defineProps<{
  data: BookmarkNodeData
}>()

const emit = defineEmits<{
  select: [bookmark: typeof props.data.bookmark]
}>()
</script>

<template>
  <div
    class="group relative flex min-w-[180px] max-w-[260px] cursor-pointer items-center gap-2.5 rounded-xl border border-slate-700/60 bg-slate-900/90 px-3.5 py-2.5 shadow-md transition-all duration-200 hover:border-rose-500/40 hover:shadow-rose-500/10"
    @dblclick="emit('select', data.bookmark)"
  >
    <Handle type="target" :position="Position.Left" class="!-left-1 !h-2 !w-2 !border-slate-600 !bg-slate-500" />

    <!-- Favicon -->
    <img
      :src="data.faviconUrl"
      alt=""
      class="h-5 w-5 shrink-0 rounded-sm"
      @error="($event.target as HTMLImageElement).style.visibility = 'hidden'"
    />

    <!-- Title -->
    <div class="min-w-0 flex-1">
      <p class="truncate text-xs font-medium text-slate-200 group-hover:text-white">{{ data.label }}</p>
      <p class="truncate text-[10px] text-slate-500">{{ data.bookmark.url }}</p>
    </div>

    <!-- Tags indicator -->
    <div v-if="data.bookmark.tags.length > 0" class="flex shrink-0 items-center">
      <span class="rounded-full bg-rose-500/15 px-1.5 py-0.5 text-[9px] font-medium text-rose-400">
        {{ data.bookmark.tags.length }}
      </span>
    </div>
  </div>
</template>
