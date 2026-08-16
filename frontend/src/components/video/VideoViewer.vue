<template>
  <Modal
    :open="!!video"
    :title="video ? `Video ${index + 1} of ${total}` : ''"
    fullscreen
    @close="$emit('close')"
  >
    <div v-if="video" class="flex h-full min-h-0">
      <!-- Player -->
      <div class="flex min-w-0 flex-1 items-center justify-center bg-black p-4">
        <video
          v-if="video.status === 'completed' && url"
          :src="url"
          controls
          autoplay
          class="max-h-full max-w-full rounded-lg"
        />
        <div v-else class="text-center text-slate-400">
          <LoaderCircle class="mx-auto h-8 w-8 animate-spin" aria-hidden="true" />
          <p class="mt-2 text-xs font-bold">
            {{ video.status === 'failed' ? 'Generation failed' : 'Video is still rendering…' }}
          </p>
          <p v-if="video.error_message" class="mt-1 px-6 text-[11px] text-red-400">
            {{ video.error_message }}
          </p>
        </div>
      </div>

      <!-- Metadata panel -->
      <aside
        class="flex w-72 shrink-0 flex-col gap-3 overflow-y-auto border-l border-gray-200 bg-slate-100 p-3 text-slate-900 dark:border-white/10 dark:bg-white/5 dark:text-white"
      >
        <p
          class="text-[10px] font-black tracking-wider text-slate-400 uppercase dark:text-white/50"
        >
          Prompt
        </p>
        <p
          class="text-xs font-semibold break-words whitespace-pre-wrap text-slate-800 dark:text-white/90"
        >
          {{ video.prompt }}
        </p>

        <div class="flex flex-wrap gap-1 text-[10px] font-bold text-slate-500 dark:text-white/60">
          <span class="rounded bg-slate-200 px-2 py-1 dark:bg-white/10">{{ video.provider }}</span>
          <span class="rounded bg-slate-200 px-2 py-1 dark:bg-white/10">{{ video.model }}</span>
          <span v-if="video.size" class="rounded bg-slate-200 px-2 py-1 dark:bg-white/10">{{
            video.size
          }}</span>
          <span class="rounded bg-slate-200 px-2 py-1 dark:bg-white/10">{{
            formatDuration(video.seconds)
          }}</span>
          <span class="rounded bg-slate-200 px-2 py-1 dark:bg-white/10">{{
            formatBytes(video.size_bytes)
          }}</span>
          <span class="rounded bg-slate-200 px-2 py-1 dark:bg-white/10">{{
            formatVideoDate(video.created_at)
          }}</span>
        </div>

        <div class="mt-auto grid gap-2">
          <Button variant="secondary" size="sm" @click="$emit('reuse', video)">Reuse prompt</Button>
          <a
            v-if="video.status === 'completed' && url"
            :href="url"
            :download="video.filename || 'video.mp4'"
            class="inline-flex items-center justify-center gap-1.5 rounded-lg bg-slate-200 px-3 py-1.5 text-center text-xs font-black text-slate-800 transition hover:bg-slate-300 dark:bg-white/10 dark:text-white dark:hover:bg-white/20"
          >
            <Download class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
            Download
          </a>
          <Button variant="ghost" size="sm" @click="$emit('delete', video)">
            <span class="inline-flex items-center gap-1.5 text-red-600 dark:text-red-400">
              <Trash2 class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
              Delete
            </span>
          </Button>
        </div>
      </aside>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import type { GeneratedVideo } from '@browser-server/shared-types';
import { computed } from 'vue';
import { Download, LoaderCircle, Trash2 } from '@lucide/vue';
import { getGeneratedVideoUrl } from '../../lib/api/ai';
import Button from '../ui/Button.vue';
import Modal from '../ui/Modal.vue';
import { formatBytes, formatDuration, formatVideoDate } from './format';

const props = defineProps<{
  video: GeneratedVideo | null;
  index: number;
  total: number;
}>();

defineEmits<{
  close: [];
  step: [delta: number];
  reuse: [video: GeneratedVideo];
  delete: [video: GeneratedVideo];
}>();

const url = computed(() =>
  props.video && props.video.status === 'completed' ? getGeneratedVideoUrl(props.video.id) : '',
);
</script>
