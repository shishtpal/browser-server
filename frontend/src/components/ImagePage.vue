<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
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
        <Button variant="ghost" size="sm" @click="showPromptLibrary = true">
          <span class="inline-flex items-center gap-1.5">
            <ListOrdered class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
            Prompt Library
          </span>
        </Button>
        <Button variant="ghost" size="sm" :disabled="loading" @click="gen.load">
          <span class="inline-flex items-center gap-1.5">
            <RefreshCw class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
            Refresh
          </span>
        </Button>
      </template>
    </PageHeader>

    <ErrorBanner v-if="error" :message="error" :on-retry="gen.load" />

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

      <div class="min-w-0">
        <EmptyState
          v-if="!images.length && !busy"
          title="No images yet"
          description="Write a prompt on the left to generate your first image."
          icon="default"
          color="violet"
        />

        <template v-else>
          <div class="mb-3 flex items-center justify-end">
            <div
              class="inline-flex overflow-hidden rounded-lg border border-gray-300 dark:border-slate-600"
            >
              <button
                type="button"
                class="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-black transition"
                :class="
                  viewMode === 'grouped'
                    ? 'bg-violet-600 text-white'
                    : 'bg-white text-slate-600 hover:bg-slate-100 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700'
                "
                :aria-pressed="viewMode === 'grouped'"
                @click="viewMode = 'grouped'"
              >
                <Folder class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
                Grouped
              </button>
              <button
                type="button"
                class="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-black transition"
                :class="
                  viewMode === 'all'
                    ? 'bg-violet-600 text-white'
                    : 'bg-white text-slate-600 hover:bg-slate-100 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700'
                "
                :aria-pressed="viewMode === 'all'"
                @click="viewMode = 'all'"
              >
                <LayoutGrid class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
                All
              </button>
            </div>
          </div>

          <!-- Grouped by prompt -->
          <div v-if="viewMode === 'grouped'" class="space-y-3">
            <ImageGroup
              v-for="group in groups"
              :key="group.key"
              :group="group"
              :can-edit="canEdit"
              @open="openPreview"
              @edit="useAsSource"
              @reuse="reusePrompt"
              @delete="confirmDeleteImage"
              @use-prompt="(g) => injectPrompt(g.prompt)"
            />
            <div
              v-if="singles.length || busy"
              class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
            >
              <!-- In-progress placeholder -->
              <article
                v-if="busy"
                class="overflow-hidden rounded-xl border border-violet-200 bg-white shadow-sm transition-colors dark:border-violet-900/30 dark:bg-slate-800/90"
                role="status"
                aria-label="Generating image"
              >
                <div
                  class="grid aspect-square place-items-center bg-violet-50 dark:bg-violet-900/10"
                >
                  <LoaderCircle class="h-8 w-8 animate-spin text-violet-500" aria-hidden="true" />
                </div>
                <div class="p-3">
                  <p class="line-clamp-2 text-xs font-semibold text-slate-500 dark:text-slate-400">
                    {{ prompt }}
                  </p>
                </div>
              </article>

              <ImageCard
                v-for="image in singles"
                :key="image.id"
                :image="image"
                :can-edit="canEdit"
                @open="openPreview"
                @edit="useAsSource"
                @reuse="reusePrompt"
                @delete="confirmDeleteImage"
              />
            </div>
          </div>

          <!-- Flat grid -->
          <div v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <!-- In-progress placeholder -->
            <article
              v-if="busy"
              class="overflow-hidden rounded-xl border border-violet-200 bg-white shadow-sm transition-colors dark:border-violet-900/30 dark:bg-slate-800/90"
              role="status"
              aria-label="Generating image"
            >
              <div class="grid aspect-square place-items-center bg-violet-50 dark:bg-violet-900/10">
                <LoaderCircle class="h-8 w-8 animate-spin text-violet-500" aria-hidden="true" />
              </div>
              <div class="p-3">
                <p class="line-clamp-2 text-xs font-semibold text-slate-500 dark:text-slate-400">
                  {{ prompt }}
                </p>
              </div>
            </article>

            <ImageCard
              v-for="image in images"
              :key="image.id"
              :image="image"
              :can-edit="canEdit"
              @open="openPreview"
              @edit="useAsSource"
              @reuse="reusePrompt"
              @delete="confirmDeleteImage"
            />
          </div>
        </template>
      </div>
    </div>

    <ImageViewer
      :image="preview"
      :index="previewIndex"
      :total="images.length"
      :can-edit="canEdit"
      @close="closePreview"
      @step="step"
      @reuse="reuseFromPreview"
      @edit="editFromPreview"
    />

    <PromptManager
      :open="showPromptLibrary"
      :user-id="currentUserId"
      @select="applyPrompt"
      @close="showPromptLibrary = false"
    />
  </div>
</template>

<script setup lang="ts">
import { Folder, LayoutGrid, ListOrdered, LoaderCircle, RefreshCw } from '@lucide/vue';
import { useUser } from '../composables/useUser';
import { useImagePage } from './image/composables/useImagePage';
import ImageCard from './image/ImageCard.vue';
import ImageComposer from './image/ImageComposer.vue';
import ImageGroup from './image/ImageGroup.vue';
import ImageViewer from './image/ImageViewer.vue';
import PromptManager from './prompts/PromptManager.vue';
import Button from './ui/Button.vue';
import EmptyState from './ui/EmptyState.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';

const { currentUserId } = useUser();

const {
  gen,
  composerRef,
  preview,
  previewIndex,
  showPromptLibrary,
  viewMode,
  openPreview,
  closePreview,
  step,
  useAsSource,
  injectPrompt,
  reusePrompt,
  reuseFromPreview,
  editFromPreview,
  applyPrompt,
  confirmDeleteImage,
} = useImagePage();

const {
  config,
  images,
  groups,
  singles,
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
  submit,
} = gen;
</script>
