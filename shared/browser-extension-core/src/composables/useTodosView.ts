import type { BrowserServerClient, Todo, TodoPriority, UpdateTodoInput } from '@browser-server/shared-client'
import { ref, watch, type Ref } from 'vue'
import { captureVisibleTab, dataUrlToBlob, getActiveTabDomain } from '../lib/browser'
import { timeAgo } from '@browser-server/shared-utils'

export interface TodoView {
  id: number
  title: string
  description: string
  status: Todo['status']
  priority: TodoPriority
  pinned: boolean
  startDate: string | null
  dueDateLabel: string | null
  tags: string[]
  subtaskTotal: number
  subtaskCompleted: number
  hasScreenshot: boolean
  screenshotUrl: string | null
  screenshotPath: string
  updatedAtLabel: string
  createdAtLabel: string
  domain: string
}

function formatDueDate(value: string | null): string | null {
  if (!value) return null
  const date = new Date(value.length === 10 ? `${value}T00:00:00` : value)
  if (Number.isNaN(date.getTime())) return null

  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const due = new Date(date)
  due.setHours(0, 0, 0, 0)
  const dayDifference = Math.round((due.getTime() - today.getTime()) / 86_400_000)
  if (dayDifference < 0) return 'Overdue'
  if (dayDifference === 0) return 'Today'
  if (dayDifference === 1) return 'Tomorrow'
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

function toView(todo: Todo, client: BrowserServerClient): TodoView {
  const subtasks = todo.subtasks ?? []
  return {
    id: todo.id,
    title: todo.title,
    description: todo.description,
    status: todo.status,
    priority: todo.priority,
    pinned: todo.pinned,
    startDate: todo.start_date,
    dueDateLabel: formatDueDate(todo.start_date),
    tags: todo.tags ?? [],
    subtaskTotal: subtasks.length,
    subtaskCompleted: subtasks.filter((subtask) => subtask.status === 'completed').length,
    hasScreenshot: Boolean(todo.screenshot_path),
    screenshotUrl: todo.screenshot_path ? client.getScreenshotUrl(todo.id) : null,
    screenshotPath: todo.screenshot_path,
    updatedAtLabel: timeAgo(todo.updated_at),
    createdAtLabel: timeAgo(todo.created_at),
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
  const actionError = ref<string | null>(null)
  const isLoading = ref(false)
  const total = ref(0)
  const completed = ref(0)

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
      total.value = 0
      completed.value = 0
      stats.value = '0 todos · 0 done'
      errorMessage.value = null
      return
    }

    isLoading.value = true
    try {
      const todos = await client.value.getTodos(userId.value, currentDomain.value)
      items.value = todos.map((todo) => toView(todo, client.value!))
      const done = todos.filter((todo) => todo.status === 'completed').length
      total.value = todos.length
      completed.value = done
      stats.value = `${todos.length} todos · ${done} done`
      errorMessage.value = null
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      errorMessage.value = `Server not reachable. ${message}`
      total.value = 0
      completed.value = 0
      stats.value = '0 todos · 0 done'
      items.value = []
    } finally {
      isLoading.value = false
    }
  }

  let domainDetected = false

  async function init() {
    if (!domainDetected) {
      domainDetected = true
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
    }

    await refresh()
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

  async function add(title: string, description = '', priority: TodoPriority = 'medium', startDate = '', tags: string[] = []) {
    const trimmed = title.trim()
    if (!trimmed || !currentDomain.value || !client.value || !userId.value) {
      return false
    }

    try {
      const todo = await client.value.createTodo({
        user_id: userId.value,
        title: trimmed,
        description: description.trim() || undefined,
        domain: currentDomain.value,
        priority,
        start_date: startDate || null,
        tags,
      })

      if (screenshotPreview.value) {
        await client.value.uploadScreenshot(todo.id, dataUrlToBlob(screenshotPreview.value))
      }

      setPreview(null)
      actionError.value = null
      await refresh()
      return true
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      actionError.value = `Failed to add todo: ${message}`
      return false
    }
  }

  async function update(id: number, changes: UpdateTodoInput) {
    if (!client.value || !userId.value) {
      return false
    }

    const todo = items.value.find((item) => item.id === id)
    if (!todo) {
      return false
    }

    try {
      await client.value.updateTodo(id, {
        title: changes.title?.trim() || todo.title,
        description: changes.description?.trim() ?? todo.description,
        status: changes.status,
        priority: changes.priority,
        pinned: changes.pinned,
        start_date: changes.start_date,
        tags: changes.tags,
      })
      actionError.value = null
      await refresh()
      return true
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      actionError.value = `Failed to update todo: ${message}`
      return false
    }
  }

  async function toggle(id: number, completed: boolean) {
    await update(id, { status: completed ? 'completed' : 'pending' })
  }

  async function togglePin(id: number) {
    const todo = items.value.find((item) => item.id === id)
    if (todo) await update(id, { pinned: !todo.pinned })
  }

  async function remove(id: number) {
    if (!client.value) {
      return
    }

    try {
      await client.value.deleteTodo(id)
      actionError.value = null
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      actionError.value = `Failed to delete todo: ${message}`
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
      actionError.value = null
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      actionError.value = `Failed to clear todos: ${message}`
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
    total,
    completed,
    isLoading,
    errorMessage,
    actionError,
    init,
    refresh,
    add,
    update,
    toggle,
    togglePin,
    remove,
    clearAll,
  }
}
