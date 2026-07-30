/**
 * Link enrichment registry.
 *
 * Each LinkProvider knows how to recognise a class of URLs (YouTube, Vimeo, …) and
 * fetch richer metadata (title, thumbnail) for them via public APIs (e.g. oembed).
 *
 * Providers auto-register on import. The first matching provider wins. To add a new
 * service, drop a new file in this folder and import it for its side-effects from
 * this barrel.
 */

export interface LinkThumbnail {
  url: string
  filename: string
  mimeType: string
}

export interface LinkMetadata {
  /** Provider identifier, e.g. 'youtube', 'vimeo'. Set by the matching provider. */
  source?: string
  title?: string
  description?: string
  thumbnail?: LinkThumbnail
}

export interface LinkEnrichmentOptions {
  includeThumbnail: boolean
}

export interface LinkProvider {
  match(url: string): boolean
  enrich(url: string, options: LinkEnrichmentOptions): Promise<LinkMetadata | null>
}

const providers: LinkProvider[] = []

export function registerLinkProvider(provider: LinkProvider): void {
  providers.push(provider)
}

export async function enrichLink(
  url: string,
  options: LinkEnrichmentOptions,
): Promise<LinkMetadata | null> {
  for (const provider of providers) {
    if (!provider.match(url)) continue
    try {
      const result = await provider.enrich(url, options)
      if (result) return result
    } catch {
      // Provider failures are non-fatal — caller falls back to default behavior.
    }
  }
  return null
}
