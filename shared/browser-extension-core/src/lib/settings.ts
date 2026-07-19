import { getBrowserApi } from '../browserApi'

export interface ExtensionSettings {
  apiBase: string
  apiToken: string
  userId: string
  autoCapture: boolean
  unsafePreview: boolean
}

const SETTINGS_KEY = 'tracker_settings'

export const DEFAULT_SETTINGS: ExtensionSettings = {
  apiBase: 'http://localhost:9191',
  apiToken: '',
  userId: '1',
  autoCapture: true,
  unsafePreview: false,
}

export async function getSettings(): Promise<ExtensionSettings> {
  const stored = await getBrowserApi().storage.local.get(SETTINGS_KEY)
  const settings = stored[SETTINGS_KEY] as Partial<ExtensionSettings> | undefined
  return {
    ...DEFAULT_SETTINGS,
    ...settings,
  }
}

export async function saveSettings(settings: ExtensionSettings): Promise<void> {
  await getBrowserApi().storage.local.set({ [SETTINGS_KEY]: settings })
}

export async function resetSettings(): Promise<ExtensionSettings> {
  await getBrowserApi().storage.local.remove(SETTINGS_KEY)
  return DEFAULT_SETTINGS
}
