import { createBrowserServerClient, type History, type Todo } from '@browser-server/shared-client'
import { captureVisibleTab, dataUrlToBlob, getActiveTabDomain } from '../lib/browser'
import { escapeHtml, faviconUrl, timeAgo } from '../lib/format'
import { getSettings } from '../lib/settings'
import '../styles/tailwind.css'

type ActivePanel = 'history' | 'todos'

let activePanel: ActivePanel = 'history'
let currentDomain: string | null = null
let lastScreenshotDataUrl: string | null = null

const app = document.querySelector<HTMLDivElement>('#app')
if (!app) {
  throw new Error('Popup root element not found')
}

app.innerHTML = `
  <main class="w-[480px] max-h-[540px] overflow-hidden rounded-xl border border-slate-800 bg-slate-900 shadow-2xl">
    <header class="border-b border-slate-800 bg-slate-950 px-4 py-3">
      <h1 class="text-base font-semibold text-rose-400">Browser Server</h1>
      <p id="stats" class="mt-1 text-xs text-slate-400">Loading...</p>
    </header>

    <nav class="grid grid-cols-2 border-b border-slate-800 bg-slate-950/60">
      <button id="tab-history" class="border-b-2 border-rose-400 px-3 py-2 text-sm font-medium text-rose-300">History</button>
      <button id="tab-todos" class="border-b-2 border-transparent px-3 py-2 text-sm font-medium text-slate-400">Todos</button>
    </nav>

    <section id="section-history" class="max-h-[392px] overflow-y-auto">
      <p id="history-status" class="px-4 py-2 text-center text-xs text-slate-500"></p>
      <div id="history-list"></div>
    </section>

    <section id="section-todos" class="hidden max-h-[392px] overflow-y-auto">
      <p id="domain-display" class="border-b border-slate-800 px-4 py-2 text-center text-xs text-slate-400"></p>
      <div id="screenshot-preview" class="hidden items-center gap-3 border-b border-slate-800 px-4 py-3">
        <img id="screenshot-img" alt="Screenshot preview" class="h-14 w-20 rounded border border-slate-700 object-cover" />
        <span class="text-xs text-slate-400">Screenshot captured for the next todo</span>
      </div>
      <div class="flex gap-2 border-b border-slate-800 px-4 py-3">
        <input id="todo-title" type="text" placeholder="Add todo for this page..." class="flex-1 rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 outline-none ring-0 placeholder:text-slate-500 focus:border-rose-400" />
        <button id="btn-add-todo" class="rounded-md bg-rose-500 px-3 py-2 text-sm font-medium text-white hover:bg-rose-400">Add</button>
      </div>
      <div id="todo-list"></div>
    </section>

    <footer class="border-t border-slate-800 bg-slate-950/70 px-4 py-3">
      <div id="actions-history" class="flex gap-2">
        <button id="btn-refresh" class="flex-1 rounded-md border border-slate-700 px-3 py-2 text-sm text-slate-300 hover:border-slate-500 hover:text-white">Refresh</button>
        <button id="btn-clear" class="flex-1 rounded-md border border-rose-800 px-3 py-2 text-sm text-rose-300 hover:bg-rose-500 hover:text-white">Clear All</button>
      </div>
      <div id="actions-todos" class="hidden gap-2">
        <button id="btn-refresh-todos" class="flex-1 rounded-md border border-slate-700 px-3 py-2 text-sm text-slate-300 hover:border-slate-500 hover:text-white">Refresh</button>
        <button id="btn-clear-todos" class="flex-1 rounded-md border border-rose-800 px-3 py-2 text-sm text-rose-300 hover:bg-rose-500 hover:text-white">Clear All</button>
      </div>
    </footer>
  </main>
`

const statsEl = document.querySelector<HTMLParagraphElement>('#stats')!
const historyStatusEl = document.querySelector<HTMLParagraphElement>('#history-status')!
const historyListEl = document.querySelector<HTMLDivElement>('#history-list')!
const todoListEl = document.querySelector<HTMLDivElement>('#todo-list')!
const domainDisplayEl = document.querySelector<HTMLParagraphElement>('#domain-display')!
const screenshotPreviewEl = document.querySelector<HTMLDivElement>('#screenshot-preview')!
const screenshotImgEl = document.querySelector<HTMLImageElement>('#screenshot-img')!
const todoTitleEl = document.querySelector<HTMLInputElement>('#todo-title')!
const sectionHistoryEl = document.querySelector<HTMLElement>('#section-history')!
const sectionTodosEl = document.querySelector<HTMLElement>('#section-todos')!
const tabHistoryEl = document.querySelector<HTMLButtonElement>('#tab-history')!
const tabTodosEl = document.querySelector<HTMLButtonElement>('#tab-todos')!
const actionsHistoryEl = document.querySelector<HTMLDivElement>('#actions-history')!
const actionsTodosEl = document.querySelector<HTMLDivElement>('#actions-todos')!

function setStats(label: string): void {
  statsEl.textContent = label
}

async function getClient() {
  const settings = await getSettings()
  return {
    client: createBrowserServerClient(settings.apiBase),
    userId: Number.parseInt(settings.userId, 10),
    apiBase: settings.apiBase,
    autoCapture: settings.autoCapture,
  }
}

function setActivePanel(panel: ActivePanel): void {
  activePanel = panel

  const isHistory = panel === 'history'
  tabHistoryEl.className = isHistory
    ? 'border-b-2 border-rose-400 px-3 py-2 text-sm font-medium text-rose-300'
    : 'border-b-2 border-transparent px-3 py-2 text-sm font-medium text-slate-400'
  tabTodosEl.className = isHistory
    ? 'border-b-2 border-transparent px-3 py-2 text-sm font-medium text-slate-400'
    : 'border-b-2 border-rose-400 px-3 py-2 text-sm font-medium text-rose-300'

  sectionHistoryEl.classList.toggle('hidden', !isHistory)
  sectionTodosEl.classList.toggle('hidden', isHistory)
  actionsHistoryEl.classList.toggle('hidden', !isHistory)
  actionsTodosEl.classList.toggle('hidden', isHistory)

  if (isHistory) {
    void loadHistory()
  } else {
    void initTodosSection()
  }
}

function renderEmpty(container: HTMLElement, message: string): void {
  container.innerHTML = `<div class="px-4 py-10 text-center text-sm text-slate-500">${message}</div>`
}

function summarizeHistory(entries: History[]) {
  const grouped = new Map<string, { url: string; title: string; count: number; lastVisited: string }>()

  for (const entry of entries) {
    const current = grouped.get(entry.url)
    if (!current) {
      grouped.set(entry.url, {
        url: entry.url,
        title: entry.title || entry.url,
        count: 1,
        lastVisited: entry.visited_at,
      })
      continue
    }

    current.count += 1
    if (Date.parse(entry.visited_at) > Date.parse(current.lastVisited)) {
      current.lastVisited = entry.visited_at
      current.title = entry.title || current.title
    }
  }

  return Array.from(grouped.values()).sort((left, right) => Date.parse(right.lastVisited) - Date.parse(left.lastVisited))
}

async function loadHistory(): Promise<void> {
  const { client, userId } = await getClient()

  try {
    const entries = await client.getHistory(userId)
    const groupedEntries = summarizeHistory(entries)
    const totalVisits = groupedEntries.reduce((sum, entry) => sum + entry.count, 0)

    setStats(`${groupedEntries.length} pages · ${totalVisits} visits`)
    historyStatusEl.textContent = ''

    if (groupedEntries.length === 0) {
      renderEmpty(historyListEl, 'No history yet. Browse pages to start tracking.')
      return
    }

    historyListEl.innerHTML = groupedEntries
      .map((entry) => {
        const title = escapeHtml(entry.title)
        const url = escapeHtml(entry.url)
        const icon = faviconUrl(entry.url)
        const meta = escapeHtml(timeAgo(entry.lastVisited))

        return `
          <article class="flex items-center gap-3 border-b border-slate-800 px-4 py-3 hover:bg-slate-800/40">
            <img src="${icon}" alt="" class="h-4 w-4 shrink-0 rounded-sm" onerror="this.style.display='none'" />
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-slate-100" title="${title}">${title}</p>
              <p class="truncate text-xs text-slate-500" title="${url}">${url}</p>
              <p class="mt-1 text-[11px] text-slate-500">${meta}</p>
            </div>
            <span class="rounded-full bg-rose-500/10 px-2.5 py-1 text-xs font-semibold text-rose-300">${entry.count}</span>
          </article>
        `
      })
      .join('')
  } catch (error) {
    const message = error instanceof Error ? escapeHtml(error.message) : 'Unknown error'
    renderEmpty(historyListEl, `Server not reachable. ${message}`)
    setStats('0 pages · 0 visits')
    historyStatusEl.textContent = ''
  }
}

async function clearAllHistory(): Promise<void> {
  const { client, userId } = await getClient()

  try {
    const entries = await client.getHistory(userId)
    await Promise.all(entries.map((entry) => client.deleteHistory(entry.id)))
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    console.debug('Clear history failed', message)
  }

  await loadHistory()
}

async function initTodosSection(): Promise<void> {
  const { autoCapture } = await getClient()
  currentDomain = await getActiveTabDomain()

  domainDisplayEl.textContent = currentDomain
    ? `Todos for: ${currentDomain}`
    : 'Could not determine current domain.'

  if (autoCapture && currentDomain) {
    lastScreenshotDataUrl = await captureVisibleTab()
    if (lastScreenshotDataUrl) {
      screenshotImgEl.src = lastScreenshotDataUrl
      screenshotPreviewEl.classList.remove('hidden')
      screenshotPreviewEl.classList.add('flex')
    } else {
      screenshotPreviewEl.classList.add('hidden')
      screenshotPreviewEl.classList.remove('flex')
    }
  } else {
    lastScreenshotDataUrl = null
    screenshotPreviewEl.classList.add('hidden')
    screenshotPreviewEl.classList.remove('flex')
  }

  await loadTodos()
}

function sortTodos(todos: Todo[]): Todo[] {
  return [...todos].sort((left, right) => Number(left.completed) - Number(right.completed) || Date.parse(right.updated_at) - Date.parse(left.updated_at))
}

async function loadTodos(): Promise<void> {
  if (!currentDomain) {
    renderEmpty(todoListEl, 'No active domain detected.')
    setStats('0 todos · 0 done')
    return
  }

  const { client, userId } = await getClient()

  try {
    const todos = sortTodos(await client.getTodos(userId, currentDomain))
    const completedCount = todos.filter((todo) => todo.completed).length

    setStats(`${todos.length} todos · ${completedCount} done`)

    if (todos.length === 0) {
      renderEmpty(todoListEl, 'No todos for this site.')
      return
    }

    todoListEl.innerHTML = todos
      .map((todo) => {
        const title = escapeHtml(todo.title)
        const updated = escapeHtml(timeAgo(todo.updated_at))
        const screenshot = todo.screenshot_path ? `<img src="${client.getScreenshotUrl(todo.id)}" alt="Screenshot" class="h-8 w-12 shrink-0 rounded border border-slate-700 object-cover" />` : ''

        return `
          <article class="flex items-center gap-3 border-b border-slate-800 px-4 py-3 hover:bg-slate-800/40">
            <input type="checkbox" class="todo-check h-4 w-4 shrink-0 accent-rose-500" data-id="${todo.id}" ${todo.completed ? 'checked' : ''} />
            ${screenshot}
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium ${todo.completed ? 'text-slate-500 line-through' : 'text-slate-100'}" title="${title}">${title}</p>
              <p class="mt-1 text-[11px] text-slate-500">${updated}</p>
            </div>
            <button class="todo-delete rounded px-2 py-1 text-xs text-slate-400 hover:bg-rose-500 hover:text-white" data-id="${todo.id}">Delete</button>
          </article>
        `
      })
      .join('')
  } catch (error) {
    const message = error instanceof Error ? escapeHtml(error.message) : 'Unknown error'
    renderEmpty(todoListEl, `Server not reachable. ${message}`)
    setStats('0 todos · 0 done')
  }
}

async function addTodo(): Promise<void> {
  const title = todoTitleEl.value.trim()
  if (!title || !currentDomain) {
    return
  }

  const { client, userId } = await getClient()

  try {
    const todo = await client.createTodo({
      user_id: userId,
      title,
      domain: currentDomain,
    })

    if (lastScreenshotDataUrl) {
      await client.uploadScreenshot(todo.id, dataUrlToBlob(lastScreenshotDataUrl))
    }

    todoTitleEl.value = ''
    lastScreenshotDataUrl = null
    screenshotPreviewEl.classList.add('hidden')
    screenshotPreviewEl.classList.remove('flex')
    await loadTodos()
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    domainDisplayEl.textContent = `Failed to add todo: ${message}`
  }
}

async function toggleTodo(id: number, completed: boolean): Promise<void> {
  const { client, userId } = await getClient()

  try {
    await client.updateTodo(id, { user_id: userId, completed })
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    console.debug('Toggle failed', message)
  }

  await loadTodos()
}

async function deleteTodo(id: number): Promise<void> {
  const { client } = await getClient()

  try {
    await client.deleteTodo(id)
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    console.debug('Delete failed', message)
  }

  await loadTodos()
}

async function clearAllTodos(): Promise<void> {
  if (!currentDomain) {
    return
  }

  const { client, userId } = await getClient()

  try {
    const todos = await client.getTodos(userId, currentDomain)
    await Promise.all(todos.map((todo) => client.deleteTodo(todo.id)))
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    console.debug('Clear todos failed', message)
  }

  await loadTodos()
}

todoListEl.addEventListener('change', (event) => {
  const target = event.target
  if (!(target instanceof HTMLInputElement) || !target.classList.contains('todo-check')) {
    return
  }

  const id = Number.parseInt(target.dataset.id || '', 10)
  if (Number.isNaN(id)) {
    return
  }

  void toggleTodo(id, target.checked)
})

todoListEl.addEventListener('click', (event) => {
  const target = event.target
  if (!(target instanceof HTMLElement) || !target.classList.contains('todo-delete')) {
    return
  }

  const id = Number.parseInt(target.dataset.id || '', 10)
  if (Number.isNaN(id)) {
    return
  }

  void deleteTodo(id)
})

tabHistoryEl.addEventListener('click', () => setActivePanel('history'))
tabTodosEl.addEventListener('click', () => setActivePanel('todos'))
document.querySelector<HTMLButtonElement>('#btn-refresh')!.addEventListener('click', () => void loadHistory())
document.querySelector<HTMLButtonElement>('#btn-clear')!.addEventListener('click', () => void clearAllHistory())
document.querySelector<HTMLButtonElement>('#btn-add-todo')!.addEventListener('click', () => void addTodo())
document.querySelector<HTMLButtonElement>('#btn-refresh-todos')!.addEventListener('click', () => void loadTodos())
document.querySelector<HTMLButtonElement>('#btn-clear-todos')!.addEventListener('click', () => void clearAllTodos())
todoTitleEl.addEventListener('keydown', (event) => {
  if (event.key === 'Enter') {
    void addTodo()
  }
})

setActivePanel(activePanel)
