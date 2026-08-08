<template>
  <form
    class="self-start rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-colors lg:sticky lg:top-4 dark:border-white/10 dark:bg-slate-800/90"
    @submit.prevent="$emit('submit')"
  >
    <div class="mb-1 flex items-baseline justify-between">
      <label
        for="image-prompt"
        class="text-xs font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
      >
        {{ sourceImage ? 'Edit instruction' : 'Prompt' }}
      </label>
      <span class="text-[10px] font-semibold text-slate-400 tabular-nums dark:text-slate-500">
        {{ prompt.length }}
      </span>
    </div>
    <textarea
      id="image-prompt"
      ref="textareaRef"
      :value="prompt"
      rows="6"
      :placeholder="
        sourceImage
          ? 'Describe the change, e.g. make the sky stormy'
          : 'Describe the image you want to create'
      "
      class="w-full resize-y rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:ring-4 focus:ring-violet-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30"
      @input="$emit('update:prompt', ($event.target as HTMLTextAreaElement).value)"
      @keydown.enter.meta.prevent="$emit('submit')"
      @keydown.enter.ctrl.prevent="$emit('submit')"
    />
    <p class="mt-1 text-[10px] font-semibold text-slate-400 dark:text-slate-500">
      <kbd class="rounded border border-gray-300 px-1 dark:border-slate-600">Ctrl</kbd> +
      <kbd class="rounded border border-gray-300 px-1 dark:border-slate-600">Enter</kbd> to generate
    </p>

    <div
      v-if="sourceImage"
      class="mt-3 flex items-center gap-2 rounded-lg border border-violet-200 bg-violet-50/80 p-2 transition-colors dark:border-violet-900/30 dark:bg-violet-900/20"
    >
      <img
        :src="imageUrl(sourceImage.id)"
        :alt="sourceImage.prompt"
        class="h-10 w-10 shrink-0 rounded object-cover"
      />
      <div class="min-w-0 flex-1">
        <p
          class="text-[10px] font-black tracking-wider text-violet-700 uppercase dark:text-violet-400"
        >
          Editing this image
        </p>
        <p class="truncate text-xs font-semibold text-slate-600 dark:text-slate-300">
          {{ sourceImage.prompt }}
        </p>
      </div>
      <button
        type="button"
        class="grid h-6 w-6 shrink-0 place-items-center rounded bg-violet-200 text-sm font-black text-violet-800 transition hover:bg-violet-300 dark:bg-violet-800 dark:text-violet-200 dark:hover:bg-violet-700"
        aria-label="Clear source image"
        @click="$emit('clear-source')"
      >
        &times;
      </button>
    </div>

    <div class="mt-3 space-y-3">
      <div v-if="providerNames.length > 1">
        <label for="image-provider" :class="labelClass">Provider</label>
        <select
          id="image-provider"
          :value="provider"
          :class="selectClass"
          @change="$emit('update:provider', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="name in providerNames" :key="name" :value="name">{{ name }}</option>
        </select>
      </div>

      <div>
        <label for="image-model" :class="labelClass">Model</label>
        <select
          id="image-model"
          :value="model"
          :class="selectClass"
          @change="$emit('update:model', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="m in models" :key="m.id" :value="m.id">{{ m.label }}</option>
        </select>
        <p
          v-if="!canEdit"
          class="mt-1 text-[10px] font-semibold text-slate-400 dark:text-slate-500"
        >
          This model generates from text only — it cannot edit existing images.
        </p>
      </div>

      <div v-if="sizes.length > 1">
        <span :class="labelClass">Size</span>
        <div class="flex flex-wrap gap-1">
          <FilterPill
            v-for="s in sizes"
            :key="s"
            :active="size === s"
            @click="$emit('update:size', s)"
          >
            {{ s }}
          </FilterPill>
        </div>
      </div>

      <div v-if="aspectRatios.length">
        <label for="image-aspect" :class="labelClass">Aspect ratio</label>
        <select
          id="image-aspect"
          :value="aspectRatio"
          :class="selectClass"
          @change="$emit('update:aspectRatio', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">Provider default</option>
          <option v-for="ratio in aspectRatios" :key="ratio" :value="ratio">{{ ratio }}</option>
        </select>
      </div>

      <div v-if="maxImages > 1">
        <label for="image-count" :class="labelClass">Images</label>
        <select
          id="image-count"
          :value="count"
          :class="selectClass"
          @change="$emit('update:count', Number(($event.target as HTMLSelectElement).value))"
        >
          <option v-for="n in Math.min(maxImages, 6)" :key="n" :value="n">{{ n }}</option>
        </select>
      </div>
    </div>

    <Button
      type="submit"
      variant="gradient-violet"
      size="lg"
      class="mt-4 w-full"
      :loading="busy"
      :loading-text="sourceImage ? 'Editing...' : 'Generating...'"
      :disabled="!prompt.trim()"
    >
      {{ sourceImage ? 'Edit image' : 'Generate image' }}
    </Button>
  </form>
</template>

<script setup lang="ts">
import type { AIImageModel, GeneratedImage } from '@browser-server/shared-types';
import { ref } from 'vue';
import { getGeneratedImageUrl } from '../../lib/api/ai';
import Button from '../ui/Button.vue';
import FilterPill from '../ui/FilterPill.vue';

defineProps<{
  prompt: string;
  provider: string;
  model: string;
  size: string;
  aspectRatio: string;
  count: number;
  providerNames: string[];
  models: AIImageModel[];
  sizes: string[];
  aspectRatios: string[];
  maxImages: number;
  canEdit: boolean;
  busy: boolean;
  sourceImage: GeneratedImage | null;
}>();

defineEmits<{
  'update:prompt': [value: string];
  'update:provider': [value: string];
  'update:model': [value: string];
  'update:size': [value: string];
  'update:aspectRatio': [value: string];
  'update:count': [value: number];
  'clear-source': [];
  submit: [];
}>();

const imageUrl = (id: string) => getGeneratedImageUrl(id);

const textareaRef = ref<HTMLTextAreaElement | null>(null);
defineExpose({ focus: () => textareaRef.value?.focus() });

const labelClass =
  'mb-1 block text-xs font-black uppercase tracking-wider text-slate-500 dark:text-slate-400';
const selectClass =
  'w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-violet-400 focus:outline-none focus:ring-4 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-violet-900/30';
</script>
