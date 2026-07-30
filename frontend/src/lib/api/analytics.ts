import type { AnalyticsSummary, AnalyticsSummaryParams } from '@browser-server/shared-types'
import { client } from './client'

export function getAnalyticsSummary(params: AnalyticsSummaryParams): Promise<AnalyticsSummary> {
  return client.getAnalyticsSummary(params)
}
