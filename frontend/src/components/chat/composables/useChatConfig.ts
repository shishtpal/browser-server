import type { AIConfig, AIProfile, AISkill } from '@browser-server/shared-types';
import { computed, ref, watch } from 'vue';
import { useLocalStorage } from '@vueuse/core';

export interface AIModelInfo {
  id: string;
  label?: string;
  default?: boolean;
  supports_tools?: boolean;
  supports_vision?: boolean;
}

export function useChatConfig() {
  const config = ref<AIConfig | null>(null);

  const selectedProvider = ref('');
  const selectedModel = ref('');
  const selectedProfile = ref('');

  const yoloMode = useLocalStorage(`bs.ai.yoloMode`, false);
  const userToolsEnabled = useLocalStorage(`bs.ai.userToolsEnabled`, true);
  const includeAllToolDefinitions = useLocalStorage(`bs.ai.includeAllToolDefinitions`, false);
  const disabledTools = useLocalStorage<Set<string>>(`bs.ai.disabledTools`, new Set(), {
    serializer: {
      read: (v) => (v ? new Set(JSON.parse(v)) : new Set()),
      write: (v) => JSON.stringify([...v]),
    },
  });

  const activeSkills = useLocalStorage<string[]>(`bs.ai.activeSkills`, []);

  const showThinking = useLocalStorage(`bs.ai.showThinking`, true);

  /** Available profiles from the server config */
  const profiles = computed<AIProfile[]>(() => config.value?.profiles ?? []);

  /** Available skills from the server config */
  const skills = computed<AISkill[]>(() => config.value?.skills ?? []);

  /** Sanitized MCP server availability reported by the backend. */
  const mcp = computed(() => config.value?.mcp ?? { configured: false, servers: [] });

  const configLabel = computed(() => {
    if (!config.value) return 'Loading…';
    return config.value.enabled
      ? `Ready · ${selectedModel.value.split('/').pop() || 'select model'}`
      : 'Disabled';
  });

  const providerModels = computed<AIModelInfo[]>(() => {
    if (!config.value || !selectedProvider.value) return [];
    return config.value.providers[selectedProvider.value]?.models ?? [];
  });

  const selectedModelSupportsTools = computed(() => {
    const current = providerModels.value.find((m) => m.id === selectedModel.value);
    return current?.supports_tools ?? false;
  });

  const selectedModelSupportsVision = computed(() => {
    const current = providerModels.value.find((m) => m.id === selectedModel.value);
    return current?.supports_vision ?? false;
  });

  const attachmentsConfig = computed(() => config.value?.chat?.attachments ?? null);

  const toolsEnabled = computed(
    () =>
      (config.value?.tools?.enabled ?? false) &&
      selectedModelSupportsTools.value &&
      userToolsEnabled.value,
  );

  /** All tools declared in the server config */
  const availableTools = computed<string[]>(() => config.value?.tools?.allowed ?? []);

  /** Tool name → category mapping from the server */
  const toolCategories = computed<Record<string, string>>(
    () => config.value?.tools?.categories ?? {},
  );

  /** Tools grouped by category for UI display */
  const toolsByCategory = computed<{ category: string; tools: string[] }[]>(() => {
    const cats = toolCategories.value;
    const map = new Map<string, string[]>();
    for (const tool of availableTools.value) {
      const cat = cats[tool] || 'Other';
      if (!map.has(cat)) map.set(cat, []);
      map.get(cat)!.push(tool);
    }
    return Array.from(map.entries()).map(([category, tools]) => ({ category, tools }));
  });

  /** Tools the user has chosen to keep active (allowed minus user-disabled) */
  const activeTools = computed<string[]>(() =>
    availableTools.value.filter((t) => !disabledTools.value.has(t)),
  );

  // Sync model when provider changes
  watch(selectedProvider, () => {
    const models = providerModels.value;
    if (models.length > 0 && !models.some((m) => m.id === selectedModel.value)) {
      selectedModel.value = models.find((m) => m.default)?.id || models[0].id;
    }
  });

  function toggleTool(name: string, enabled: boolean) {
    const next = new Set(disabledTools.value);
    if (enabled) {
      next.delete(name);
    } else {
      next.add(name);
    }
    disabledTools.value = next;
  }

  function setActiveSkills(names: string[]) {
    activeSkills.value = names;
  }

  function initFromConfig(cfg: AIConfig) {
    config.value = cfg;
    if (!cfg.enabled) return;
    selectedProvider.value = cfg.default_provider || Object.keys(cfg.providers ?? {})[0] || '';
    const provider = cfg.providers?.[selectedProvider.value];
    const models = provider?.models ?? [];
    selectedModel.value =
      provider?.default_model || models.find((m) => m.default)?.id || models[0]?.id || '';
  }

  return {
    config,
    selectedProvider,
    selectedModel,
    selectedProfile,
    profiles,
    skills,
    mcp,
    activeSkills,
    showThinking,
    yoloMode,
    userToolsEnabled,
    includeAllToolDefinitions,
    disabledTools,
    configLabel,
    providerModels,
    selectedModelSupportsTools,
    selectedModelSupportsVision,
    attachmentsConfig,
    toolsEnabled,
    availableTools,
    toolCategories,
    toolsByCategory,
    activeTools,
    toggleTool,
    setActiveSkills,
    initFromConfig,
  };
}
