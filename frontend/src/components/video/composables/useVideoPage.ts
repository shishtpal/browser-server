import type { GeneratedVideo } from '@browser-server/shared-types';
import type { PromptResponse } from '../../../types';
import { computed, nextTick, onMounted, ref } from 'vue';
import { useModal } from '@browser-server/shared-modal';
import { useVideoGeneration } from './useVideoGeneration';

/**
 * Orchestrates the Video page: preview modal navigation, delete confirmation via
 * the shared modal, and prompt reuse/library flows — so VideoPage.vue stays pure
 * wiring. Generation state lives in useVideoGeneration.
 */
export function useVideoPage() {
  const gen = useVideoGeneration();
  const { videos, prompt, remove } = gen;

  const preview = ref<GeneratedVideo | null>(null);
  const showPromptLibrary = ref(false);

  const composerRef = ref<{ focus: () => void } | null>(null);
  const focusComposer = () => nextTick(() => composerRef.value?.focus());

  const previewIndex = computed(() =>
    preview.value ? videos.value.findIndex((v) => v.id === preview.value?.id) : -1,
  );

  const openPreview = (video: GeneratedVideo) => (preview.value = video);
  const closePreview = () => (preview.value = null);

  function step(delta: number) {
    const next = videos.value[previewIndex.value + delta];
    if (next) preview.value = next;
  }

  function injectPrompt(text: string) {
    prompt.value = text;
    focusComposer();
  }

  function reusePrompt(video: GeneratedVideo) {
    injectPrompt(video.prompt);
  }

  function reuseFromPreview(video: GeneratedVideo) {
    closePreview();
    reusePrompt(video);
  }

  function applyPrompt(p: PromptResponse) {
    if (!p.content) return;
    prompt.value = p.content;
    showPromptLibrary.value = false;
    focusComposer();
  }

  const { confirmDelete } = useModal();

  async function confirmDeleteVideo(video: GeneratedVideo) {
    const confirmed = await confirmDelete(
      `Delete "${video.prompt.slice(0, 80) || 'this video'}"?`,
      'The video file and its gallery entry are removed permanently.',
    );
    if (!confirmed) return;
    await remove(video.id);
    if (preview.value?.id === video.id) preview.value = null;
  }

  onMounted(gen.load);

  return {
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
  };
}
