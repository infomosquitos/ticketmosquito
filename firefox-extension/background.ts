import { ApiClient } from './lib/api';
import { captureAndSave, isJiraUrl, probeMyselfInBrowser } from './lib/session';

const DEFAULT_BACKEND_URL = 'http://127.0.0.1:11043';

const ACTION_ICON_ACTIVE = {
  16: 'assets/icon-active-16.png',
  32: 'assets/icon-active-32.png',
};
const ACTION_ICON_INACTIVE = {
  16: 'assets/icon-inactive-16.png',
  32: 'assets/icon-inactive-32.png',
};

async function getSettings(): Promise<{ backendUrl: string }> {
  const stored = await chrome.storage.local.get(['backend_url']);
  return {
    backendUrl: (stored.backend_url as string) || DEFAULT_BACKEND_URL,
  };
}

async function updateActionIconForTab(tabId: number, url?: string): Promise<void> {
  if (tabId < 0) return;
  const path = isJiraUrl(url || '') ? ACTION_ICON_ACTIVE : ACTION_ICON_INACTIVE;
  try {
    await chrome.action.setIcon({ tabId, path });
  } catch {
    /* ignore */
  }
}

chrome.tabs.onActivated.addListener(async (info) => {
  try {
    const tab = await chrome.tabs.get(info.tabId);
    await updateActionIconForTab(info.tabId, tab.url);
  } catch {
    /* ignore */
  }
});

chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.url || changeInfo.status === 'complete') {
    void updateActionIconForTab(tabId, tab.url);
  }
});

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  const type = message?.type as string;
  (async () => {
    const settings = await getSettings();
    const api = new ApiClient(settings.backendUrl);

    switch (type) {
      case 'GET_SETTINGS':
        sendResponse({ backendUrl: settings.backendUrl });
        return;
      case 'SET_BACKEND_URL': {
        const url = String(message.url || '').replace(/\/+$/, '');
        await chrome.storage.local.set({ backend_url: url });
        sendResponse({ ok: true, backendUrl: url });
        return;
      }
      case 'CHECK_HEALTH':
        sendResponse({ ok: await api.checkHealth() });
        return;
      case 'CAPTURE_SESSION': {
        const tabUrl = String(message.url || '');
        if (!isJiraUrl(tabUrl)) {
          sendResponse({ success: false, error: 'Not on a Jira page' });
          return;
        }
        try {
          const cookieStoreId = message.cookieStoreId
            ? String(message.cookieStoreId)
            : undefined;
          const result = await captureAndSave(tabUrl, api, cookieStoreId);
          if (result.success) {
            try {
              const browserProbe = await probeMyselfInBrowser(tabUrl);
              await api.validateSession(tabUrl, browserProbe);
            } catch {
              /* validate best-effort */
            }
          }
          sendResponse(result);
        } catch (err) {
          sendResponse({ success: false, error: (err as Error).message });
        }
        return;
      }
      case 'SESSION_STATUS': {
        try {
          const status = await api.getSessionStatus(message.url);
          sendResponse(status);
        } catch (err) {
          sendResponse({ valid: false, exists: false, message: (err as Error).message });
        }
        return;
      }
      case 'VALIDATE_SESSION': {
        try {
          const tabUrl = String(message.url || '');
          const browserProbe = tabUrl ? await probeMyselfInBrowser(tabUrl) : undefined;
          const result = await api.validateSession(tabUrl || undefined, browserProbe);
          sendResponse(result);
        } catch (err) {
          sendResponse({ valid: false, message: (err as Error).message });
        }
        return;
      }
      case 'DELETE_SESSION': {
        try {
          const result = await api.deleteSession();
          sendResponse(result);
        } catch (err) {
          sendResponse({ error: (err as Error).message });
        }
        return;
      }
      default:
        sendResponse({ error: 'unknown message type' });
    }
  })();
  return true;
});
