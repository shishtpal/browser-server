import type { AnalyticsSummary, AnalyticsSummaryParams, UsageBatchRequest, UsageBatchResponse } from '@browser-server/shared-types'
import { type TokenProvider, apiFetch, buildQuery } from '../internals'

export function createAnalyticsMethods(baseUrl: string, getToken?: TokenProvider) {
  return {
    batchUpsertUsage(data: UsageBatchRequest): Promise<UsageBatchResponse> {
      return apiFetch<UsageBatchResponse>(baseUrl, 'POST', '/api/analytics/usage', data, getToken)
    },

    getAnalyticsSummary(params: AnalyticsSummaryParams): Promise<AnalyticsSummary> {
      return apiFetch<AnalyticsSummary>(
        baseUrl,
        'GET',
        `/api/analytics/summary${buildQuery(params as unknown as Record<string, string | number | undefined>)}`,
        undefined,
        getToken,
      )
    },
  }
}
