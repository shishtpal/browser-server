import type { ExtensionSettings } from '../lib/settings'
import { computed, type ComputedRef } from 'vue'
import { createBrowserServerClient, type BrowserServerClient } from '@browser-server/shared-client'

export function createApiClient(settings: ExtensionSettings): BrowserServerClient {
  return createBrowserServerClient(settings.apiBase)
}

export function useUserId(settingsRef: ComputedRef<ExtensionSettings | null>) {
  return computed(() => {
    const settings = settingsRef.value
    if (!settings) {
      return 0
    }
    return Number.parseInt(settings.userId, 10) || 0
  })
}
