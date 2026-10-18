import type { Cookie } from './types';
import type { ApiClient } from './api';

const ATLASSIAN_COOKIE_PATTERNS = [
  'session', 'token', 'sso', 'atlassian', 'aui',
  'seraph', 'crowd', 'remember', 'jsession',
];

function shouldKeepCookie(name: string): boolean {
  const lower = name.toLowerCase();
  return ATLASSIAN_COOKIE_PATTERNS.some((p) => lower.includes(p));
}

function normalizeSameSite(value?: string): string {
  if (!value) return 'Lax';
  if (value === 'no_restriction' || value === 'none') return 'None';
  if (value === 'unspecified' || value === 'lax') return 'Lax';
  if (value === 'strict') return 'Strict';
  return value;
}

export function isJiraUrl(url: string): boolean {
  if (!url) return false;
  const lower = url.toLowerCase();
  return (
    lower.includes('atlassian.net') ||
    lower.includes('/browse/') ||
    lower.includes('/jira/') ||
    lower.includes('/secure/') ||
    lower.includes('selectedissue=') ||
    /\/projects\/[^/]+/.test(lower)
  );
}

export async function captureCookies(tabUrl: string): Promise<Cookie[]> {
  const url = new URL(tabUrl);
  const domain = url.hostname;
  const rawCookies = await chrome.cookies.getAll({ domain });
  const filtered = rawCookies.filter((c) => c.name && shouldKeepCookie(c.name));

  return filtered.map((c) => ({
    name: c.name,
    value: c.value,
    domain: c.domain || domain,
    path: c.path || '/',
    expires: c.expirationDate,
    secure: !!c.secure,
    httpOnly: !!c.httpOnly,
    sameSite: normalizeSameSite(c.sameSite),
  }));
}

export async function captureAndSave(
  jiraUrl: string,
  apiClient: ApiClient
): Promise<{ success: boolean; cookieCount: number }> {
  const cookies = await captureCookies(jiraUrl);
  if (cookies.length === 0) {
    return { success: false, cookieCount: 0 };
  }
  await apiClient.captureSession(jiraUrl, cookies);
  return { success: true, cookieCount: cookies.length };
}
