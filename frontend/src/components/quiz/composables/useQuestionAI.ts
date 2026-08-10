import { computed, ref, watch } from 'vue';
import { useLocalStorage } from '@vueuse/core';
import type { AIConfig, QuestionResponse } from '../../../types';
import {
  createAIConversation,
  deleteAIConversation,
  getAIConfig,
  sendAIMessageStream,
  stopAIGeneration,
} from '../../../lib/api';
import { questionCrosscheckPrompt, questionExplainPrompt } from '../quizFormat';

export type QuestionAIMode = 'explain' | 'crosscheck';

/** Fetch /api/ai/config once for the whole page lifetime, module-cached. */
let configPromise: Promise<AIConfig | null> | null = null;
function loadAIConfig(): Promise<AIConfig | null> {
  if (!configPromise) configPromise = getAIConfig().catch(() => null);
  return configPromise;
}

/**
 * Flashcard "Ask AI" actions (explain / cross-check).
 *
 * Selected provider/model are persisted in localStorage, independent of the
 * Chat page's selection. Each run uses an ephemeral AI conversation that is
 * deleted once the answer settles, so the Chat sidebar stays clean.
 */
export function useQuestionAI() {
  const config = ref<AIConfig | null>(null);
  const isReady = ref(false);

  const provider = useLocalStorage('bs.quiz.aiProvider', '');
  const model = useLocalStorage('bs.quiz.aiModel', '');

  const mode = ref<QuestionAIMode | null>(null);
  const content = ref('');
  const error = ref('');
  const isStreaming = ref(false);

  let conversationId: string | null = null;
  let controller: AbortController | null = null;
  let runId = 0;

  const providerNames = computed(() => Object.keys(config.value?.providers ?? {}));
  const providerItems = computed(() =>
    providerNames.value.map((name) => ({ value: name, label: name })),
  );
  const models = computed(() => config.value?.providers[provider.value]?.models ?? []);
  const modelItems = computed(() =>
    models.value.map((m) => ({ value: m.id, label: m.label || m.id })),
  );
  const activeModelLabel = computed(
    () => models.value.find((m) => m.id === model.value)?.label || model.value,
  );

  /** Feature usable: config loaded, AI enabled, and at least one model picked. */
  const available = computed(
    () => isReady.value && (config.value?.enabled ?? false) && Boolean(model.value),
  );

  function applyDefaults(cfg: AIConfig) {
    if (!providerNames.value.includes(provider.value)) {
      provider.value = cfg.default_provider || providerNames.value[0] || '';
    }
    const providerModels = cfg.providers[provider.value]?.models ?? [];
    if (!providerModels.some((m) => m.id === model.value)) {
      model.value =
        cfg.providers[provider.value]?.default_model ||
        providerModels.find((m) => m.default)?.id ||
        providerModels[0]?.id ||
        '';
    }
  }

  /** Load config exactly once; safe to call repeatedly. */
  async function init() {
    if (isReady.value) return;
    const cfg = await loadAIConfig();
    config.value = cfg;
    if (cfg?.enabled) applyDefaults(cfg);
    isReady.value = true;
  }

  // Provider switched → snap model to that provider's default.
  watch(provider, () => {
    if (!isReady.value) return;
    const providerModels = models.value;
    if (!providerModels.some((m) => m.id === model.value)) {
      model.value =
        providerModels.find((m) => m.default)?.id || providerModels[0]?.id || model.value;
    }
  });

  /** Abort the in-flight run (if any) and delete the ephemeral conversation. */
  function settle() {
    isStreaming.value = false;
    controller = null;
    const id = conversationId;
    conversationId = null;
    if (id) void deleteAIConversation(id).catch(() => {});
  }

  async function ask(question: QuestionResponse, nextMode: QuestionAIMode) {
    if (isStreaming.value) return;
    runId += 1;
    const run = runId;
    controller?.abort();
    settle();
    isStreaming.value = false;

    mode.value = nextMode;
    content.value = '';
    error.value = '';
    isStreaming.value = true;

    try {
      const conversation = await createAIConversation({
        provider: provider.value || undefined,
        model: model.value || undefined,
      });
      if (run !== runId) {
        void deleteAIConversation(conversation.id).catch(() => {});
        return;
      }
      conversationId = conversation.id;

      const prompt =
        nextMode === 'explain'
          ? questionExplainPrompt(question)
          : questionCrosscheckPrompt(question);

      controller = sendAIMessageStream(
        conversation.id,
        {
          content: prompt,
          provider: provider.value || undefined,
          model: model.value || undefined,
          stream: true,
          tools_enabled: false,
        },
        (event) => {
          if (run !== runId) return;
          switch (event.type) {
            case 'delta':
              content.value += event.content;
              break;
            case 'done':
              settle();
              break;
            case 'error':
              error.value = event.message || 'AI generation failed';
              settle();
              break;
          }
        },
        (err) => {
          if (run !== runId) return;
          error.value = err.message || 'Stream connection failed';
          settle();
        },
      );
    } catch (err) {
      if (run !== runId) return;
      error.value = err instanceof Error ? err.message : 'Failed to ask AI';
      settle();
    }
  }

  async function stop() {
    if (!isStreaming.value) return;
    controller?.abort();
    controller = null;
    const id = conversationId;
    if (id) {
      try {
        await stopAIGeneration(id);
      } catch {
        /* best-effort */
      }
    }
    settle();
    if (!error.value && content.value) content.value += '\n\n*(stopped)*';
  }

  /** Drop the run entirely (question changed or panel collapsed). */
  function clear() {
    runId += 1;
    controller?.abort();
    controller = null;
    settle();
    mode.value = null;
    content.value = '';
    error.value = '';
  }

  return {
    init,
    available,
    provider,
    model,
    providerItems,
    modelItems,
    activeModelLabel,
    mode,
    content,
    error,
    isStreaming,
    ask,
    stop,
    clear,
  };
}
