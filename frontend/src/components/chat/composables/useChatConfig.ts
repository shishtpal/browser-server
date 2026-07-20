import type { AIConfig, AIProfile } from '@browser-server/shared-types'
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
  const selectedProfile = ref('')
  const yoloMode = ref(false)
  const userToolsEnabled = ref(true)
  const disabledTools = ref<Set<string>>(new Set())

  /** Available profiles from the server config */
  const profiles = computed<AIProfile[]>(() => config.value?.profiles ?? [])

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

  const toolsEnabled = computed(() =>
    (config.value?.tools?.enabled ?? false) && selectedModelSupportsTools.value && userToolsEnabled.value
  )

  /** All tools declared in the server config */
  const availableTools = computed<string[]>(() => config.value?.tools?.allowed ?? [])

  /** Tool name → category mapping from the server */
  const toolCategories = computed<Record<string, string>>(() => config.value?.tools?.categories ?? {})

  /** Tools grouped by category for UI display */
  const toolsByCategory = computed<{ category: string; tools: string[] }[]>(() => {
    const cats = toolCategories.value
    const map = new Map<string, string[]>()
    for (const tool of availableTools.value) {
      const cat = cats[tool] || 'Other'
      if (!map.has(cat)) map.set(cat, [])
      map.get(cat)!.push(tool)
    }
    return Array.from(map.entries()).map(([category, tools]) => ({ category, tools }))
  })

  /** Tools the user has chosen to keep active (allowed minus user-disabled) */
  const activeTools = computed<string[]>(() =>
    availableTools.value.filter((t) => !disabledTools.value.has(t))
  )

  // Sync model when provider changes
  watch(selectedProvider, () => {
    const models = providerModels.value
    if (models.length > 0 && !models.some((m) => m.id === selectedModel.value)) {
      selectedModel.value = models.find((m) => m.default)?.id || models[0].id
    }
  })

  // Persist YOLO mode and tool preferences
  watch(yoloMode, (enabled) => {
    localStorage.setItem('ai-yolo-mode', String(enabled))
  })

  watch(userToolsEnabled, (enabled) => {
    localStorage.setItem('ai-tools-enabled', String(enabled))
  })

  watch(disabledTools, (set) => {
    localStorage.setItem('ai-disabled-tools', JSON.stringify([...set]))
  }, { deep: true })

  function toggleTool(name: string, enabled: boolean) {
    const next = new Set(disabledTools.value)
    if (enabled) {
      next.delete(name)
    } else {
      next.add(name)
    }
    disabledTools.value = next
  }

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
    const storedToolsEnabled = localStorage.getItem('ai-tools-enabled')
    if (storedToolsEnabled !== null) {
      userToolsEnabled.value = storedToolsEnabled !== 'false'
    }
    try {
      const stored = localStorage.getItem('ai-disabled-tools')
      if (stored) {
        disabledTools.value = new Set(JSON.parse(stored))
      }
    } catch { /* ignore malformed storage */ }
  }

  return {
    config,
    selectedProvider,
    selectedModel,
    selectedProfile,
    profiles,
    yoloMode,
    userToolsEnabled,
    disabledTools,
    configLabel,
    providerModels,
    selectedModelSupportsTools,
    toolsEnabled,
    availableTools,
    toolCategories,
    toolsByCategory,
    activeTools,
    toggleTool,
    initFromConfig,
    loadPersistedSettings,
  }
}
