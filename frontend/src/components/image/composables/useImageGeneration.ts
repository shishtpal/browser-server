import type { AIImageConfig, GeneratedImage } from '@browser-server/shared-types';
import { computed, ref, watch } from 'vue';
import {
  deleteGeneratedImage,
  generateImage,
  getAIImageConfig,
  listGeneratedImages,
} from '../../../lib/api/ai';

/** A set of images generated from the exact same prompt — shown as a folder. */
export interface ImageGroup {
  key: string;
  prompt: string;
  images: GeneratedImage[];
  models: number;
  providers: number;
}

/** Config, generation form state and gallery for the image workspace. */
export function useImageGeneration() {
  const config = ref<AIImageConfig | null>(null);
  const images = ref<GeneratedImage[]>([]);
  const loading = ref(true);
  const busy = ref(false);
  const error = ref('');

  const prompt = ref('');
  const search = ref('');
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

  // Only images whose prompt/provider/model match the search box. When the
  // search box is empty this is the full gallery.
  const filteredImages = computed<GeneratedImage[]>(() => {
    const q = search.value.trim().toLowerCase();
    if (!q) return images.value;
    return images.value.filter(
      (img) =>
        img.prompt.toLowerCase().includes(q) ||
        img.provider.toLowerCase().includes(q) ||
        img.model.toLowerCase().includes(q),
    );
  });

  // Group images by exact prompt so the user can compare outputs from
  // different providers/models side by side. Prompts with a single image are
  // not folded into a folder — they render as plain cards.
  const groups = computed<ImageGroup[]>(() => {
    const map = new Map<string, GeneratedImage[]>();
    for (const img of filteredImages.value) {
      const key = img.prompt;
      const bucket = map.get(key);
      if (bucket) bucket.push(img);
      else map.set(key, [img]);
    }
    const out: ImageGroup[] = [];
    for (const [key, items] of map) {
      if (items.length < 2) continue;
      out.push({
        key,
        prompt: key,
        images: items,
        models: new Set(items.map((i) => i.model)).size,
        providers: new Set(items.map((i) => i.provider)).size,
      });
    }
    return out;
  });

  // Images that belong to a prompt with exactly one generation; rendered as
  // plain cards (never wrapped in a folder).
  const singles = computed<GeneratedImage[]>(() => {
    const counts = new Map<string, number>();
    for (const img of filteredImages.value) {
      counts.set(img.prompt, (counts.get(img.prompt) ?? 0) + 1);
    }
    return filteredImages.value.filter((i) => (counts.get(i.prompt) ?? 0) < 2);
  });

  return {
    config,
    images,
    loading,
    busy,
    error,
    prompt,
    search,
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
    groups,
    singles,
    filteredImages,
    load,
    submit,
    remove,
  };
}
