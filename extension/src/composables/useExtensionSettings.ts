import { onBeforeUnmount, ref, type Ref } from 'vue'
import { getSettings, type ExtensionSettings } from '../lib/settings'

const settingsRef: Ref<ExtensionSettings | null> = ref(null)
const listeners = new Set<() => void>()

async function refreshSettings(): Promise<ExtensionSettings> {
  const next = await getSettings()
  settingsRef.value = next
  return next
}

export function useExtensionSettings() {
  if (settingsRef.value === null) {
    void refreshSettings()
  }

  function onChange(): void {
    void refreshSettings()
  }

  chrome.storage.onChanged.addListener(onChange)
  listeners.add(onChange)

  onBeforeUnmount(() => {
    chrome.storage.onChanged.removeListener(onChange)
    listeners.delete(onChange)
  })

  return {
    settings: settingsRef,
    refresh: refreshSettings,
  }
}
