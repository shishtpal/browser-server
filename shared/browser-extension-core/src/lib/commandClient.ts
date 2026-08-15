import { createBrowserServerClient } from '@browser-server/shared-client'
import type { BrowserCommand } from '@browser-server/shared-types'
import { getBrowserApi } from '../browserApi'
import { getSettings } from './settings'
import { getOrCreateInstanceId } from './instance'

export const COMMAND_POLL_ALARM = 'bs-command-poll'

/**
 * CommandClient delivers queued browser commands to the extension.
 *
 * Primary path: a long-lived SSE stream (GET /api/browser/events?instance_id=)
 * with the operator token in the Authorization header. EventSource cannot send
 * headers, so the stream is read with fetch() + ReadableStream, which MV3
 * service workers support.
 *
 * Fallback path: an alarm (every 30s) polls GET /api/browser/queue; combined
 * with the server's queue replay on SSE connect, this guarantees delivery even
 * if the service worker is suspended when a command is enqueued.
 */
export class CommandClient {
  private controller: AbortController | null = null
  private retryTimer: ReturnType<typeof setTimeout> | null = null
  private backoffMs = 2_000
  private started = false
  private polling = false

  constructor(
    private readonly onCommand: (command: BrowserCommand) => void,
  ) {}

  start(): void {
    if (this.started) {
      return
    }
    this.started = true
    const api = getBrowserApi()
    void this.connect()
    api.alarms.onAlarm.addListener((alarm) => {
      if (alarm.name === COMMAND_POLL_ALARM) {
        void this.pollQueue()
      }
    })
    api.alarms.create(COMMAND_POLL_ALARM, { periodInMinutes: 0.5 })
  }

  stop(): void {
    this.started = false
    this.controller?.abort()
    if (this.retryTimer) {
      clearTimeout(this.retryTimer)
      this.retryTimer = null
    }
  }

  private async connect(): Promise<void> {
    const settings = await getSettings()
    if (!settings.apiToken) {
      this.scheduleRetry()
      return
    }
    const instanceId = await getOrCreateInstanceId()
    const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })
    this.controller = new AbortController()
    const url = `${settings.apiBase}/api/browser/events?instance_id=${encodeURIComponent(instanceId)}`

    try {
      const response = await fetch(url, {
        headers: { Authorization: `Bearer ${settings.apiToken}` },
        signal: this.controller.signal,
      })
      if (!response.ok || !response.body) {
        throw new Error(`SSE status ${response.status}`)
      }
      this.backoffMs = 2_000
      await this.readStream(response.body)
    } catch (error) {
      if (this.started && (error as Error).name !== 'AbortError') {
        // Also drain the queue in case the stream dropped events.
        void this.pollQueue()
      }
      this.scheduleRetry()
    } finally {
      void client
    }
  }

  private async readStream(body: ReadableStream<Uint8Array>): Promise<void> {
    const reader = body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    for (;;) {
      const { done, value } = await reader.read()
      if (done) {
        return
      }
      buffer += decoder.decode(value, { stream: true })
      const frames = buffer.split('\n\n')
      buffer = frames.pop() ?? ''
      for (const frame of frames) {
        const event = this.parseFrame(frame)
        if (event?.type === 'command' && event.command) {
          this.onCommand(event.command)
        }
      }
    }
  }

  private parseFrame(frame: string): { type?: string; command?: BrowserCommand } | null {
    let type: string | undefined
    let data = ''
    for (const line of frame.split('\n')) {
      if (line.startsWith('event:')) {
        type = line.slice(6).trim()
      } else if (line.startsWith('data:')) {
        data += line.slice(5).trim()
      }
    }
    if (!data) {
      return null
    }
    try {
      const parsed = JSON.parse(data) as { type?: string; command?: BrowserCommand }
      return parsed
    } catch {
      return null
    }
  }

  private scheduleRetry(): void {
    if (!this.started || this.retryTimer) {
      return
    }
    this.retryTimer = setTimeout(() => {
      this.retryTimer = null
      void this.connect()
    }, this.backoffMs)
    this.backoffMs = Math.min(this.backoffMs * 2, 60_000)
  }

  /** Alarm fallback: fetch queued commands that SSE may have missed. */
  async pollQueue(): Promise<void> {
    if (this.polling) {
      return
    }
    this.polling = true
    try {
      const settings = await getSettings()
      if (!settings.apiToken) {
        return
      }
      const instanceId = await getOrCreateInstanceId()
      const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })
      const queued = await client.browserQueue(instanceId)
      for (const command of queued) {
        this.onCommand(command)
      }
    } catch {
      // Server offline; retried on the next alarm.
    } finally {
      this.polling = false
    }
  }
}
