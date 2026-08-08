<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Generation" title="Image" color="violet">
      <template #stats>
        <StatCard :value="images.length" label="In gallery" variant="dark" color="violet" />
        <StatCard
          v-if="modelCount"
          :value="modelCount"
          label="Models"
          variant="primary"
          color="violet"
        />
      </template>
      <template #controls>
        <Button variant="ghost" size="sm" @click="showPromptLibrary = true">Prompt Library</Button>
        <Button variant="ghost" size="sm" :disabled="loading" @click="load">Refresh</Button>
      </template>
    </PageHeader>

    <ErrorBanner v-if="error" :message="error" :on-retry="load" />

    <EmptyState
      v-if="!loading && !config"
      title="Image generation is unavailable"
      description="Configure bs-ai-image-models.json with a provider and API key, then restart the server."
      icon="default"
      color="amber"
    />

    <LoadingSpinner v-else-if="loading" message="Loading image workspace..." color="violet" />

    <div v-else class="grid gap-4 lg:grid-cols-[minmax(320px,380px)_1fr]">
      <ImageComposer
        ref="composerRef"
        v-model:prompt="prompt"
        v-model:provider="provider"
        v-model:model="model"
        v-model:size="size"
        v-model:aspect-ratio="aspectRatio"
        v-model:count="count"
        :provider-names="providerNames"
        :models="models"
        :sizes="sizes"
        :aspect-ratios="aspectRatios"
        :max-images="maxImages"
        :can-edit="canEdit"
        :busy="busy"
        :source-image="sourceImage"
        @clear-source="source = ''"
        @submit="submit"
      />

      <div>
        <EmptyState
          v-if="!images.length && !busy"
          title="No images yet"
          description="Write a prompt on the left to generate your first image."
          icon="default"
          color="violet"
        />

        <div v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <article
            v-if="busy"
            class="overflow-hidden rounded-xl border border-violet-200 bg-white shadow-sm transition-colors dark:border-violet-900/30 dark:bg-slate-800/90"
          >
            <div class="grid aspect-square place-items-center bg-violet-50 dark:bg-violet-900/10">
              <div
                class="h-8 w-8 animate-spin rounded-full border-4 border-violet-500 border-t-transparent"
              ></div>
            </div>
            <div class="p-3">
              <p class="line-clamp-2 text-xs font-semibold text-slate-500 dark:text-slate-400">
                {{ prompt }}
              </p>
            </div>
          </article>

          <ImageCard
            v-for="i in images"
            :key="i.id"
            :image="i"
            :can-edit="canEdit"
            @open="preview = $event"
            @edit="useAsSource"
            @reuse="reusePrompt"
            @delete="confirming = $event"
          />
        </div>
      </div>
    </div>

    <ImageViewer
      :image="preview"
      :index="previewIndex"
      :total="images.length"
      :can-edit="canEdit"
      @close="preview = null"
      @step="step"
      @reuse="reuseFromPreview"
      @edit="editFromPreview"
    />

    <Modal
      :open="!!confirming"
      title="Delete this image?"
      description="The image file and its gallery entry are removed permanently."
      @close="confirming = null"
    >
      <div class="flex justify-end gap-2">
        <Button variant="secondary" @click="confirming = null">Cancel</Button>
        <Button variant="danger" @click="confirmRemove">Delete</Button>
      </div>
    </Modal>

    <PromptManager
      :open="showPromptLibrary"
      :user-id="currentUserId"
      @select="applyPrompt"
      @close="showPromptLibrary = false"
    />
  </div>
</template>

<script setup lang="ts">
import type { GeneratedImage } from '@browser-server/shared-types';
import type { PromptResponse } from '../types';
import { computed, nextTick, onMounted, ref } from 'vue';
import { useImageGeneration } from '../composables/useImageGeneration';
import { useUser } from '../composables/useUser';
import ImageCard from './image/ImageCard.vue';
import ImageComposer from './image/ImageComposer.vue';
import ImageViewer from './image/ImageViewer.vue';
import PromptManager from './prompts/PromptManager.vue';
import Button from './ui/Button.vue';
import EmptyState from './ui/EmptyState.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import Modal from './ui/Modal.vue';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';

const { currentUserId } = useUser();
const {
  config,
  images,
  loading,
  busy,
  error,
  prompt,
  provider,
  model,
  size,
  aspectRatio,
  count,
  source,
  providerNames,
  models,
  sizes,
  aspectRatios,
  maxImages,
  canEdit,
  sourceImage,
  modelCount,
  load,
  submit,
  remove,
} = useImageGeneration();

const composerRef = ref<InstanceType<typeof ImageComposer> | null>(null);
const preview = ref<GeneratedImage | null>(null);
const confirming = ref<GeneratedImage | null>(null);
const showPromptLibrary = ref(false);

const previewIndex = computed(() =>
  preview.value ? images.value.findIndex((i) => i.id === preview.value?.id) : -1,
);

function focusPrompt() {
  nextTick(() => composerRef.value?.focus());
}

function useAsSource(image: GeneratedImage) {
  source.value = image.id;
  focusPrompt();
}

function reusePrompt(image: GeneratedImage) {
  prompt.value = image.prompt;
  focusPrompt();
}

function applyPrompt(p: PromptResponse) {
  if (!p.content) return;
  prompt.value = p.content;
  showPromptLibrary.value = false;
  focusPrompt();
}

function step(delta: number) {
  const next = images.value[previewIndex.value + delta];
  if (next) preview.value = next;
}

function reuseFromPreview(image: GeneratedImage) {
  preview.value = null;
  reusePrompt(image);
}

function editFromPreview(image: GeneratedImage) {
  preview.value = null;
  useAsSource(image);
}

async function confirmRemove() {
  const target = confirming.value;
  if (!target) return;
  confirming.value = null;
  await remove(target.id);
  if (preview.value?.id === target.id) preview.value = null;
}

onMounted(load);
</script>
