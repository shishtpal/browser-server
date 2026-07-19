import type { CreateWalletInput, UpdateWalletInput, WalletEntry } from '@browser-server/shared-types'
import { type TokenProvider, apiFetch, buildQuery } from '../internals'

export function createWalletMethods(baseUrl: string, getToken?: TokenProvider) {
  return {
    getWallet(userId?: number, website?: string): Promise<WalletEntry[]> {
      return apiFetch<WalletEntry[]>(baseUrl, 'GET', `/api/wallet${buildQuery({ user_id: userId, website })}`, undefined, getToken)
    },

    createWallet(data: CreateWalletInput): Promise<WalletEntry> {
      return apiFetch<WalletEntry>(baseUrl, 'POST', '/api/wallet', data, getToken)
    },

    async revealWalletPassword(userId: number, id: number): Promise<string> {
      const result = await apiFetch<{ password: string }>(
        baseUrl,
        'GET',
        `/api/wallet/reveal${buildQuery({ user_id: userId, id })}`,
        undefined,
        getToken,
      )
      return result.password
    },

    updateWallet(id: number, data: UpdateWalletInput): Promise<WalletEntry> {
      return apiFetch<WalletEntry>(baseUrl, 'PUT', `/api/wallet/${id}`, data, getToken)
    },
  }
}
