import type { BrowserServerClient, Todo } from '@browser-server/shared-client'
import { ref, watch, type Ref } from 'vue'
import { captureVisibleTab, dataUrlToBlob, getActiveTabDomain } from '../lib/browser'
import { timeAgo } from '../lib/format'

export interface TodoView {
  id: number
  title: string
  completed: boolean
  hasScreenshot: boolean
  screenshotUrl: string | null
  updatedAtLabel: string
  domain: string
}

function sortTodos(todos: Todo[]): Todo[] {
  return [...todos].sort(
    (left, right) => Number(left.completed) - Number(right.completed) || Date.parse(right.updated_at) - Date.parse(left.updated_at),
  )
}

function toView(todo: Todo, client: BrowserServerClient): TodoView {
  return {
    id: todo.id,
    title: todo.title,
    completed: todo.completed,
    hasScreenshot: Boolean(todo.screenshot_path),
    screenshotUrl: todo.screenshot_path ? client.getScreenshotUrl(todo.id) : null,
    updatedAtLabel: timeAgo(todo.updated_at),
    domain: todo.domain,
  }
}

export function useTodosView(
  client: Ref<BrowserServerClient | null>,
  userId: Ref<number>,
  autoCapture: Ref<boolean> = ref(false),
) {
  const currentDomain = ref<string | null>(null)
  const domainDisplay = ref<string>('Detecting active tab…')
  const screenshotPreview = ref<string | null>(null)
  const items = ref<TodoView[]>([])
  const stats = ref<string>('0 todos · 0 done')
  const errorMessage = ref<string | null>(null)

  function setPreview(dataUrl: string | null) {
    screenshotPreview.value = dataUrl
  }

  function reset() {
    items.value = []
    errorMessage.value = null
  }

  async function refresh() {
    if (!client.value || !userId.value) {
      return
    }

    if (!currentDomain.value) {
      items.value = []
      stats.value = '0 todos · 0 done'
      errorMessage.value = null
      return
    }

    try {
      const todos = sortTodos(await client.value.getTodos(userId.value, currentDomain.value))
      items.value = todos.map((todo) => toView(todo, client.value!))
      const completed = todos.filter((todo) => todo.completed).length
      stats.value = `${todos.length} todos · ${completed} done`
      errorMessage.value = null
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      errorMessage.value = `Server not reachable. ${message}`
      stats.value = '0 todos · 0 done'
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
        ? `Todos for: ${currentDomain.value}`
        : 'Could not determine current domain.'

      if (autoCapture.value && currentDomain.value) {
        setPreview(await captureVisibleTab())
      } else {
        setPreview(null)
      }

      reset()
      await refresh()
    })()

    return initPromise
  }

  function refreshWithScreenshot() {
    if (!client.value || !userId.value) {
      return
    }
    if (autoCapture.value && currentDomain.value) {
      void captureVisibleTab().then((dataUrl) => {
        setPreview(dataUrl)
      })
    } else {
      setPreview(null)
    }
  }

  async function add(title: string) {
    const trimmed = title.trim()
    if (!trimmed || !currentDomain.value || !client.value || !userId.value) {
      return
    }

    try {
      const todo = await client.value.createTodo({
        user_id: userId.value,
        title: trimmed,
        domain: currentDomain.value,
      })

      if (screenshotPreview.value) {
        await client.value.uploadScreenshot(todo.id, dataUrlToBlob(screenshotPreview.value))
      }

      setPreview(null)
      await refresh()
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      domainDisplay.value = `Failed to add todo: ${message}`
    }
  }

  async function toggle(id: number, completed: boolean) {
    if (!client.value || !userId.value) {
      return
    }

    try {
      await client.value.updateTodo(id, { user_id: userId.value, completed })
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      console.debug('Toggle failed', message)
    }
    await refresh()
  }

  async function remove(id: number) {
    if (!client.value) {
      return
    }

    try {
      await client.value.deleteTodo(id)
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      console.debug('Delete failed', message)
    }
    await refresh()
  }

  async function clearAll() {
    if (!client.value || !userId.value || !currentDomain.value) {
      return
    }

    try {
      const todos: Todo[] = await client.value.getTodos(userId.value, currentDomain.value)
      await Promise.all(todos.map((todo: Todo) => client.value!.deleteTodo(todo.id)))
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      console.debug('Clear todos failed', message)
    }
    await refresh()
  }

  watch(autoCapture, () => {
    refreshWithScreenshot()
  })

  return {
    currentDomain,
    domainDisplay,
    screenshotPreview,
    items,
    stats,
    errorMessage,
    init,
    add,
    toggle,
    remove,
    clearAll,
  }
}
