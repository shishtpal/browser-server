import type { HealthResponse } from '@browser-server/shared-types'
import { type TokenProvider, apiFetch } from '../internals'

export function createHealthMethods(baseUrl: string, _getToken?: TokenProvider) {
  return {
    async ping(): Promise<boolean> {
      try {
        const controller = new AbortController()
        const timeout = setTimeout(() => controller.abort(), 3000)
        const response = await fetch(`${baseUrl}/health`, {
          method: 'GET',
          signal: controller.signal,
        })
        clearTimeout(timeout)
        return response.ok
      } catch {
        return false
      }
    },

    health(): Promise<HealthResponse> {
      return apiFetch<HealthResponse>(baseUrl, 'GET', '/health')
    },
  }
}
