export interface DownloadedThumbnail {
  blob: Blob
  filename: string
  mimeType: string
}

export async function downloadThumbnail(
  url: string,
  fallbackFilename = 'thumbnail.jpg',
  fallbackMimeType = 'image/jpeg',
): Promise<DownloadedThumbnail | null> {
  try {
    const response = await fetch(url)
    if (!response.ok) return null
    const blob = await response.blob()
    const mimeType = blob.type || fallbackMimeType
    const ext = mimeType.split('/')[1]?.toLowerCase().replace('jpeg', 'jpg') ?? 'jpg'
    const filename = `${fallbackFilename.replace(/\.[^.]+$/, '')}.${ext}`
    return { blob, filename, mimeType }
  } catch {
    return null
  }
}
