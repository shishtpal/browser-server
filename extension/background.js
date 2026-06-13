const SETTINGS_KEY = 'tracker_settings';
const DEFAULTS = { apiBase: 'http://localhost:8080', userId: '1' };

let lastUrl = null;

async function getSettings() {
  const { [SETTINGS_KEY]: settings } = await chrome.storage.local.get(SETTINGS_KEY);
  return { ...DEFAULTS, ...settings };
}

async function postVisit(url, title) {
  if (!url || url.startsWith('chrome://') || url.startsWith('chrome-extension://') || url.startsWith('edge://') || url.startsWith('brave://') || url.startsWith('opera://') || url.startsWith('browser://') || url === 'about:blank') {
    return;
  }

  const { apiBase, userId } = await getSettings();

  try {
    await fetch(`${apiBase}/api/history`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: parseInt(userId, 10),
        url: url,
        title: title || url,
        duration: 0
      })
    });
  } catch (err) {
    console.debug('History sync failed (server offline?)', err.message);
  }
}

chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.url) {
    if (changeInfo.url !== lastUrl) {
      lastUrl = changeInfo.url;
      postVisit(changeInfo.url, tab.title);
    }
  } else if (changeInfo.status === 'complete' && tab.url) {
    lastUrl = tab.url;
    postVisit(tab.url, tab.title);
  }
});

chrome.tabs.onActivated.addListener((activeInfo) => {
  chrome.tabs.get(activeInfo.tabId, (tab) => {
    if (tab.url && tab.url !== lastUrl) {
      lastUrl = tab.url;
      postVisit(tab.url, tab.title);
    }
  });
});

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.type === 'captureScreenshot') {
    chrome.tabs.captureVisibleTab(null, { format: 'png' }, (dataUrl) => {
      sendResponse({ dataUrl });
    });
    return true;
  }
});
