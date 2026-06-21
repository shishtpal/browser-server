export interface ExtensionSettings {
  apiBase: string
  apiToken: string
  userId: string
  autoCapture: boolean
}

const SETTINGS_KEY = 'tracker_settings'

export const DEFAULT_SETTINGS: ExtensionSettings = {
  apiBase: 'http://localhost:8080',
  apiToken: '',
  userId: '1',
  autoCapture: true,
}

export async function getSettings(): Promise<ExtensionSettings> {
  const stored = await chrome.storage.local.get(SETTINGS_KEY)
  const settings = stored[SETTINGS_KEY] as Partial<ExtensionSettings> | undefined
  return {
    ...DEFAULT_SETTINGS,
    ...settings,
  }
}

export async function saveSettings(settings: ExtensionSettings): Promise<void> {
  await chrome.storage.local.set({ [SETTINGS_KEY]: settings })
}

export async function resetSettings(): Promise<ExtensionSettings> {
  await chrome.storage.local.remove(SETTINGS_KEY)
  return DEFAULT_SETTINGS
}
