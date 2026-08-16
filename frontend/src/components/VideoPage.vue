<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Generation" title="Video" color="cyan">
      <template #stats>
        <StatCard :value="videos.length" label="In gallery" variant="dark" color="cyan" />
        <StatCard
          v-if="modelCount"
          :value="modelCount"
          label="Models"
          variant="primary"
          color="cyan"
        />
        <StatCard
          v-if="hasPending"
          :value="pendingCount"
          label="Rendering"
          variant="primary"
          color="amber"
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
      title="Video generation is unavailable"
      description="Configure bs-ai-video-models.json with a provider and API key, then restart the server."
      icon="default"
      color="amber"
    />

    <LoadingSpinner v-else-if="loading" message="Loading video workspace..." color="cyan" />

    <div v-else class="grid gap-4 lg:grid-cols-[minmax(320px,380px)_1fr]">
      <VideoComposer
        ref="composerRef"
        v-model:prompt="prompt"
        v-model:provider="provider"
        v-model:model="model"
        v-model:params="params"
        :provider-names="providerNames"
        :models="models"
        :parameters="parameters"
        :busy="busy"
        @submit="submit"
      />

      <div class="min-w-0">
        <EmptyState
          v-if="!videos.length && !hasPending"
          title="No videos yet"
          description="Write a prompt on the left to generate your first video."
          icon="default"
          color="cyan"
        />

        <template v-else>
          <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
            <div
              class="relative inline-flex items-center rounded-lg border border-gray-300 bg-white px-2.5 focus-within:border-cyan-400 focus-within:ring-4 focus-within:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:focus-within:ring-cyan-900/30"
            >
              <Search class="h-3.5 w-3.5 text-slate-400" :stroke-width="2.5" aria-hidden="true" />
              <input
                v-model="search"
                type="search"
                placeholder="Search prompt, provider, or model…"
                aria-label="Search video generations"
                class="search-input w-48 bg-transparent px-2 py-1.5 text-xs font-semibold text-slate-700 placeholder:text-slate-400 focus:outline-none sm:w-64 dark:text-slate-200"
              />
              <button
                v-if="search"
                type="button"
                class="grid h-5 w-5 place-items-center rounded text-slate-400 transition hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-slate-700 dark:hover:text-slate-200"
                aria-label="Clear search"
                @click="search = ''"
              >
                <X class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
              </button>
            </div>
          </div>

          <!-- In-progress placeholders -->
          <div v-if="hasPending" class="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <article
              v-for="v in pendingVideos"
              :key="v.id"
              class="overflow-hidden rounded-xl border border-cyan-200 bg-white shadow-sm dark:border-cyan-900/30 dark:bg-slate-800/90"
            >
              <div
                class="grid aspect-video place-items-center bg-gradient-to-br from-slate-800 to-slate-950"
              >
                <div class="flex flex-col items-center gap-2">
                  <LoaderCircle class="h-7 w-7 animate-spin text-cyan-400" aria-hidden="true" />
                  <p class="text-[11px] font-bold text-slate-300">
                    {{ v.status === 'queued' ? 'Queued…' : 'Generating…' }}
                  </p>
                  <div class="h-1.5 w-40 overflow-hidden rounded-full bg-slate-700">
                    <div
                      class="h-full bg-cyan-400 transition-all"
                      :style="{ width: `${Math.max(4, v.progress)}%` }"
                    />
                  </div>
                  <p class="text-[10px] text-slate-400 tabular-nums">{{ v.progress }}%</p>
                </div>
              </div>
              <div class="p-3">
                <p class="line-clamp-2 text-xs font-semibold text-slate-500 dark:text-slate-400">
                  {{ v.prompt }}
                </p>
              </div>
            </article>
          </div>

          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <VideoCard
              v-for="video in completedVideos"
              :key="video.id"
              :video="video"
              @open="openPreview"
              @reuse="reusePrompt"
              @delete="confirmDeleteVideo"
            />
          </div>

          <EmptyState
            v-if="search && !completedVideos.length && !pendingVideos.length"
            title="No matching videos"
            :description="`Nothing matches “${search}”. Try a different prompt, provider, or model.`"
            icon="search"
            color="cyan"
          />
        </template>
      </div>
    </div>

    <VideoViewer
      :video="preview"
      :index="previewIndex"
      :total="videos.length"
      @close="closePreview"
      @step="step"
      @reuse="reuseFromPreview"
      @delete="confirmDeleteVideo"
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
import { computed } from 'vue';
import { ListOrdered, LoaderCircle, RefreshCw, Search, X } from '@lucide/vue';
import { useUser } from '../composables/useUser';
import { useVideoPage } from './video/composables/useVideoPage';
import VideoCard from './video/VideoCard.vue';
import VideoComposer from './video/VideoComposer.vue';
import VideoViewer from './video/VideoViewer.vue';
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
  openPreview,
  closePreview,
  step,
  reusePrompt,
  reuseFromPreview,
  applyPrompt,
  confirmDeleteVideo,
} = useVideoPage();

const {
  config,
  videos,
  loading,
  busy,
  error,
  prompt,
  search,
  provider,
  model,
  params,
  providerNames,
  models,
  parameters,
  modelCount,
  hasPending,
  filteredVideos,
  submit,
} = gen;

const visibleVideos = computed(() => filteredVideos.value);
const pendingVideos = computed(() =>
  visibleVideos.value.filter((v) => v.status !== 'completed' && v.status !== 'failed'),
);
const completedVideos = computed(() =>
  visibleVideos.value.filter((v) => v.status === 'completed' || v.status === 'failed'),
);
const pendingCount = computed(
  () => videos.value.filter((v) => v.status !== 'completed' && v.status !== 'failed').length,
);
</script>

<style scoped>
.search-input::-webkit-search-cancel-button {
  display: none;
}
.search-input::-moz-search-clear {
  display: none;
}
</style>
