import { createBrowserServerClient } from '@browser-server/shared-client'
import type { UsageEntry } from '@browser-server/shared-types'
import { getBrowserApi } from '../browserApi'
import { getSettings } from './settings'

const STORAGE_KEY = 'usage_buffer'
const MIN_SECONDS = 1

export class TimeTracker {
  private activeDomain: string | null = null
  private activeStartTime: number = 0
  private buffer: UsageEntry[] = []
  private isIdle = false

  startTracking(domain: string | null): void {
    if (!domain) {
      this.stopTracking()
      return
    }

    if (domain === this.activeDomain) {
      return
    }

    this.captureElapsed()
    this.activeDomain = domain
    this.activeStartTime = Date.now()
    this.isIdle = false
  }

  stopTracking(): void {
    this.captureElapsed()
    this.activeDomain = null
    this.activeStartTime = 0
  }

  private captureElapsed(): void {
    if (!this.activeDomain || this.isIdle || this.activeStartTime === 0) {
      return
    }

    const now = Date.now()
    const elapsed = Math.floor((now - this.activeStartTime) / 1000)
    if (elapsed < MIN_SECONDS) {
      return
    }

    this.addElapsedEntries(this.activeDomain, this.activeStartTime, now)
    this.activeStartTime = now
    this.persistBuffer()
  }

  handleIdleState(state: 'active' | 'idle' | 'locked'): void {
    if (state === 'idle' || state === 'locked') {
      this.captureElapsed()
      this.isIdle = true
    } else {
      this.isIdle = false
      if (this.activeDomain) {
        this.activeStartTime = Date.now()
      }
    }
  }

  async flush(): Promise<void> {
    this.captureElapsed()

    if (this.buffer.length === 0) {
      return
    }

    const settings = await getSettings()
    if (!settings) {
      return
    }

    const entries = [...this.buffer]
    this.buffer = []
    this.persistBuffer()

    try {
      const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })
      await client.batchUpsertUsage({
        user_id: Number.parseInt(settings.userId, 10),
        entries,
      })
    } catch {
      this.buffer = [...entries, ...this.buffer]
      this.persistBuffer()
    }
  }

  async restore(): Promise<void> {
    try {
      const data = await getBrowserApi().storage.local.get(STORAGE_KEY)
      const saved = (data as Record<string, UsageEntry[] | undefined>)[STORAGE_KEY]
      if (saved && saved.length > 0) {
        this.buffer = saved
        await this.flush()
      }
    } catch {
      // storage.session may not be available
    }
  }

  private persistBuffer(): void {
    try {
      void getBrowserApi().storage.local.set({ [STORAGE_KEY]: this.buffer })
    } catch {
      // best effort
    }
  }

  private addElapsedEntries(domain: string, startTime: number, endTime: number): void {
    let cursor = startTime

    while (cursor < endTime) {
      const boundary = Math.min(this.nextLocalMidnight(cursor), endTime)
      const seconds = Math.floor((boundary - cursor) / 1000)

      if (seconds >= MIN_SECONDS) {
        this.buffer.push({
          domain,
          date: this.localDateKey(cursor),
          seconds,
        })
      }

      cursor = boundary
    }
  }

  private localDateKey(timestamp: number): string {
    const d = new Date(timestamp)
    const year = d.getFullYear()
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
  }

  private nextLocalMidnight(timestamp: number): number {
    const d = new Date(timestamp)
    d.setHours(24, 0, 0, 0)
    return d.getTime()
  }
}
