const SETTINGS_KEY = 'tracker_settings';
const DEFAULTS = { apiBase: 'http://localhost:8080', userId: '1', autoCapture: true };

async function getSettings() {
  const { [SETTINGS_KEY]: settings } = await chrome.storage.local.get(SETTINGS_KEY);
  return { ...DEFAULTS, ...settings };
}

// ── Tab switching ─────────────────────────────────

let activeTab = 'history';

function switchTab(tab) {
  activeTab = tab;
  document.getElementById('tab-history').classList.toggle('active', tab === 'history');
  document.getElementById('tab-todos').classList.toggle('active', tab === 'todos');
  document.getElementById('section-history').classList.toggle('active', tab === 'history');
  document.getElementById('section-todos').classList.toggle('active', tab === 'todos');
  document.getElementById('actions-history').style.display = tab === 'history' ? 'flex' : 'none';
  document.getElementById('actions-todos').style.display = tab === 'todos' ? 'flex' : 'none';

  if (tab === 'history') loadHistory();
  if (tab === 'todos') initTodosSection();
}

// ── Utilities ─────────────────────────────────────

function faviconUrl(url) {
  try {
    const host = new URL(url).hostname;
    return `https://www.google.com/s2/favicons?domain=${host}&sz=16`;
  } catch {
    return '';
  }
}

function timeAgo(ts) {
  const diff = Date.now() - Date.parse(ts);
  if (isNaN(diff)) return '';
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// ── History ────────────────────────────────────────

async function loadHistory() {
  const container = document.getElementById('history-list');
  const statsEl = document.getElementById('stats');
  const statusEl = document.getElementById('history-status');
  const { apiBase, userId } = await getSettings();

  try {
    const res = await fetch(`${apiBase}/api/history?user_id=${userId}`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const entries = await res.json();

    const grouped = {};
    for (const e of entries) {
      if (grouped[e.url]) {
        grouped[e.url].count += 1;
        if (new Date(e.visited_at) > new Date(grouped[e.url].lastVisited)) {
          grouped[e.url].lastVisited = e.visited_at;
          grouped[e.url].title = e.title || grouped[e.url].title;
        }
      } else {
        grouped[e.url] = {
          url: e.url,
          title: e.title || e.url,
          count: 1,
          lastVisited: e.visited_at
        };
      }
    }

    const sorted = Object.values(grouped).sort(
      (a, b) => Date.parse(b.lastVisited) - Date.parse(a.lastVisited)
    );

    const totalEntries = sorted.length;
    const totalVisits = sorted.reduce((sum, e) => sum + e.count, 0);
    statsEl.textContent = `${totalEntries} pages \u00b7 ${totalVisits} visits`;
    statusEl.textContent = '';

    if (sorted.length === 0) {
      container.innerHTML = '<div class="empty">No history yet.<br>Browse pages to start tracking.</div>';
      return;
    }

    container.innerHTML = sorted.map(e => `
      <div class="entry">
        <img class="favicon" src="${faviconUrl(e.url)}" width="16" height="16" onerror="this.style.display='none'">
        <div class="info">
          <div class="title" title="${escapeHtml(e.title)}">${escapeHtml(e.title || e.url)}</div>
          <div class="url" title="${escapeHtml(e.url)}">${escapeHtml(e.url)}</div>
          <div style="font-size:10px;color:#445566;margin-top:1px">${timeAgo(e.lastVisited)}</div>
        </div>
        <div class="count">${e.count}</div>
      </div>
    `).join('');

  } catch (err) {
    container.innerHTML = `<div class="empty">Server not reachable.<br><small>${escapeHtml(err.message)}</small></div>`;
    statsEl.textContent = '0 pages \u00b7 0 visits';
    statusEl.textContent = '';
  }
}

async function clearAllHistory() {
  const { apiBase, userId } = await getSettings();
  try {
    const res = await fetch(`${apiBase}/api/history?user_id=${userId}`);
    const entries = await res.json();
    for (const e of entries) {
      await fetch(`${apiBase}/api/history/${e.id}`, { method: 'DELETE' });
    }
  } catch (err) {
    console.debug('Clear failed', err.message);
  }
  await loadHistory();
}

// ── Todos ──────────────────────────────────────────

let currentDomain = null;
let lastScreenshotDataUrl = null;

async function getActiveDomain() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  try {
    return new URL(tab.url).hostname;
  } catch {
    return null;
  }
}

async function captureScreenshot() {
  return new Promise((resolve) => {
    chrome.runtime.sendMessage({ type: 'captureScreenshot' }, (response) => {
      resolve(response?.dataUrl || null);
    });
  });
}

function dataUrlToBlob(dataUrl) {
  const [header, base64] = dataUrl.split(',');
  const mime = (header.match(/data:(.*);base64/) || ['', 'image/png'])[1];
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return new Blob([bytes], { type: mime });
}

async function initTodosSection() {
  const { autoCapture } = await getSettings();
  currentDomain = await getActiveDomain();
  const display = document.getElementById('domain-display');
  if (currentDomain) {
    display.textContent = `Todos for: ${currentDomain}`;
  } else {
    display.textContent = 'Could not determine current domain.';
  }

  if (autoCapture && currentDomain) {
    lastScreenshotDataUrl = await captureScreenshot();
    const preview = document.getElementById('screenshot-preview');
    const img = document.getElementById('screenshot-img');
    if (lastScreenshotDataUrl) {
      img.src = lastScreenshotDataUrl;
      preview.style.display = 'flex';
    } else {
      preview.style.display = 'none';
    }
  } else {
    document.getElementById('screenshot-preview').style.display = 'none';
    lastScreenshotDataUrl = null;
  }

  await loadTodos();
}

async function loadTodos() {
  const container = document.getElementById('todo-list');
  const statsEl = document.getElementById('stats');
  const { apiBase, userId } = await getSettings();

  if (!currentDomain) {
    container.innerHTML = '<div class="empty">No active domain detected.</div>';
    return;
  }

  try {
    const res = await fetch(`${apiBase}/api/todos?user_id=${userId}&domain=${encodeURIComponent(currentDomain)}`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const todos = await res.json();

    const total = todos.length;
    const completed = todos.filter(t => t.completed).length;
    statsEl.textContent = `${total} todos \u00b7 ${completed} done`;

    if (todos.length === 0) {
      container.innerHTML = '<div class="empty">No todos for this site.</div>';
      return;
    }

    todos.sort((a, b) => a.completed - b.completed || new Date(b.updated_at) - new Date(a.updated_at));

    container.innerHTML = todos.map(t => `
      <div class="todo-entry">
        <input type="checkbox" class="todo-check" data-id="${t.id}" ${t.completed ? 'checked' : ''} />
        ${t.screenshot_path ? `<img class="screenshot-thumb" src="${apiBase}/api/screenshots/${t.id}" title="Screenshot" />` : ''}
        <div class="todo-info">
          <div class="todo-title ${t.completed ? 'done' : ''}" title="${escapeHtml(t.title)}">${escapeHtml(t.title)}</div>
          <div class="todo-meta">${timeAgo(t.updated_at)}</div>
        </div>
        <button class="todo-delete" data-id="${t.id}">Del</button>
      </div>
    `).join('');

  } catch (err) {
    container.innerHTML = `<div class="empty">Server not reachable.<br><small>${escapeHtml(err.message)}</small></div>`;
    statsEl.textContent = '0 todos \u00b7 0 done';
  }
}

async function addTodo() {
  const input = document.getElementById('todo-title');
  const title = input.value.trim();
  if (!title || !currentDomain) return;

  const { apiBase, userId } = await getSettings();

  try {
    const res = await fetch(`${apiBase}/api/todos`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: parseInt(userId, 10),
        title: title,
        domain: currentDomain
      })
    });

    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const todo = await res.json();

    if (lastScreenshotDataUrl) {
      const blob = dataUrlToBlob(lastScreenshotDataUrl);
      const formData = new FormData();
      formData.append('file', blob, 'screenshot.png');
      await fetch(`${apiBase}/api/screenshots?todo_id=${todo.id}`, {
        method: 'POST',
        body: formData
      });
    }

    input.value = '';
    lastScreenshotDataUrl = null;
    document.getElementById('screenshot-preview').style.display = 'none';
    await loadTodos();

  } catch (err) {
    document.getElementById('domain-display').textContent = 'Failed to add todo: ' + err.message;
  }
}

async function toggleTodo(id, completed) {
  const { apiBase, userId } = await getSettings();
  try {
    await fetch(`${apiBase}/api/todos/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: parseInt(userId, 10), completed: completed })
    });
    await loadTodos();
  } catch (err) {
    console.debug('Toggle failed', err.message);
  }
}

async function deleteTodo(id) {
  const { apiBase } = await getSettings();
  try {
    await fetch(`${apiBase}/api/todos/${id}`, { method: 'DELETE' });
    await loadTodos();
  } catch (err) {
    console.debug('Delete failed', err.message);
  }
}

async function clearAllTodos() {
  const { apiBase, userId } = await getSettings();
  if (!currentDomain) return;
  try {
    const res = await fetch(`${apiBase}/api/todos?user_id=${userId}&domain=${encodeURIComponent(currentDomain)}`);
    const todos = await res.json();
    for (const t of todos) {
      await fetch(`${apiBase}/api/todos/${t.id}`, { method: 'DELETE' });
    }
  } catch (err) {
    console.debug('Clear todos failed', err.message);
  }
  await loadTodos();
}

// ── Event delegation ───────────────────────────────

document.getElementById('todo-list').addEventListener('change', (e) => {
  if (e.target.classList.contains('todo-check')) {
    const id = parseInt(e.target.dataset.id, 10);
    toggleTodo(id, e.target.checked);
  }
});

document.getElementById('todo-list').addEventListener('click', (e) => {
  if (e.target.classList.contains('todo-delete')) {
    const id = parseInt(e.target.dataset.id, 10);
    deleteTodo(id);
  }
});

// ── Button handlers ────────────────────────────────

document.getElementById('tab-history').addEventListener('click', () => switchTab('history'));
document.getElementById('tab-todos').addEventListener('click', () => switchTab('todos'));
document.getElementById('btn-refresh').addEventListener('click', loadHistory);
document.getElementById('btn-clear').addEventListener('click', clearAllHistory);
document.getElementById('btn-add-todo').addEventListener('click', addTodo);
document.getElementById('btn-refresh-todos').addEventListener('click', loadTodos);
document.getElementById('btn-clear-todos').addEventListener('click', clearAllTodos);

document.getElementById('todo-title').addEventListener('keydown', (e) => {
  if (e.key === 'Enter') addTodo();
});

// ── Init ───────────────────────────────────────────

switchTab('history');
