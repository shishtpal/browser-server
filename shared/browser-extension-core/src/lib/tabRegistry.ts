import { createBrowserServerClient } from '@browser-server/shared-client'
import type { BrowserTabInfo } from '@browser-server/shared-types'
import { getBrowserApi } from '../browserApi'
import { getOrCreateInstanceId, registerInstance } from './instance'
import { getSettings, SETTINGS_KEY } from './settings'

const TAB_MAP_KEY = 'bs_tab_uuids' // { [tabId: string]: tab_uuid }

const SYNC_DEBOUNCE_MS = 2_000
const MAX_TABS = 500

function uuid(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `tab_${Math.random().toString(36).slice(2)}${Date.now().toString(36)}`
}

/**
 * TabRegistry maintains the native tabId → stable tab_uuid map and pushes a
 * latest-snapshot sync to the server whenever tabs change. The snapshot is
 * rebuilt from chrome.tabs.query(), so closed/native-reused tabs cannot leak
 * stale metadata.
 */
export class TabRegistry {
  private tabUuids = new Map<number, string>()
  private lastActive = new Map<string, string>() // tab_uuid → ISO when this tab was last active
  private syncTimer: ReturnType<typeof setTimeout> | null = null
  private started = false

  /** Resolve a tab_uuid to a native tab id, or undefined when unknown. */
  tabIdFor(uuid: string): number | undefined {
    for (const [tabId, id] of this.tabUuids) {
      if (id === uuid) {
        return tabId
      }
    }
    return undefined
  }

  /** Record a native tabId → tab_uuid association (created by us). */
  assign(tabId: number): string {
    const existing = this.tabUuids.get(tabId)
    if (existing) {
      return existing
    }
    const fresh = uuid()
    this.tabUuids.set(tabId, fresh)
    void this.persist()
    return fresh
  }

  start(): void {
    if (this.started) {
      return
    }
    this.started = true
    const api = getBrowserApi()

    void this.restore().then(() => {
      void this.refreshSnapshot()
    })

    // Re-sync when settings change (e.g. the token/base URL is configured in
    // options after install, or the instance label changed). Without this the
    // first sync would have already run with an empty token and tabs would
    // stay empty until the next tab event.
    api.storage.onChanged.addListener((changes) => {
      if (changes[SETTINGS_KEY]) {
        void this.refreshSnapshot()
      }
    })

    api.tabs.onUpdated.addListener(() => {
      void this.refreshSnapshot()
    })
    api.tabs.onActivated.addListener(() => {
      void this.refreshSnapshot()
    })
    api.tabs.onRemoved.addListener((tabId) => {
      const uuidKey = this.tabUuids.get(tabId)
      if (uuidKey !== undefined) {
        this.tabUuids.delete(tabId)
        this.lastActive.delete(uuidKey)
      }
      void this.persist()
      void this.scheduleSync()
    })
  }

  private async restore(): Promise<void> {
    try {
      const stored = await getBrowserApi().storage.local.get([TAB_MAP_KEY])
      const map = stored[TAB_MAP_KEY] as Record<string, string> | undefined
      if (map) {
        for (const [tabId, tabUuid] of Object.entries(map)) {
          this.tabUuids.set(Number(tabId), tabUuid)
        }
      }
    } catch {
      // First run or storage unavailable.
    }
  }

  private async persist(): Promise<void> {
    try {
      await getBrowserApi().storage.local.set({
        [TAB_MAP_KEY]: Object.fromEntries(this.tabUuids),
      })
    } catch {
      // Best effort.
    }
  }

  /**
   * Rebuild the current tab snapshot from native tabs and push it to the
   * server. `lastActive` survives per-uuid so an active tab does not lose its
   * ordering when the user toggles focus back and forth.
   */
  async refreshSnapshot(): Promise<void> {
    try {
      const tabs = await getBrowserApi().tabs.query({})
      const now = new Date().toISOString()
      const snapshot: BrowserTabInfo[] = []

      const instanceId = await this.instanceId()
      if (!instanceId) {
        console.debug('[browser-tabs] no instance id, skipping tab sync')
        return
      }

      for (const tab of tabs.slice(0, MAX_TABS)) {
        if (tab.id === undefined) {
          continue
        }
        let tabUuid = this.tabUuids.get(tab.id)
        if (!tabUuid) {
          tabUuid = uuid()
          this.tabUuids.set(tab.id, tabUuid)
        }
        let lastActive = this.lastActive.get(tabUuid) ?? ''
        if (tab.active) {
          lastActive = now
          this.lastActive.set(tabUuid, lastActive)
        }
        snapshot.push({
          tab_uuid: tabUuid,
          instance_id: instanceId,
          tab_id: tab.id,
          window_id: tab.windowId ?? 0,
          url: tab.url ?? '',
          title: tab.title ?? '',
          active: Boolean(tab.active),
          // Send null instead of '' for tabs that have never been active.
          // Go's time.Time JSON parser rejects an empty string, causing the
          // entire /api/browser/tabs POST to fail and tabs to never sync.
          last_active_at: lastActive || null,
          updated_at: now,
        })
      }

      // Native tab ids can be reused quickly; drop entries for closed tabs
      // before persisting.
      const liveIds = new Set<number>()
      for (const tab of tabs) {
        if (tab.id !== undefined) {
          liveIds.add(tab.id)
        }
      }
      for (const tabId of [...this.tabUuids.keys()]) {
        if (!liveIds.has(tabId)) {
          const uuidKey = this.tabUuids.get(tabId)!
          this.tabUuids.delete(tabId)
          this.lastActive.delete(uuidKey)
        }
      }
      await this.persist()

      const settings = await getSettings()
      if (!settings.apiToken) {
        console.debug('[browser-tabs] no API token configured; skipping tab sync')
        return
      }
      const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })
      try {
        await client.browserSyncTabs({ instance_id: instanceId, tabs: snapshot })
        console.debug(`[browser-tabs] synced ${snapshot.length} tabs to ${settings.apiBase}`)
      } catch (error) {
        console.debug('[browser-tabs] sync failed; re-registering instance and retrying', error)
        // Most commonly this means the instance is not registered yet
        // (server restarted after the extension, or the bus was unavailable
        // when the extension started). Re-register and retry once, then
        // defer to the next tab change.
        await registerInstance()
        try {
          await client.browserSyncTabs({ instance_id: instanceId, tabs: snapshot })
          console.debug(`[browser-tabs] retry synced ${snapshot.length} tabs`)
        } catch (retryError) {
          console.debug('[browser-tabs] retry failed; server offline or unreachable', retryError)
          // Server offline; retried on the next change.
        }
      }
    } catch (error) {
      console.debug('[browser-tabs] snapshot pass failed; skipping this pass', error)
    }
  }

  private scheduleSync(): void {
    if (this.syncTimer) {
      clearTimeout(this.syncTimer)
    }
    this.syncTimer = setTimeout(() => {
      this.syncTimer = null
      void this.refreshSnapshot()
    }, SYNC_DEBOUNCE_MS)
  }

  private async instanceId(): Promise<string | null> {
    return getOrCreateInstanceId()
  }
}
