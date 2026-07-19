import { ApiError, createBrowserServerClient } from '@browser-server/shared-client'
import type { BrowserTab, ContextMenuClickInfo } from '../browserApi'
import { getBrowserApi } from '../browserApi'
import { dataUrlToBlob, isTrackableUrl } from './browser'
import { getSettings, type ExtensionSettings } from './settings'

const MENU_ROOT = 'browser-server-capture'
const MENU_OPEN_WEB = 'browser-server-open-web'
const MENU_BOOKMARK = 'browser-server-save-bookmark'
const MENU_TODO = 'browser-server-create-todo'
const MENU_TODO_SCREENSHOT = 'browser-server-create-todo-screenshot'
const COMMAND_BOOKMARK = 'save-page-bookmark'
const COMMAND_TODO = 'create-page-todo'
const RETRY_ALARM = 'capture-sync'

const QUEUE_DB = 'browser-server-capture-queue'
const QUEUE_STORE = 'captures'

type CaptureKind = 'bookmark' | 'todo'
type FailureKind = 'transient' | 'auth' | 'permanent'

interface PendingCapture {
  id: string
  kind: CaptureKind
  userId: number
  title: string
  url: string
  description: string
  domain: string
  screenshot?: Blob
  todoId?: number
  createdAt: number
}

let queueOperation: Promise<void> = Promise.resolve()

function serializeQueueOperation<T>(operation: () => Promise<T>): Promise<T> {
  const result = queueOperation.then(operation, operation)
  queueOperation = result.then(() => undefined, () => undefined)
  return result
}

function openQueue(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(QUEUE_DB, 1)
    request.onupgradeneeded = () => {
      if (!request.result.objectStoreNames.contains(QUEUE_STORE)) {
        request.result.createObjectStore(QUEUE_STORE, { keyPath: 'id' })
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

async function putCapture(capture: PendingCapture): Promise<void> {
  const database = await openQueue()
  try {
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(QUEUE_STORE, 'readwrite')
      transaction.objectStore(QUEUE_STORE).put(capture)
      transaction.oncomplete = () => resolve()
      transaction.onerror = () => reject(transaction.error)
      transaction.onabort = () => reject(transaction.error)
    })
  } finally {
    database.close()
  }
}

async function deleteCapture(id: string): Promise<void> {
  const database = await openQueue()
  try {
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(QUEUE_STORE, 'readwrite')
      transaction.objectStore(QUEUE_STORE).delete(id)
      transaction.oncomplete = () => resolve()
      transaction.onerror = () => reject(transaction.error)
      transaction.onabort = () => reject(transaction.error)
    })
  } finally {
    database.close()
  }
}

async function getQueuedCaptures(): Promise<PendingCapture[]> {
  const database = await openQueue()
  try {
    const captures = await new Promise<PendingCapture[]>((resolve, reject) => {
      const request = database.transaction(QUEUE_STORE, 'readonly').objectStore(QUEUE_STORE).getAll()
      request.onsuccess = () => resolve(request.result as PendingCapture[])
      request.onerror = () => reject(request.error)
    })
    return captures.sort((left, right) => left.createdAt - right.createdAt)
  } finally {
    database.close()
  }
}

function normalizeUrl(value: string): string {
  try {
    const url = new URL(value)
    url.hash = ''
    url.hostname = url.hostname.toLowerCase()
    if (url.pathname.length > 1) {
      url.pathname = url.pathname.replace(/\/+$/, '')
    }
    return url.toString()
  } catch {
    return value.trim()
  }
}

function domainFromUrl(url: string): string {
  try {
    return new URL(url).hostname
  } catch {
    return ''
  }
}

function captureDescription(url: string, selectionText?: string): string {
  const selection = selectionText?.trim()
  return selection ? `${selection}\n\nSource: ${url}` : `Source: ${url}`
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function classifyFailure(error: unknown): FailureKind {
  if (!(error instanceof ApiError)) {
    return 'transient'
  }
  if (error.status === 401 || error.status === 403) {
    return 'auth'
  }
  return error.status === 408 || error.status === 429 || error.status === 503 || error.status >= 500
    ? 'transient'
    : 'permanent'
}

function notify(title: string, message: string): void {
  void getBrowserApi().notifications.create({
    type: 'basic',
    iconUrl: 'icons/icon128.png',
    title,
    message,
  }).catch((error) => console.debug('Capture notification failed', errorMessage(error)))
}

function configuredUser(settings: ExtensionSettings): number | null {
  const userId = Number.parseInt(settings.userId, 10)
  return settings.apiToken && Number.isInteger(userId) && userId > 0 ? userId : null
}

async function executeCapture(capture: PendingCapture, settings: ExtensionSettings): Promise<'saved' | 'duplicate'> {
  const client = createBrowserServerClient(settings.apiBase, { getToken: () => settings.apiToken })

  if (capture.kind === 'bookmark') {
    const bookmarks = await client.getBookmarks(capture.userId)
    const targetUrl = normalizeUrl(capture.url)
    if (bookmarks.some((bookmark) => normalizeUrl(bookmark.url) === targetUrl)) {
      return 'duplicate'
    }

    await client.createBookmark({
      user_id: capture.userId,
      title: capture.title,
      url: capture.url,
      description: capture.description,
      capture_id: capture.id,
    })
    return 'saved'
  }

  if (!capture.todoId) {
    const todo = await client.createTodo({
      user_id: capture.userId,
      title: capture.title,
      description: capture.description,
      domain: capture.domain,
      capture_id: capture.id,
    })
    capture.todoId = todo.id
    await putCapture(capture)
  }

  if (capture.screenshot) {
    await client.uploadScreenshot(capture.todoId, capture.screenshot, capture.id)
  }
  return 'saved'
}

async function submitCapture(capture: PendingCapture, settings: ExtensionSettings): Promise<void> {
  try {
    await putCapture(capture)
  } catch (error) {
    notify('Capture could not be queued', errorMessage(error))
    return
  }

  try {
    const result = await executeCapture(capture, settings)
    await deleteCapture(capture.id)
    if (result === 'duplicate') {
      notify('Bookmark already saved', capture.title)
    } else {
      notify(capture.kind === 'bookmark' ? 'Bookmark saved' : 'Todo created', capture.title)
    }
  } catch (error) {
    const failure = classifyFailure(error)
    if (failure === 'transient') {
      notify('Capture queued for sync', `${capture.title}\nThe server will be retried automatically.`)
      return
    }
    if (failure === 'auth') {
      notify('Capture waiting for authentication', 'Update the Browser Server token to resume syncing.')
      getBrowserApi().runtime.openOptionsPage()
      return
    }

    await deleteCapture(capture.id)
    if (capture.kind === 'todo' && capture.todoId && capture.screenshot) {
      notify('Todo created without screenshot', errorMessage(error))
    } else {
      notify('Capture failed', errorMessage(error))
    }
  }
}

async function buildCapture(
  kind: CaptureKind,
  tab: BrowserTab,
  selectionText?: string,
  includeScreenshot = false,
): Promise<PendingCapture | null> {
  const url = tab.url
  if (!url || !isTrackableUrl(url)) {
    notify('Page cannot be captured', 'Open a regular web page and try again.')
    return null
  }

  const settings = await getSettings()
  const userId = configuredUser(settings)
  if (!userId) {
    notify('Browser Server setup required', 'Set an API token and user before capturing pages.')
    getBrowserApi().runtime.openOptionsPage()
    return null
  }

  let screenshot: Blob | undefined
  if (includeScreenshot) {
    if (tab.windowId === undefined) {
      notify('Screenshot failed', 'The active browser window could not be found.')
      return null
    }
    try {
      screenshot = dataUrlToBlob(await getBrowserApi().tabs.captureVisibleTab(tab.windowId, { format: 'png' }))
    } catch (error) {
      notify('Screenshot failed', errorMessage(error))
      return null
    }
  }

  return {
    id: crypto.randomUUID(),
    kind,
    userId,
    title: tab.title?.trim() || url,
    url,
    description: captureDescription(url, selectionText),
    domain: domainFromUrl(url),
    screenshot,
    createdAt: Date.now(),
  }
}

async function captureTab(
  kind: CaptureKind,
  tab: BrowserTab | undefined,
  selectionText?: string,
  includeScreenshot = false,
): Promise<void> {
  const targetTab = tab ?? (await getBrowserApi().tabs.query({ active: true, currentWindow: true }))[0]
  if (!targetTab) {
    notify('Page cannot be captured', 'No active browser tab was found.')
    return
  }

  const capture = await buildCapture(kind, targetTab, selectionText, includeScreenshot)
  if (!capture) return

  const settings = await getSettings()
  await serializeQueueOperation(() => submitCapture(capture, settings))
}

async function flushQueue(): Promise<void> {
  const settings = await getSettings()
  if (!configuredUser(settings)) return

  let synced = 0
  let duplicates = 0
  const captures = await getQueuedCaptures()
  for (const capture of captures) {
    try {
      const result = await executeCapture(capture, settings)
      await deleteCapture(capture.id)
      if (result === 'duplicate') duplicates += 1
      else synced += 1
    } catch (error) {
      const failure = classifyFailure(error)
      if (failure === 'transient') break
      if (failure === 'auth') {
        notify('Captures waiting for authentication', 'Update the Browser Server token to resume syncing.')
        break
      }
      await deleteCapture(capture.id)
      if (capture.kind === 'todo' && capture.todoId && capture.screenshot) {
        notify('Todo created without screenshot', errorMessage(error))
      } else {
        notify('Queued capture failed', errorMessage(error))
      }
    }
  }

  if (synced > 0) {
    notify('Captures synced', `${synced} queued capture${synced === 1 ? '' : 's'} saved.`)
  }
  if (duplicates > 0) {
    notify('Duplicate bookmarks skipped', `${duplicates} queued bookmark${duplicates === 1 ? '' : 's'} already saved.`)
  }
}

async function createContextMenus(): Promise<void> {
  const menus = getBrowserApi().contextMenus
  await menus.removeAll()
  menus.create({ id: MENU_ROOT, title: 'Browser Server', contexts: ['page', 'selection'] })
  menus.create({ id: MENU_OPEN_WEB, parentId: MENU_ROOT, title: 'Open Web App', contexts: ['page', 'selection'] })
  menus.create({ id: MENU_BOOKMARK, parentId: MENU_ROOT, title: 'Save page as bookmark', contexts: ['page', 'selection'] })
  menus.create({ id: MENU_TODO, parentId: MENU_ROOT, title: 'Create todo from page', contexts: ['page', 'selection'] })
  menus.create({ id: MENU_TODO_SCREENSHOT, parentId: MENU_ROOT, title: 'Create todo with screenshot', contexts: ['page', 'selection'] })
}

export function initOneClickCapture(): void {
  const api = getBrowserApi()

  api.runtime.onInstalled.addListener(() => {
    void createContextMenus().catch((error) => console.debug('Capture menu setup failed', errorMessage(error)))
  })

  api.contextMenus.onClicked.addListener((info: ContextMenuClickInfo, tab?: BrowserTab) => {
    switch (String(info.menuItemId)) {
      case MENU_OPEN_WEB:
        void getSettings().then((settings) => {
          if (settings.apiBase) {
            void api.tabs.create({ url: settings.apiBase, active: true })
          }
        })
        break
      case MENU_BOOKMARK:
        void captureTab('bookmark', tab, info.selectionText).catch((error) => notify('Capture failed', errorMessage(error)))
        break
      case MENU_TODO:
        void captureTab('todo', tab, info.selectionText).catch((error) => notify('Capture failed', errorMessage(error)))
        break
      case MENU_TODO_SCREENSHOT:
        void captureTab('todo', tab, info.selectionText, true).catch((error) => notify('Capture failed', errorMessage(error)))
        break
    }
  })

  api.commands.onCommand.addListener((command, tab) => {
    if (command === COMMAND_BOOKMARK) {
      void captureTab('bookmark', tab).catch((error) => notify('Capture failed', errorMessage(error)))
    } else if (command === COMMAND_TODO) {
      void captureTab('todo', tab).catch((error) => notify('Capture failed', errorMessage(error)))
    }
  })

  api.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name === RETRY_ALARM) {
      void serializeQueueOperation(flushQueue).catch((error) => console.debug('Capture queue flush failed', errorMessage(error)))
    }
  })
  api.alarms.create(RETRY_ALARM, { periodInMinutes: 1 })
  void serializeQueueOperation(flushQueue).catch((error) => console.debug('Capture queue restore failed', errorMessage(error)))
}
