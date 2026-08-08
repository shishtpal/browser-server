<template>
  <article
    class="group overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:-translate-y-0.5 hover:shadow-lg dark:border-white/10 dark:bg-slate-800/90"
  >
    <div class="relative">
      <button
        type="button"
        class="block w-full bg-gray-100 dark:bg-slate-900"
        :aria-label="`View image: ${image.prompt}`"
        @click="$emit('open', image)"
      >
        <img
          :src="url"
          :alt="image.prompt"
          loading="lazy"
          class="aspect-square w-full object-contain"
        />
      </button>
      <div
        class="pointer-events-none absolute inset-x-0 bottom-0 flex justify-end gap-1 bg-gradient-to-t from-slate-900/80 to-transparent p-2 opacity-0 transition group-hover:pointer-events-auto group-hover:opacity-100"
      >
        <button v-if="canEdit" type="button" :class="overlayButton" @click="$emit('edit', image)">
          Edit
        </button>
        <button type="button" :class="overlayButton" @click="$emit('reuse', image)">Reuse</button>
      </div>
    </div>
    <div class="p-3">
      <p
        class="line-clamp-2 text-xs font-semibold text-slate-700 dark:text-slate-200"
        :title="image.prompt"
      >
        {{ image.prompt }}
      </p>
      <div
        class="mt-2 flex flex-wrap items-center gap-1 text-[10px] font-bold text-slate-400 dark:text-slate-500"
      >
        <span class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-slate-700">
          {{ image.image_size }}
        </span>
        <span>{{ formatBytes(image.size_bytes) }}</span>
        <span>·</span>
        <span>{{ formatImageDate(image.created_at) }}</span>
      </div>
      <div class="mt-3 flex items-center gap-2">
        <a
          :href="url"
          :download="image.filename"
          class="text-xs font-black text-violet-600 underline decoration-violet-300 underline-offset-4 transition hover:text-violet-700 dark:text-violet-400 dark:decoration-violet-800"
        >
          Download
        </a>
        <button
          type="button"
          class="ml-auto text-xs font-black text-red-600 transition hover:text-red-700 dark:text-red-400"
          @click="$emit('delete', image)"
        >
          Delete
        </button>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import type { GeneratedImage } from '@browser-server/shared-types'
import { computed } from 'vue'
import { getGeneratedImageUrl } from '../../lib/api/ai'
import { formatBytes, formatImageDate } from './format'

const props = defineProps<{
  image: GeneratedImage
  canEdit: boolean
}>()

defineEmits<{
  open: [image: GeneratedImage]
  edit: [image: GeneratedImage]
  reuse: [image: GeneratedImage]
  delete: [image: GeneratedImage]
}>()

const url = computed(() => getGeneratedImageUrl(props.image.id))

const overlayButton =
  'rounded-md bg-white/95 px-2 py-1 text-[10px] font-black text-slate-800 shadow transition hover:bg-white'
</script>
