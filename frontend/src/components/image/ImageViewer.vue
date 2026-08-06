<template>
  <Modal
    :open="!!image"
    :title="image ? `Image ${index + 1} of ${total}` : ''"
    fullscreen
    @close="$emit('close')"
  >
    <div v-if="image" ref="containerRef" class="flex h-full min-h-0">
      <!-- Prompt panel -->
      <aside
        v-if="panelOpen"
        class="flex min-w-[200px] shrink-0 flex-col gap-3 overflow-y-auto rounded-l-xl bg-white/5 p-3 text-white"
        :style="{ width: `${sidebarWidth}px` }"
      >
        <div class="flex items-center justify-between gap-2">
          <p class="text-[10px] font-black uppercase tracking-wider text-white/50">Prompt</p>
          <button
            type="button"
            :class="chipButton"
            aria-label="Collapse prompt panel"
            @click="panelOpen = false"
          >
            Hide
          </button>
        </div>
        <p class="whitespace-pre-wrap break-words text-xs font-semibold text-white/90">
          {{ image.prompt }}
        </p>
        <div class="flex flex-wrap gap-1 text-[10px] font-bold text-white/60">
          <span class="rounded bg-white/10 px-2 py-1">{{ image.model }}</span>
          <span class="rounded bg-white/10 px-2 py-1">{{ image.image_size }}</span>
          <span class="rounded bg-white/10 px-2 py-1">{{ formatBytes(image.size_bytes) }}</span>
          <span class="rounded bg-white/10 px-2 py-1">{{ formatImageDate(image.created_at) }}</span>
        </div>
        <div class="mt-auto grid gap-2">
          <Button variant="secondary" size="sm" @click="$emit('reuse', image)">Reuse prompt</Button>
          <Button v-if="canEdit" variant="secondary" size="sm" @click="$emit('edit', image)">
            Edit image
          </Button>
          <a
            :href="url"
            :download="image.filename"
            class="rounded-lg bg-white/10 px-3 py-1.5 text-center text-xs font-black text-white transition hover:bg-white/20"
          >
            Download
          </a>
        </div>
      </aside>

      <!-- Resizer -->
      <div
        v-if="panelOpen"
        class="w-1 shrink-0 cursor-col-resize bg-white/10 transition hover:bg-violet-400"
        :class="{ '!bg-violet-500': isResizing }"
        @pointerdown="startResize"
        @pointermove="onResize"
        @pointerup="stopResize"
        @pointercancel="stopResize"
      ></div>

      <button
        v-else
        type="button"
        class="shrink-0 rounded-l-xl bg-white/5 px-2 text-[10px] font-black uppercase tracking-wider text-white/60 transition hover:bg-white/15"
        aria-label="Expand prompt panel"
        @click="panelOpen = true"
      >
        <span class="[writing-mode:vertical-rl]">Prompt</span>
      </button>

      <!-- Image panel -->
      <div class="flex min-w-0 flex-1 flex-col gap-2 pl-3">
        <div
          class="relative min-h-0 flex-1 overflow-hidden rounded-xl bg-black/40"
          @wheel.prevent="onWheel"
          @pointerdown="startPan"
          @pointermove="onPan"
          @pointerup="endPan"
          @pointercancel="endPan"
        >
          <img
            :src="url"
            :alt="image.prompt"
            draggable="false"
            class="absolute inset-0 h-full w-full select-none object-contain"
            :class="{ 'cursor-grab': canPan && !panning, 'cursor-grabbing': panning }"
            :style="transform"
          />
          <button
            type="button"
            :class="[navButton, 'left-2']"
            aria-label="Previous image"
            :disabled="index <= 0"
            @click="$emit('step', -1)"
          >
            &lsaquo;
          </button>
          <button
            type="button"
            :class="[navButton, 'right-2']"
            aria-label="Next image"
            :disabled="index >= total - 1"
            @click="$emit('step', 1)"
          >
            &rsaquo;
          </button>
        </div>
        <div class="flex items-center justify-center gap-2 text-white">
          <button
            type="button"
            :class="zoomButton"
            aria-label="Zoom out"
            @click="setZoom(zoom - 0.25)"
          >
            &minus;
          </button>
          <button
            type="button"
            class="min-w-16 rounded-lg bg-white/10 px-3 py-1.5 text-xs font-black tabular-nums transition hover:bg-white/20"
            @click="reset"
          >
            {{ Math.round(zoom * 100) }}%
          </button>
          <button
            type="button"
            :class="zoomButton"
            aria-label="Zoom in"
            @click="setZoom(zoom + 0.25)"
          >
            +
          </button>
        </div>
      </div>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import type { GeneratedImage } from "@browser-server/shared-types";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { getGeneratedImageUrl } from "../../lib/api/ai";
import { useImageZoom } from "../../composables/useImageZoom";
import { useResizableSidebar } from "../../composables/useResizableSidebar";
import Button from "../ui/Button.vue";
import Modal from "../ui/Modal.vue";
import { formatBytes, formatImageDate } from "./format";

const props = defineProps<{
  image: GeneratedImage | null;
  index: number;
  total: number;
  canEdit: boolean;
}>();

const emit = defineEmits<{
  close: [];
  step: [delta: number];
  reuse: [image: GeneratedImage];
  edit: [image: GeneratedImage];
}>();

const panelOpen = ref(true);

const { containerRef, sidebarWidth, isResizing, startResize, onResize, stopResize } =
  useResizableSidebar({ storageKey: "image.viewerPanelWidth", initial: 288, reserve: 320 });

const { zoom, panning, canPan, transform, reset, setZoom, onWheel, startPan, onPan, endPan } =
  useImageZoom();

const url = computed(() => (props.image ? getGeneratedImageUrl(props.image.id) : ""));

watch(() => props.image?.id, reset);

function onKeydown(e: KeyboardEvent) {
  if (!props.image) return;
  if (e.key === "ArrowLeft") emit("step", -1);
  else if (e.key === "ArrowRight") emit("step", 1);
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => window.removeEventListener("keydown", onKeydown));

const chipButton =
  "rounded bg-white/10 px-2 py-1 text-[10px] font-black text-white transition hover:bg-white/20";
const navButton =
  "absolute top-1/2 grid h-10 w-10 -translate-y-1/2 place-items-center rounded-full bg-white/10 text-xl font-black text-white transition hover:bg-white/25 disabled:opacity-25";
const zoomButton =
  "grid h-8 w-8 place-items-center rounded-lg bg-white/10 text-lg font-black transition hover:bg-white/20";
</script>
