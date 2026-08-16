import type {
  AIVideoConfig,
  AIVideoModel,
  GeneratedVideo,
  VideoParamSpec,
} from '@browser-server/shared-types';
import { computed, ref, watch } from 'vue';
import {
  deleteGeneratedVideo,
  generateVideo,
  getAIVideoConfig,
  listGeneratedVideos,
} from '../../../lib/api/ai';

const terminal = (status: string) => status === 'completed' || status === 'failed';

/** Build the default value for a single parameter spec. */
function defaultValue(spec: VideoParamSpec): unknown {
  if (spec.default !== undefined && spec.default !== null) {
    return spec.default;
  }
  switch (spec.type) {
    case 'boolean':
      return false;
    case 'image_urls':
      return [];
    case 'select':
      return spec.options?.[0] ?? '';
    case 'number':
      return spec.min ?? undefined;
    default:
      return '';
  }
}

/** Config, generation form state and gallery for the video workspace. */
export function useVideoGeneration() {
  const config = ref<AIVideoConfig | null>(null);
  const videos = ref<GeneratedVideo[]>([]);
  const loading = ref(true);
  const busy = ref(false);
  const error = ref('');

  const prompt = ref('');
  const search = ref('');
  const provider = ref('');
  const model = ref('');
  const params = ref<Record<string, unknown>>({});

  const providerNames = computed(() => Object.keys(config.value?.providers ?? {}));
  const models = computed<AIVideoModel[]>(
    () => config.value?.providers[provider.value]?.models ?? [],
  );
  const selected = computed(() => models.value.find((m) => m.id === model.value));
  const parameters = computed<VideoParamSpec[]>(() => selected.value?.parameters ?? []);
  const modelCount = computed(() =>
    Object.values(config.value?.providers ?? {}).reduce((n, p) => n + p.models.length, 0),
  );

  function buildDefaults(m: AIVideoModel | undefined): Record<string, unknown> {
    const out: Record<string, unknown> = {};
    for (const spec of m?.parameters ?? []) {
      if (spec.key === 'prompt') continue; // prompt is the dedicated textarea, not a generic param
      out[spec.key] = defaultValue(spec);
    }
    return out;
  }

  watch(config, (c) => {
    if (!c) return;
    provider.value = c.default_provider;
    model.value = c.providers[provider.value]?.models.find((x) => x.default)?.id ?? '';
    params.value = buildDefaults(selected.value);
  });

  watch(provider, () => {
    model.value = models.value.find((x) => x.default)?.id ?? models.value[0]?.id ?? '';
  });

  // Parameters are per-model; reset them when the model changes so the server
  // never receives an inconsistent option set.
  watch(
    model,
    () => {
      params.value = buildDefaults(selected.value);
    },
    { immediate: true },
  );

  async function load() {
    loading.value = true;
    error.value = '';
    try {
      config.value = await getAIVideoConfig();
      videos.value = await listGeneratedVideos();
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load video tools';
    } finally {
      loading.value = false;
    }
  }

  async function submit() {
    if (!prompt.value.trim() || busy.value) return;
    busy.value = true;
    error.value = '';
    try {
      const r = await generateVideo({
        prompt: prompt.value,
        provider: provider.value,
        model: model.value,
        params: params.value,
      });
      videos.value.unshift(r.video);
      startPolling();
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Generation failed';
    } finally {
      busy.value = false;
    }
  }

  async function remove(id: string) {
    try {
      await deleteGeneratedVideo(id);
      // Mutate in place so cards/modal bound to the same array stay in sync;
      // replacing the array wholesale leaves the open preview pointing at a
      // stale object (see refreshOnce).
      const idx = videos.value.findIndex((v) => v.id === id);
      if (idx >= 0) videos.value.splice(idx, 1);
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Delete failed';
    }
  }

  // ─── Async status polling ────────────────────────────────────────────────
  // Video generation is asynchronous: the server creates a task and polls the
  // provider in the background. We refresh the gallery while any task is still
  // in flight, then stop once everything has settled.

  const polling = ref(false);
  let timer: ReturnType<typeof setInterval> | null = null;

  function anyPending() {
    return videos.value.some((v) => !terminal(v.status));
  }

  async function refreshOnce() {
    try {
      // Merge by id instead of replacing the array wholesale: cards and the
      // preview modal hold object references derived from `videos`, and a
      // fresh array every poll would reset scroll and leave the open viewer
      // showing a stale progress snapshot. 2xx responses are always
      // authoritative — scroll state and object identity are preserved.
      const latest = await listGeneratedVideos();
      const byId = new Map(videos.value.map((v) => [v.id, v]));
      videos.value = latest.map((v) => {
        const existing = byId.get(v.id);
        return existing ? Object.assign(existing, v) : v;
      });
    } catch {
      // keep polling; a transient error shouldn't kill the loop
    }
    if (!anyPending()) stopPolling();
  }

  function startPolling() {
    if (polling.value) return;
    polling.value = true;
    timer = setInterval(refreshOnce, 3000);
  }

  function stopPolling() {
    polling.value = false;
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
  }

  // Resume polling automatically if the page loads with in-flight tasks.
  watch(loading, (done) => {
    if (done && !polling.value && anyPending()) startPolling();
  });

  const searchQuery = computed(() => search.value.trim().toLowerCase());
  const filteredVideos = computed<GeneratedVideo[]>(() => {
    const q = searchQuery.value;
    if (!q) return videos.value;
    return videos.value.filter(
      (v) =>
        v.prompt.toLowerCase().includes(q) ||
        v.provider.toLowerCase().includes(q) ||
        v.model.toLowerCase().includes(q),
    );
  });

  const hasPending = computed(() => anyPending());

  return {
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
    selected,
    parameters,
    modelCount,
    filteredVideos,
    hasPending,
    load,
    submit,
    remove,
  };
}
