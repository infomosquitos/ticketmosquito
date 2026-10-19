import { isJiraUrl } from '../lib/session';

function $(id: string): HTMLElement {
  const el = document.getElementById(id);
  if (!el) throw new Error(`missing #${id}`);
  return el;
}

function setMsg(text: string): void {
  $('status-msg').textContent = text;
}

function setDot(state: 'ok' | 'bad' | 'checking'): void {
  const dot = $('session-dot');
  dot.classList.remove('ok', 'bad', 'checking');
  dot.classList.add(state);
}

async function send<T = unknown>(type: string, payload: Record<string, unknown> = {}): Promise<T> {
  return chrome.runtime.sendMessage({ type, ...payload }) as Promise<T>;
}

async function activeTab(): Promise<chrome.tabs.Tab | undefined> {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  return tabs[0];
}

async function refresh(): Promise<void> {
  const settings = await send<{ backendUrl: string }>('GET_SETTINGS');
  const input = $('backend-url') as HTMLInputElement;
  input.value = settings.backendUrl;

  const healthy = await send<{ ok: boolean }>('CHECK_HEALTH');
  const banner = $('backend-banner');
  const hostEl = $('backend-host');
  if (!healthy.ok) {
    banner.classList.remove('hidden');
    try {
      hostEl.textContent = new URL(settings.backendUrl).host;
    } catch {
      hostEl.textContent = settings.backendUrl;
    }
  } else {
    banner.classList.add('hidden');
  }

  const tab = await activeTab();
  const url = tab?.url || '';
  const onJira = isJiraUrl(url);
  $('page-host').textContent = onJira ? (() => {
    try { return new URL(url).host; } catch { return url; }
  })() : 'Not on Jira';
  $('page-hint').textContent = onJira
    ? 'Capture cookies from this tab into TicketMosquito.'
    : 'Open a Jira tab (Cloud or Server/DC) to capture a session.';

  const capture = $('btn-capture') as HTMLButtonElement;
  const validate = $('btn-validate') as HTMLButtonElement;
  const del = $('btn-delete') as HTMLButtonElement;
  capture.disabled = !healthy.ok || !onJira;
  validate.disabled = !healthy.ok;
  del.disabled = !healthy.ok;

  if (healthy.ok && onJira) {
    const status = await send<{ valid: boolean; exists: boolean; message?: string; flavor?: string }>(
      'SESSION_STATUS',
      { url }
    );
    if (status.valid) {
      setDot('ok');
      setMsg(`Session valid${status.flavor ? ` (${status.flavor})` : ''}`);
    } else if (status.exists) {
      setDot('bad');
      setMsg(status.message || 'Session needs validation');
    } else {
      setDot('checking');
      setMsg('No session for this host yet');
    }
  } else if (!healthy.ok) {
    setDot('bad');
    setMsg('Start ticketmosquito serve');
  } else {
    setDot('checking');
    setMsg('');
  }
}

document.addEventListener('DOMContentLoaded', () => {
  void refresh();

  $('btn-save-url').addEventListener('click', async () => {
    const url = ($('backend-url') as HTMLInputElement).value.trim();
    await send('SET_BACKEND_URL', { url });
    setMsg('Backend URL saved');
    await refresh();
  });

  $('btn-capture').addEventListener('click', async () => {
    const tab = await activeTab();
    if (!tab?.url) return;
    setMsg('Capturing…');
    const result = await send<{ success: boolean; cookieCount?: number; error?: string }>(
      'CAPTURE_SESSION',
      { url: tab.url }
    );
    if (result.success) {
      setMsg(`Saved ${result.cookieCount} cookies`);
    } else {
      setMsg(result.error || 'Capture failed');
    }
    await refresh();
  });

  $('btn-validate').addEventListener('click', async () => {
    const tab = await activeTab();
    setMsg('Validating…');
    const result = await send<{ valid: boolean; message?: string; flavor?: string }>(
      'VALIDATE_SESSION',
      { url: tab?.url }
    );
    await refresh();
    // refresh() would show the generic "requires validation" status; keep the
    // concrete probe message so SSO/cookie failures are visible.
    if (result.valid) {
      setDot('ok');
      setMsg(`Valid${result.flavor ? ` (${result.flavor})` : ''}: ${result.message || 'ok'}`);
    } else {
      setDot('bad');
      setMsg(result.message || 'Validation failed');
    }
  });

  $('btn-delete').addEventListener('click', async () => {
    await send('DELETE_SESSION');
    setMsg('Session cleared');
    await refresh();
  });
});
