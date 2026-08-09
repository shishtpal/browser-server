<template>
  <div v-if="visibleAttachments.length" class="mb-2 flex flex-wrap gap-2">
    <button
      v-for="att in visibleAttachments"
      :key="att.id"
      type="button"
      class="relative overflow-hidden rounded-lg border border-white/20 bg-white/10 p-0.5 transition hover:bg-white/20"
      :aria-label="`Open image ${att.filename}`"
      @click="previewAttachment = att"
    >
      <img
        :src="imageUrl(att)"
        :alt="att.filename"
        class="h-16 w-16 object-cover sm:h-20 sm:w-20"
        loading="lazy"
        @error="onImageError(att.id)"
      />
    </button>
  </div>

  <!-- Full-size preview -->
  <Teleport v-if="previewAttachment" to="body">
    <div
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      :aria-label="previewAttachment.filename"
      @click="closePreview"
      @keydown.esc="closePreview"
    >
      <div
        class="relative max-h-[90vh] max-w-[90vw] overflow-hidden rounded-lg bg-white shadow-2xl dark:bg-slate-900"
      >
        <img
          :src="imageUrl(previewAttachment)"
          :alt="previewAttachment.filename"
          class="max-h-[80vh] max-w-[85vw] object-contain"
          @error="onImageError(previewAttachment.id)"
        />
        <div
          class="absolute right-0 bottom-0 left-0 bg-gradient-to-t from-black/70 to-transparent px-4 py-3"
        >
          <p class="truncate text-sm font-medium text-white">{{ previewAttachment.filename }}</p>
          <p class="text-xs text-white/80">{{ formatBytes(previewAttachment.size_bytes) }}</p>
        </div>
        <button
          type="button"
          class="absolute top-2 right-2 grid h-9 w-9 place-items-center rounded-full bg-black/50 text-white transition hover:bg-black/70"
          aria-label="Close preview"
          @click.stop="closePreview"
        >
          <X class="h-5 w-5" :stroke-width="2.25" aria-hidden="true" />
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { X } from '@lucide/vue';
import type { AIImageAttachment } from '@browser-server/shared-types';
import { getAIImageAttachmentUrl } from '../../../lib/api';
import { formatBytes } from '../chatFormat';

const props = defineProps<{
  attachments: AIImageAttachment[];
  conversationId: string;
}>();

const previewAttachment = ref<AIImageAttachment | null>(null);
const brokenImageIds = ref(new Set<string>());

const visibleAttachments = computed(() =>
  props.attachments.filter((a) => !brokenImageIds.value.has(a.id)),
);

function imageUrl(att: AIImageAttachment): string {
  return getAIImageAttachmentUrl(props.conversationId, att.id);
}

function onImageError(id: string) {
  const next = new Set(brokenImageIds.value);
  next.add(id);
  brokenImageIds.value = next;
}

function closePreview() {
  previewAttachment.value = null;
}
</script>
