<template>
  <section
    class="overflow-hidden rounded-xl border border-violet-200 bg-violet-50/40 shadow-sm dark:border-violet-900/30 dark:bg-violet-900/10"
  >
    <header class="flex items-center gap-2 px-3 py-2">
      <button
        type="button"
        class="inline-flex min-w-0 flex-1 items-center gap-2 text-left"
        :aria-expanded="!collapsed"
        aria-label="Toggle prompt folder"
        @click="collapsed = !collapsed"
      >
        <component
          :is="collapsed ? Folder : FolderOpen"
          class="h-4 w-4 shrink-0 text-violet-500"
          :stroke-width="2.5"
          aria-hidden="true"
        />
        <span
          class="line-clamp-1 flex-1 text-sm font-bold text-slate-700 dark:text-slate-200"
          :title="group.prompt"
        >
          {{ group.prompt }}
        </span>
        <ChevronDown
          class="h-4 w-4 shrink-0 text-slate-400 transition-transform"
          :class="{ '-rotate-90': collapsed }"
          :stroke-width="2.5"
          aria-hidden="true"
        />
      </button>
      <div class="flex shrink-0 items-center gap-1.5">
        <span
          class="rounded-full bg-violet-200 px-2 py-0.5 text-[10px] font-black text-violet-800 dark:bg-violet-800/60 dark:text-violet-200"
        >
          {{ group.images.length }} images
        </span>
        <button
          type="button"
          class="inline-flex items-center gap-1 rounded-md bg-white px-2 py-1 text-[10px] font-black text-slate-700 shadow-sm transition hover:bg-slate-100 dark:bg-slate-700 dark:text-slate-100 dark:hover:bg-slate-600"
          title="Inject this prompt into the composer"
          @click="$emit('use-prompt', group)"
        >
          <Sparkles class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          Use prompt
        </button>
      </div>
    </header>

    <div
      v-show="!collapsed"
      class="border-t border-violet-200/70 px-3 py-3 dark:border-violet-900/30"
    >
      <div class="mb-2 flex flex-wrap gap-1.5 text-[10px] font-bold">
        <span
          v-if="group.providers > 1"
          class="rounded bg-white px-1.5 py-0.5 text-violet-700 dark:bg-slate-800 dark:text-violet-300"
        >
          {{ group.providers }} providers
        </span>
        <span
          class="rounded bg-white px-1.5 py-0.5 text-violet-700 dark:bg-slate-800 dark:text-violet-300"
        >
          {{ group.models }} model{{ group.models === 1 ? '' : 's' }}
        </span>
      </div>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <ImageCard
          v-for="image in group.images"
          :key="image.id"
          :image="image"
          :can-edit="canEdit"
          @open="(i) => $emit('open', i)"
          @edit="(i) => $emit('edit', i)"
          @reuse="(i) => $emit('reuse', i)"
          @delete="(i) => $emit('delete', i)"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { ImageGroup } from './composables/useImageGeneration';
import { computed } from 'vue';
import { useLocalStorage } from '@vueuse/core';
import { ChevronDown, Folder, FolderOpen, Sparkles } from '@lucide/vue';
import ImageCard from './ImageCard.vue';

const props = defineProps<{
  group: ImageGroup;
  canEdit: boolean;
}>();

const emit = defineEmits<{
  open: [image: import('@browser-server/shared-types').GeneratedImage];
  edit: [image: import('@browser-server/shared-types').GeneratedImage];
  reuse: [image: import('@browser-server/shared-types').GeneratedImage];
  delete: [image: import('@browser-server/shared-types').GeneratedImage];
  'use-prompt': [group: ImageGroup];
}>();

// Collapsed state is per-folder and persisted across reloads. Stored as an
// array of keys: a Set would not survive the JSON round-trip of localStorage.
const collapsedKeys = useLocalStorage<string[]>('bs.image.groupsCollapsed', []);
const collapsed = computed({
  get: () => collapsedKeys.value.includes(props.group.key),
  set: (v: boolean) => {
    collapsedKeys.value = v
      ? [...collapsedKeys.value, props.group.key]
      : collapsedKeys.value.filter((k) => k !== props.group.key);
  },
});
</script>
