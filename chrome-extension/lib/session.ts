import type { Cookie } from './types';
import type { ApiClient } from './api';

function normalizeSameSite(value?: string): string {
  if (!value) return 'Lax';
  if (value === 'no_restriction' || value === 'none') return 'None';
  if (value === 'unspecified' || value === 'lax') return 'Lax';
  if (value === 'strict') return 'Strict';
  return value;
}

function cookieKey(c: chrome.cookies.Cookie): string {
  return `${c.name}|${c.domain}|${c.path}`;
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

/** Hostname plus registrable parents (jira.a.b → a.b, b skipped if too short). */
function domainCandidates(hostname: string): string[] {
  const parts = hostname.split('.').filter(Boolean);
  const out: string[] = [];
  for (let i = 0; i < parts.length - 1; i++) {
    const d = parts.slice(i).join('.');
    if (d.includes('.')) {
      out.push(d);
      out.push('.' + d);
    }
  }
  return [...new Set(out)];
}

/**
 * Collect every cookie the browser would send to tabUrl, plus explicit parent-
 * domain scans. Corporate SSO often uses unusual cookie names — do not filter
 * by name (a name allowlist caused 401s with only JSESSIONID kept).
 */
async function getAllCookiesForTab(tabUrl: string): Promise<chrome.cookies.Cookie[]> {
  const hostname = new URL(tabUrl).hostname;
  const queries: chrome.cookies.GetAllDetails[] = [
    { url: tabUrl },
    ...domainCandidates(hostname).map((domain) => ({ domain })),
  ];
  const byKey = new Map<string, chrome.cookies.Cookie>();
  for (const q of queries) {
    try {
      const batch = await chrome.cookies.getAll(q);
      for (const c of batch) {
        if (c?.name) byKey.set(cookieKey(c), c);
      }
    } catch {
      /* ignore query shape failures */
    }
  }
  return Array.from(byKey.values());
}

export async function captureCookies(tabUrl: string): Promise<{
  cookies: Cookie[];
  rawCount: number;
  rawNames: string[];
}> {
  const domain = new URL(tabUrl).hostname;
  const rawCookies = await getAllCookiesForTab(tabUrl);
  const rawNames = [...new Set(rawCookies.map((c) => c.name).filter(Boolean))];

  const cookies = rawCookies
    .filter((c) => c.name)
    .map((c) => ({
      name: c.name,
      value: c.value,
      domain: c.domain || domain,
      path: c.path || '/',
      expires: c.expirationDate,
      secure: !!c.secure,
      httpOnly: !!c.httpOnly,
      sameSite: normalizeSameSite(c.sameSite),
    }));

  return { cookies, rawCount: rawCookies.length, rawNames };
}

export async function captureAndSave(
  jiraUrl: string,
  apiClient: ApiClient
): Promise<{ success: boolean; cookieCount: number; error?: string }> {
  const { cookies, rawNames } = await captureCookies(jiraUrl);
  if (cookies.length === 0) {
    return {
      success: false,
      cookieCount: 0,
      error:
        `No cookies visible for this tab (saw: ${rawNames.slice(0, 12).join(', ') || 'none'}). ` +
        `Stay logged in, then capture again.`,
    };
  }
  await apiClient.captureSession(jiraUrl, cookies);
  return { success: true, cookieCount: cookies.length };
}
