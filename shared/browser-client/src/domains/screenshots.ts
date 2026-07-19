import type { Screenshot } from '@browser-server/shared-types'
import { type TokenProvider, apiErrorFromBody, authHeader, buildQuery } from '../internals'

export function createScreenshotMethods(baseUrl: string, getToken?: TokenProvider) {
  return {
    async uploadScreenshot(todoId: number, file: Blob, captureId?: string): Promise<Screenshot> {
      const formData = new FormData()
      formData.append('file', file, 'screenshot.png')

      const response = await fetch(`${baseUrl}/api/screenshots${buildQuery({ todo_id: todoId, capture_id: captureId })}`, {
        method: 'POST',
        headers: authHeader(getToken),
        body: formData,
      })

      if (!response.ok) {
        const text = await response.text()
        throw apiErrorFromBody(response.status, text, `Upload failed: ${response.status}`)
      }

      return response.json() as Promise<Screenshot>
    },

    getScreenshotUrl(todoId: number): string {
      // Screenshots load via <img src>, which can't set an Authorization
      // header, so the token is passed as a query param instead.
      const token = getToken?.()
      const suffix = token ? `?token=${encodeURIComponent(token)}` : ''
      return `${baseUrl}/api/screenshots/${todoId}${suffix}`
    },
  }
}
