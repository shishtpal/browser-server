import type { GeneratedImage } from '@browser-server/shared-types';
import type { PromptResponse } from '../../../types';
import { computed, nextTick, onMounted, ref } from 'vue';
import { useModal } from '@browser-server/shared-modal';
import { useImageGeneration } from './useImageGeneration';

/**
 * Orchestrates the Image page: preview modal navigation (prev/next/step),
 * delete confirmation via the shared modal, prompt reuse/library flows —
 * so ImagePage.vue stays pure wiring. Generation state lives in
 * useImageGeneration.
 */
export function useImagePage() {
  const gen = useImageGeneration();
  const { images, prompt, source, remove } = gen;

  const preview = ref<GeneratedImage | null>(null);
  const showPromptLibrary = ref(false);

  /** Focus the composer textarea after the DOM updates. */
  const composerRef = ref<{ focus: () => void } | null>(null);
  const focusComposer = () => nextTick(() => composerRef.value?.focus());

  /* ------------------------------- preview -------------------------------- */

  const previewIndex = computed(() =>
    preview.value ? images.value.findIndex((i) => i.id === preview.value?.id) : -1,
  );

  const openPreview = (image: GeneratedImage) => (preview.value = image);
  const closePreview = () => (preview.value = null);

  function step(delta: number) {
    const next = images.value[previewIndex.value + delta];
    if (next) preview.value = next;
  }

  /* ------------------------------ reuse flows ------------------------------ */

  function useAsSource(image: GeneratedImage) {
    source.value = image.id;
    focusComposer();
  }

  function reusePrompt(image: GeneratedImage) {
    prompt.value = image.prompt;
    focusComposer();
  }

  function reuseFromPreview(image: GeneratedImage) {
    closePreview();
    reusePrompt(image);
  }

  function editFromPreview(image: GeneratedImage) {
    closePreview();
    useAsSource(image);
  }

  function applyPrompt(p: PromptResponse) {
    if (!p.content) return;
    prompt.value = p.content;
    showPromptLibrary.value = false;
    focusComposer();
  }

  /* ------------------------------ deletion ---------------------------------- */

  const { confirmDelete } = useModal();

  async function confirmDeleteImage(image: GeneratedImage) {
    const confirmed = await confirmDelete(
      `Delete "${image.prompt.slice(0, 80) || 'this image'}"?`,
      'The image file and its gallery entry are removed permanently.',
    );
    if (!confirmed) return;
    await remove(image.id);
    if (preview.value?.id === image.id) preview.value = null;
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
    useAsSource,
    reusePrompt,
    reuseFromPreview,
    editFromPreview,
    applyPrompt,
    confirmDeleteImage,
  };
}
