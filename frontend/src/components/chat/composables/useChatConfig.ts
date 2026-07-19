import type { AIConfig } from '@browser-server/shared-types'
import { computed, ref, watch } from 'vue'

export interface AIModelInfo {
  id: string
  label?: string
  default?: boolean
  supports_tools?: boolean
}

export function useChatConfig() {
  const config = ref<AIConfig | null>(null)
  const selectedProvider = ref('')
  const selectedModel = ref('')
  const yoloMode = ref(false)

  const configLabel = computed(() => {
    if (!config.value) return 'Loading…'
    return config.value.enabled ? `Ready · ${selectedModel.value.split('/').pop() || 'select model'}` : 'Disabled'
  })

  const providerModels = computed<AIModelInfo[]>(() => {
    if (!config.value || !selectedProvider.value) return []
    return config.value.providers[selectedProvider.value]?.models ?? []
  })

  const selectedModelSupportsTools = computed(() => {
    const current = providerModels.value.find((m) => m.id === selectedModel.value)
    return current?.supports_tools ?? false
  })

  const toolsEnabled = computed(() => (config.value?.tools?.enabled ?? false) && selectedModelSupportsTools.value)

  // Sync model when provider changes
  watch(selectedProvider, () => {
    const models = providerModels.value
    if (models.length > 0 && !models.some((m) => m.id === selectedModel.value)) {
      selectedModel.value = models.find((m) => m.default)?.id || models[0].id
    }
  })

  // Persist YOLO mode
  watch(yoloMode, (enabled) => {
    localStorage.setItem('ai-yolo-mode', String(enabled))
  })

  function initFromConfig(cfg: AIConfig) {
    config.value = cfg
    if (!cfg.enabled) return
    selectedProvider.value = cfg.default_provider || Object.keys(cfg.providers ?? {})[0] || ''
    const provider = cfg.providers?.[selectedProvider.value]
    const models = provider?.models ?? []
    selectedModel.value = provider?.default_model || models.find((m) => m.default)?.id || models[0]?.id || ''
  }

  function loadPersistedSettings() {
    yoloMode.value = localStorage.getItem('ai-yolo-mode') === 'true'
  }

  return {
    config,
    selectedProvider,
    selectedModel,
    yoloMode,
    configLabel,
    providerModels,
    selectedModelSupportsTools,
    toolsEnabled,
    initFromConfig,
    loadPersistedSettings,
  }
}
