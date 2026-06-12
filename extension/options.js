const SETTINGS_KEY = 'tracker_settings';
const DEFAULTS = {
  apiBase: 'http://localhost:8080',
  userId: '1'
};

async function loadSettings() {
  const { [SETTINGS_KEY]: settings } = await chrome.storage.local.get(SETTINGS_KEY);
  const merged = { ...DEFAULTS, ...settings };

  document.getElementById('api-base').value = merged.apiBase;
  document.getElementById('user-id').value = merged.userId;
}

function showStatus(msg, ok) {
  const el = document.getElementById('status');
  el.textContent = msg;
  el.className = ok ? 'status-ok' : 'status-err';
  setTimeout(() => { el.textContent = ''; el.className = ''; }, 2500);
}

async function saveSettings() {
  const apiBase = document.getElementById('api-base').value.trim();
  const userId = document.getElementById('user-id').value.trim();

  if (!apiBase) {
    showStatus('Server URL is required.', false);
    return;
  }

  try {
    new URL(apiBase);
  } catch {
    showStatus('Invalid server URL.', false);
    return;
  }

  if (!userId || isNaN(parseInt(userId, 10)) || parseInt(userId, 10) < 1) {
    showStatus('User ID must be a positive number.', false);
    return;
  }

  await chrome.storage.local.set({
    [SETTINGS_KEY]: { apiBase, userId }
  });

  showStatus('Settings saved.', true);
}

document.getElementById('btn-save').addEventListener('click', saveSettings);
document.getElementById('btn-reset').addEventListener('click', async () => {
  await chrome.storage.local.remove(SETTINGS_KEY);
  document.getElementById('api-base').value = DEFAULTS.apiBase;
  document.getElementById('user-id').value = DEFAULTS.userId;
  showStatus('Reset to defaults.', true);
});

loadSettings();
