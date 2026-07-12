import { getBrowserApi } from '../browserApi'

const RESTRICTED_PREFIXES = [
  'chrome://',
  'chrome-extension://',
  'edge://',
  'brave://',
  'opera://',
  'browser://',
]

export function isTrackableUrl(url: string | undefined): url is string {
  if (!url || url === 'about:blank') {
    return false
  }

  return !RESTRICTED_PREFIXES.some((prefix) => url.startsWith(prefix))
}

export async function getActiveTabDomain(): Promise<string | null> {
  const [tab] = await getBrowserApi().tabs.query({ active: true, currentWindow: true })
  if (!tab?.url) {
    return null
  }

  try {
    return new URL(tab.url).hostname
  } catch {
    return null
  }
}

export async function getActiveTabInfo(): Promise<{ url: string; title: string } | null> {
  const [tab] = await getBrowserApi().tabs.query({ active: true, currentWindow: true })
  if (!tab?.url || !isTrackableUrl(tab.url)) {
    return null
  }
  return { url: tab.url, title: tab.title ?? tab.url }
}

export async function captureVisibleTab(): Promise<string | null> {
  return new Promise((resolve) => {
    getBrowserApi().runtime.sendMessage({ type: 'captureScreenshot' }).then((response) => {
      const typedResponse = response as { dataUrl?: string } | undefined
      resolve(typedResponse?.dataUrl ?? null)
    }).catch(() => {
      resolve(null)
    })
  })
}

export function dataUrlToBlob(dataUrl: string): Blob {
  const [header, base64] = dataUrl.split(',')
  const mime = /data:(.*);base64/.exec(header)?.[1] ?? 'image/png'
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)

  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }

  return new Blob([bytes], { type: mime })
}
