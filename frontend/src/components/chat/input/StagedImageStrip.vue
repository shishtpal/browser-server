<template>
  <div v-if="items.length > 0" class="mb-2 flex flex-wrap gap-2">
    <div
      v-for="item in items"
      :key="item.id"
      class="group relative flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-2 py-1.5 pr-8 dark:border-white/10 dark:bg-slate-900"
    >
      <img :src="item.previewUrl" alt="" class="h-10 w-10 rounded-md object-cover" />
      <div class="flex min-w-0 flex-col">
        <span
          class="max-w-[10rem] truncate text-[0.72rem] font-medium text-slate-700 dark:text-slate-200"
        >
          {{ item.file.name }}
        </span>
        <span class="text-[0.65rem] text-slate-500 dark:text-slate-400">
          {{ formatBytes(item.file.size) }}
        </span>
      </div>

      <span v-if="item.uploading" class="absolute top-1/2 right-1.5 -translate-y-1/2">
        <LoaderCircle class="h-3.5 w-3.5 animate-spin text-indigo-500" aria-hidden="true" />
      </span>
      <button
        v-else
        type="button"
        class="absolute top-1/2 right-1 grid h-6 w-6 -translate-y-1/2 place-items-center rounded text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-500/10 dark:hover:text-red-400"
        title="Remove image"
        aria-label="Remove image"
        @click="$emit('remove', item.id)"
      >
        <X class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
      </button>

      <div
        v-if="item.error"
        class="absolute right-0 -bottom-1 left-0 translate-y-full rounded bg-red-100 px-1.5 py-0.5 text-[0.65rem] text-red-700 dark:bg-red-950/40 dark:text-red-200"
        role="alert"
      >
        {{ item.error }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { LoaderCircle, X } from '@lucide/vue';
import type { StagedImageInput } from '../ChatInput.vue';
import { formatBytes } from '../chatFormat';

defineProps<{ items: StagedImageInput[] }>();
defineEmits<{ remove: [id: string] }>();
</script>
