<template>
  <article
    class="group overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:-translate-y-0.5 hover:shadow-lg dark:border-white/10 dark:bg-slate-800/90"
  >
    <div class="relative bg-slate-900">
      <!-- Completed: playable preview -->
      <video
        v-if="video.status === 'completed' && url"
        :src="url"
        controls
        preload="metadata"
        class="aspect-video w-full bg-black"
      />

      <!-- In flight / failed placeholder -->
      <div
        v-else
        class="grid aspect-video w-full place-items-center bg-gradient-to-br from-slate-800 to-slate-950 p-4"
      >
        <div v-if="video.status === 'failed'" class="text-center">
          <p class="text-xs font-black tracking-wider text-red-400 uppercase">Generation failed</p>
          <p class="mt-1 line-clamp-3 px-2 text-[11px] text-slate-300">
            {{ video.error_message || 'The provider did not return a result.' }}
          </p>
        </div>
        <div v-else class="flex flex-col items-center gap-2 text-center">
          <LoaderCircle class="h-7 w-7 animate-spin text-cyan-400" aria-hidden="true" />
          <p class="text-[11px] font-bold text-slate-300">
            {{ video.status === 'queued' ? 'Queued…' : 'Generating…' }}
          </p>
          <div class="h-1.5 w-40 overflow-hidden rounded-full bg-slate-700">
            <div
              class="h-full bg-cyan-400 transition-all"
              :style="{ width: `${Math.max(4, video.progress)}%` }"
            />
          </div>
          <p class="text-[10px] text-slate-400 tabular-nums">{{ video.progress }}%</p>
        </div>
      </div>

      <!-- Status badge (pointer-events-none: sits above <video controls>) -->
      <span
        class="pointer-events-none absolute top-2 left-2 rounded px-1.5 py-0.5 text-[10px] font-black uppercase"
        :class="statusBadgeClass(video.status)"
      >
        {{ videoStatusLabels[video.status] }}
      </span>
    </div>

    <div class="p-3">
      <button
        type="button"
        class="block w-full text-left"
        :title="`Reuse prompt: ${video.prompt}`"
        @click="$emit('reuse', video)"
      >
        <span
          class="line-clamp-2 text-xs font-semibold text-slate-700 underline decoration-transparent transition hover:decoration-slate-300 dark:text-slate-200 dark:hover:decoration-slate-600"
        >
          {{ video.prompt }}
        </span>
      </button>

      <div
        class="mt-2 flex flex-wrap items-center gap-1 text-[10px] font-bold text-slate-400 dark:text-slate-500"
      >
        <span
          class="rounded bg-cyan-100 px-1.5 py-0.5 text-cyan-700 dark:bg-cyan-900/40 dark:text-cyan-300"
        >
          {{ video.provider }}
        </span>
        <span
          class="rounded bg-gray-100 px-1.5 py-0.5 text-slate-600 dark:bg-slate-700 dark:text-slate-300"
        >
          {{ video.model }}
        </span>
        <template v-if="video.status === 'completed'">
          <span v-if="video.size" class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-slate-700">{{
            video.size
          }}</span>
          <span class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-slate-700">{{
            formatDuration(video.seconds)
          }}</span>
          <span>{{ formatBytes(video.size_bytes) }}</span>
        </template>
        <span>·</span>
        <span>{{ formatVideoDate(video.created_at) }}</span>
      </div>

      <div class="mt-3 flex items-center gap-2">
        <button
          type="button"
          class="inline-flex items-center gap-1 text-xs font-black text-cyan-600 transition hover:text-cyan-700 dark:text-cyan-400"
          @click="$emit('reuse', video)"
        >
          <Repeat class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          Reuse
        </button>
        <a
          v-if="video.status === 'completed' && url"
          :href="url"
          :download="video.filename || 'video.mp4'"
          class="inline-flex items-center gap-1 text-xs font-black text-cyan-600 underline decoration-cyan-300 underline-offset-4 transition hover:text-cyan-700 dark:text-cyan-400 dark:decoration-cyan-800"
        >
          <Download class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          Download
        </a>
        <button
          type="button"
          class="ml-auto inline-flex items-center gap-1 text-xs font-black text-red-600 transition hover:text-red-700 dark:text-red-400"
          @click="$emit('delete', video)"
        >
          <Trash2 class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          Delete
        </button>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import type { GeneratedVideo } from '@browser-server/shared-types';
import { computed } from 'vue';
import { Download, LoaderCircle, Repeat, Trash2 } from '@lucide/vue';
import { getGeneratedVideoUrl } from '../../lib/api/ai';
import {
  formatBytes,
  formatDuration,
  formatVideoDate,
  statusBadgeClass,
  videoStatusLabels,
} from './format';

const props = defineProps<{
  video: GeneratedVideo;
}>();

defineEmits<{
  open: [video: GeneratedVideo];
  reuse: [video: GeneratedVideo];
  delete: [video: GeneratedVideo];
}>();

const url = computed(() =>
  props.video.status === 'completed' ? getGeneratedVideoUrl(props.video.id) : '',
);
</script>
