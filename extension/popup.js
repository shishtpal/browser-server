const SETTINGS_KEY = 'tracker_settings';
const DEFAULTS = { apiBase: 'http://localhost:8080', userId: '1' };

async function getSettings() {
  const { [SETTINGS_KEY]: settings } = await chrome.storage.local.get(SETTINGS_KEY);
  return { ...DEFAULTS, ...settings };
}

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

async function loadHistory() {
  const container = document.getElementById('history-list');
  const statsEl = document.getElementById('stats');
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
  }
}

function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

document.getElementById('btn-refresh').addEventListener('click', loadHistory);
document.getElementById('btn-clear').addEventListener('click', async () => {
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
});

loadHistory();
