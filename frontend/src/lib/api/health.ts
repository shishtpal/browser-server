import type { HealthResponse } from '@browser-server/shared-types'
import { client } from './client'

export function checkHealth(): Promise<HealthResponse> {
  return client.health()
}

export async function isServerOnline(): Promise<boolean> {
  return client.ping()
}
