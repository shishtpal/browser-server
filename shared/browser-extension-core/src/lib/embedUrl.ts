/**
 * Convert YouTube watch/shorts/youtu.be URLs to their official embed format.
 * Returns the original URL unchanged for non-YouTube URLs.
 */
export function toEmbedUrl(url: string): string {
  try {
    const parsed = new URL(url)
    const hostname = parsed.hostname.replace(/^www\./, '')

    if (hostname === 'youtube.com' || hostname === 'm.youtube.com') {
      // /watch?v=VIDEO_ID
      const videoId = parsed.searchParams.get('v')
      if (videoId && parsed.pathname === '/watch') {
        return `https://www.youtube.com/embed/${videoId}`
      }
      // /shorts/VIDEO_ID
      const shortsMatch = parsed.pathname.match(/^\/shorts\/([a-zA-Z0-9_-]+)/)
      if (shortsMatch) {
        return `https://www.youtube.com/embed/${shortsMatch[1]}`
      }
      // /embed/VIDEO_ID — already an embed URL
      if (parsed.pathname.startsWith('/embed/')) {
        return url
      }
    }

    if (hostname === 'youtu.be') {
      // youtu.be/VIDEO_ID
      const videoId = parsed.pathname.slice(1)
      if (videoId) {
        return `https://www.youtube.com/embed/${videoId}`
      }
    }
  } catch {
    // Invalid URL, return as-is
  }
  return url
}
