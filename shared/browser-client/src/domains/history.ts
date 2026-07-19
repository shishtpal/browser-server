import type {
  CreateHistoryInput,
  GroupedHistoryParams,
  GroupedHistoryResponse,
  History,
  HistoryDomainSummary,
  OmniboxSearchParams,
  OmniboxSearchResult,
} from '@browser-server/shared-types'
import { type TokenProvider, apiFetch, buildQuery } from '../internals'

export function createHistoryMethods(baseUrl: string, getToken?: TokenProvider) {
  return {
    searchOmnibox(params: OmniboxSearchParams): Promise<OmniboxSearchResult[]> {
      return apiFetch<OmniboxSearchResult[]>(
        baseUrl,
        'GET',
        `/api/search/omnibox${buildQuery(params as unknown as Record<string, string | number | undefined>)}`,
        undefined,
        getToken,
      )
    },

    getHistory(userId?: number, url?: string, limit?: number, offset?: number): Promise<History[]> {
      return apiFetch<History[]>(baseUrl, 'GET', `/api/history${buildQuery({ user_id: userId, url, limit, offset })}`, undefined, getToken)
    },

    getGroupedHistory(params: GroupedHistoryParams): Promise<GroupedHistoryResponse> {
      return apiFetch<GroupedHistoryResponse>(
        baseUrl,
        'GET',
        `/api/history/grouped${buildQuery(params as unknown as Record<string, string | number | undefined>)}`,
        undefined,
        getToken,
      )
    },

    getHistoryDomains(userId?: number, query?: string): Promise<HistoryDomainSummary[]> {
      return apiFetch<HistoryDomainSummary[]>(
        baseUrl,
        'GET',
        `/api/history/domains${buildQuery({ user_id: userId, q: query })}`,
        undefined,
        getToken,
      )
    },

    createHistory(data: CreateHistoryInput): Promise<History> {
      return apiFetch<History>(baseUrl, 'POST', '/api/history', data, getToken)
    },

    deleteHistory(id: number): Promise<void> {
      return apiFetch<void>(baseUrl, 'DELETE', `/api/history/${id}`, undefined, getToken)
    },
  }
}
