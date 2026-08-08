import { computed, ref, watch } from 'vue';
import type { AIImageConfig, GeneratedImage } from '@browser-server/shared-types';
import {
  deleteGeneratedImage,
  generateImage,
  getAIImageConfig,
  listGeneratedImages,
} from '../lib/api/ai';

/** Config, generation form state and gallery for the image workspace. */
export function useImageGeneration() {
  const config = ref<AIImageConfig | null>(null);
  const images = ref<GeneratedImage[]>([]);
  const loading = ref(true);
  const busy = ref(false);
  const error = ref('');

  const prompt = ref('');
  const provider = ref('');
  const model = ref('');
  const size = ref('1K');
  const aspectRatio = ref('');
  const count = ref(1);
  const source = ref('');

  const providerNames = computed(() => Object.keys(config.value?.providers ?? {}));
  const models = computed(() => config.value?.providers[provider.value]?.models ?? []);
  const selected = computed(() => models.value.find((m) => m.id === model.value));
  const sizes = computed(() => selected.value?.image_sizes ?? ['1K']);
  const aspectRatios = computed(() => selected.value?.aspect_ratios ?? []);
  const maxImages = computed(() => selected.value?.max_images ?? 1);
  const canEdit = computed(() => selected.value?.supports_editing ?? false);
  const sourceImage = computed(() => images.value.find((i) => i.id === source.value) ?? null);
  const modelCount = computed(() =>
    Object.values(config.value?.providers ?? {}).reduce((n, p) => n + p.models.length, 0),
  );

  watch(config, (c) => {
    if (!c) return;
    provider.value = c.default_provider;
    model.value = c.providers[provider.value]?.models.find((x) => x.default)?.id ?? '';
  });

  watch(provider, () => {
    model.value = models.value.find((x) => x.default)?.id ?? models.value[0]?.id ?? '';
  });

  // Sizes and editing support are per-model, so reset both when the model changes
  // to avoid sending a combination the server will reject.
  watch(
    sizes,
    (s) => {
      if (!s.includes(size.value)) size.value = s[0] ?? '1K';
    },
    { immediate: true },
  );

  watch(canEdit, (ok) => {
    if (!ok) source.value = '';
  });

  async function load() {
    loading.value = true;
    error.value = '';
    try {
      config.value = await getAIImageConfig();
      images.value = await listGeneratedImages();
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load image tools';
    } finally {
      loading.value = false;
    }
  }

  async function submit() {
    if (!prompt.value.trim() || busy.value) return;
    busy.value = true;
    error.value = '';
    try {
      const r = await generateImage({
        prompt: prompt.value,
        provider: provider.value,
        model: model.value,
        image_size: size.value,
        aspect_ratio: aspectRatio.value || undefined,
        n: count.value,
        source_image_ids: source.value ? [source.value] : [],
      });
      images.value.unshift(...r.images);
      source.value = '';
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Generation failed';
    } finally {
      busy.value = false;
    }
  }

  async function remove(id: string) {
    try {
      await deleteGeneratedImage(id);
      images.value = images.value.filter((x) => x.id !== id);
      if (source.value === id) source.value = '';
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Delete failed';
    }
  }

  return {
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
  };
}
