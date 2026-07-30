import {
  type LinkEnrichmentOptions,
  type LinkMetadata,
  type LinkProvider,
  registerLinkProvider,
} from './index'

const OEMBED_ENDPOINT = 'https://www.youtube.com/oembed'
const YOUTUBE_HOSTNAMES = new Set(['youtube.com', 'm.youtube.com', 'youtu.be'])

interface OembedResponse {
  title?: string
  thumbnail_url?: string
}

function isYouTubeUrl(url: string): boolean {
  try {
    const hostname = new URL(url).hostname.replace(/^www\./, '')
    return YOUTUBE_HOSTNAMES.has(hostname)
  } catch {
    return false
  }
}

const youtubeProvider: LinkProvider = {
  match: isYouTubeUrl,
  async enrich(url: string, options: LinkEnrichmentOptions): Promise<LinkMetadata | null> {
    const endpoint = `${OEMBED_ENDPOINT}?url=${encodeURIComponent(url)}&format=json`
    const response = await fetch(endpoint, { headers: { Accept: 'application/json' } })
    if (!response.ok) return null
    let payload: OembedResponse
    try {
      payload = (await response.json()) as OembedResponse
    } catch {
      return null
    }
    const metadata: LinkMetadata = { source: 'youtube' }
    if (payload.title) metadata.title = payload.title
    if (options.includeThumbnail && payload.thumbnail_url) {
      const videoId = new URL(payload.thumbnail_url).pathname.split('/').filter(Boolean).pop() ?? 'thumb'
      const ext = payload.thumbnail_url.split('.').pop()?.toLowerCase() ?? 'jpg'
      const safeExt = ['jpg', 'jpeg', 'png', 'webp'].includes(ext) ? ext : 'jpg'
      const mimeType = safeExt === 'png' ? 'image/png' : safeExt === 'webp' ? 'image/webp' : 'image/jpeg'
      metadata.thumbnail = {
        url: payload.thumbnail_url,
        filename: `youtube-${videoId}.${safeExt}`,
        mimeType,
      }
    }
    return metadata
  },
}

registerLinkProvider(youtubeProvider)
