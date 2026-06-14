import type { BrowserServerClient, WalletEntry } from '@browser-server/shared-client'
import { ref, type Ref } from 'vue'
import { getActiveTabDomain } from '../lib/browser'

export interface WalletItemView {
  id: number
  website: string
  username: string
  description: string
}

export function useWalletView(client: Ref<BrowserServerClient | null>, userId: Ref<number>) {
  const currentDomain = ref<string | null>(null)
  const domainDisplay = ref<string>('Detecting active tab…')
  const items = ref<WalletItemView[]>([])
  const stats = ref<string>('0 passwords')
  const errorMessage = ref<string | null>(null)

  function toView(entry: WalletEntry): WalletItemView {
    return {
      id: entry.id,
      website: entry.website,
      username: entry.username,
      description: entry.description,
    }
  }

  async function refresh() {
    if (!client.value || !userId.value) {
      return
    }

    if (!currentDomain.value) {
      items.value = []
      stats.value = '0 passwords'
      errorMessage.value = null
      return
    }

    try {
      const entries = await client.value.getWallet(userId.value, currentDomain.value)
      items.value = entries.map(toView)
      stats.value = `${entries.length} password${entries.length === 1 ? '' : 's'}`
      errorMessage.value = null
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      errorMessage.value = `Server not reachable. ${message}`
      stats.value = '0 passwords'
      items.value = []
    }
  }

  let initPromise: Promise<void> | null = null

  async function init() {
    if (initPromise) {
      return initPromise
    }

    initPromise = (async () => {
      currentDomain.value = await getActiveTabDomain()
      domainDisplay.value = currentDomain.value
        ? `Passwords for: ${currentDomain.value}`
        : 'Could not determine current domain.'
      await refresh()
    })()

    return initPromise
  }

  async function reveal(item: WalletItemView): Promise<string> {
    if (!client.value || !userId.value) {
      throw new Error('Not ready')
    }
    return client.value.revealWalletPassword(userId.value, item.website, item.username)
  }

  return {
    currentDomain,
    domainDisplay,
    items,
    stats,
    errorMessage,
    init,
    refresh,
    reveal,
  }
}
