import type { Cookie } from './types';
import type { ApiClient } from './api';

// Name substrings that usually carry Jira / SSO auth. Keep broad enough for
// corporate IdPs (OIDC, Keycloak, ALB) — still not every tracking cookie.
const AUTH_COOKIE_PATTERNS = [
  'session', 'token', 'sso', 'atlassian', 'aui',
  'seraph', 'crowd', 'remember', 'jsession',
  'openid', 'oidc', 'oauth', 'saml', 'keycloak', 'okta',
  'auth', 'awsalb', 'alb', 'mod_auth',
];

function shouldKeepCookie(name: string): boolean {
  const lower = name.toLowerCase();
  return AUTH_COOKIE_PATTERNS.some((p) => lower.includes(p));
}

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

/**
 * Collect cookies the browser would send to tabUrl (includes parent-domain SSO
 * cookies). Also scan hostname / .hostname for completeness.
 */
async function getAllCookiesForTab(tabUrl: string): Promise<chrome.cookies.Cookie[]> {
  const hostname = new URL(tabUrl).hostname;
  const queries: chrome.cookies.GetAllDetails[] = [
    { url: tabUrl },
    { domain: hostname },
    { domain: '.' + hostname },
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
  const filtered = rawCookies.filter((c) => c.name && shouldKeepCookie(c.name));

  const cookies = filtered.map((c) => ({
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
    const sample = rawNames.slice(0, 12).join(', ') || '(none)';
    return {
      success: false,
      cookieCount: 0,
      error:
        `No Jira/SSO session cookies matched. Browser saw: ${sample}. ` +
        `Stay logged in on this tab, then capture again.`,
    };
  }
  await apiClient.captureSession(jiraUrl, cookies);
  return { success: true, cookieCount: cookies.length };
}
