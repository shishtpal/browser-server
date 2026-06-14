import { createBrowserServerClient } from '@browser-server/shared-client'
import type { UsageEntry } from '@browser-server/shared-types'
import { getSettings } from './settings'

const STORAGE_KEY = 'usage_buffer'
const FLUSH_INTERVAL = 30000
const MIN_SECONDS = 1

export class TimeTracker {
  private activeDomain: string | null = null
  private activeStartTime: number = 0
  private buffer: UsageEntry[] = []
  private isIdle = false
  private flushTimer: ReturnType<typeof setInterval> | null = null

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

    const elapsed = Math.floor((Date.now() - this.activeStartTime) / 1000)
    if (elapsed < MIN_SECONDS) {
      return
    }

    const today = new Date().toISOString().slice(0, 10)

    this.buffer.push({
      domain: this.activeDomain,
      date: today,
      seconds: elapsed,
    })

    this.activeStartTime = Date.now()
    this.persistBuffer()
  }

  handleIdleState(state: chrome.idle.IdleState): void {
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
      const client = createBrowserServerClient(settings.apiBase)
      await client.batchUpsertUsage({
        user_id: Number.parseInt(settings.userId, 10),
        entries,
      })
    } catch {
      this.buffer = [...entries, ...this.buffer]
      this.persistBuffer()
    }
  }

  startPeriodicFlush(): void {
    if (this.flushTimer) {
      clearInterval(this.flushTimer)
    }
    this.flushTimer = setInterval(() => {
      void this.flush()
    }, FLUSH_INTERVAL)
  }

  stopPeriodicFlush(): void {
    if (this.flushTimer) {
      clearInterval(this.flushTimer)
      this.flushTimer = null
    }
  }

  async restore(): Promise<void> {
    try {
      const data = await chrome.storage.session.get(STORAGE_KEY)
      const saved: UsageEntry[] = data[STORAGE_KEY]
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
      void chrome.storage.session.set({ [STORAGE_KEY]: this.buffer })
    } catch {
      // best effort
    }
  }
}
