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
  const partition = (c as chrome.cookies.Cookie & { partitionKey?: unknown }).partitionKey;
  return `${c.name}|${c.domain}|${c.path}|${JSON.stringify(partition ?? null)}`;
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

/** Hostname only (SpaceMosquito lock): do not scrape parent domains for SSO. */
function domainVariants(hostname: string): string[] {
  return [hostname, '.' + hostname];
}

/**
 * Match SpaceMosquito Firefox capture: url + hostname queries, plus partitionKey
 * so CHIPS / partitioned SSO cookies are not missed. Prefer the tab store when set.
 */
async function getAllCookiesForTab(
  tabUrl: string,
  preferredStoreId?: string
): Promise<chrome.cookies.Cookie[]> {
  const hostname = new URL(tabUrl).hostname;
  const domains = domainVariants(hostname);

  const storeBases: chrome.cookies.GetAllDetails[] = preferredStoreId
    ? [{ storeId: preferredStoreId }, {}]
    : [{}];

  const queries: chrome.cookies.GetAllDetails[] = [];
  for (const base of storeBases) {
    queries.push(
      { ...base, url: tabUrl, partitionKey: {} },
      { ...base, url: tabUrl }
    );
    for (const domain of domains) {
      queries.push(
        { ...base, domain, partitionKey: {} },
        { ...base, domain }
      );
    }
  }

  const byKey = new Map<string, chrome.cookies.Cookie>();
  for (const q of queries) {
    try {
      const batch = await chrome.cookies.getAll(q);
      for (const c of batch) {
        if (c?.name) byKey.set(cookieKey(c), c);
      }
    } catch {
      /* partitionKey/storeId unsupported combination */
    }
  }

  // Fallback: scan the whole jar and keep cookies that apply to this URL.
  // Catches cookies getAll({url}) sometimes misses under corporate SSO.
  try {
    const all = await chrome.cookies.getAll({});
    for (const c of all) {
      if (c?.name && cookieAppliesToUrl(c, tabUrl)) {
        byKey.set(cookieKey(c), c);
      }
    }
  } catch {
    /* ignore */
  }

  return Array.from(byKey.values());
}

function cookieAppliesToUrl(c: chrome.cookies.Cookie, tabUrl: string): boolean {
  try {
    const u = new URL(tabUrl);
    if (c.secure && u.protocol !== 'https:') return false;
    const host = u.hostname.toLowerCase();
    let dom = (c.domain || '').toLowerCase();
    if (dom.startsWith('.')) dom = dom.slice(1);
    if (!dom) return false;
    if (host !== dom && !host.endsWith('.' + dom)) return false;
    const path = c.path || '/';
    if (!u.pathname.startsWith(path)) {
      // path prefix match with boundary
      if (path !== '/' && !u.pathname.startsWith(path.endsWith('/') ? path : path + '/')) {
        return false;
      }
    }
    return true;
  } catch {
    return false;
  }
}

export async function captureCookies(
  tabUrl: string,
  cookieStoreId?: string
): Promise<{
  cookies: Cookie[];
  rawCount: number;
  rawNames: string[];
}> {
  const domain = new URL(tabUrl).hostname;
  const rawCookies = await getAllCookiesForTab(tabUrl, cookieStoreId);
  const rawNames = [...new Set(rawCookies.map((c) => c.name).filter(Boolean))];

  const cookies = rawCookies
    .filter((c) => c.name)
    .map((c) => {
      const expRaw = c.expirationDate;
      let expires: number | undefined;
      if (expRaw != null && expRaw !== -1) {
        const n = Math.floor(Number(expRaw));
        if (Number.isFinite(n) && n > 0) expires = n;
      }
      return {
        name: c.name,
        value: c.value ?? '',
        domain: c.domain || domain,
        path: c.path || '/',
        expires,
        secure: !!c.secure,
        httpOnly: !!c.httpOnly,
        sameSite: normalizeSameSite(c.sameSite),
      };
    });

  console.log('[ticketmosquito] cookie capture', {
    tabUrl,
    cookieStoreId,
    rawCount: rawCookies.length,
    rawNames,
    keptNames: cookies.map((c) => c.name),
    keptDomains: cookies.map((c) => `${c.name}@${c.domain}`),
  });

  return { cookies, rawCount: rawCookies.length, rawNames };
}

export async function captureAndSave(
  jiraUrl: string,
  apiClient: ApiClient,
  cookieStoreId?: string
): Promise<{ success: boolean; cookieCount: number; error?: string }> {
  const { cookies, rawNames } = await captureCookies(jiraUrl, cookieStoreId);
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

export type BrowserProbeResult = {
  path: string;
  status: number;
  ok: boolean;
  contentType: string;
  location: string;
  preview: string;
};

/** Probe /myself using the browser cookie jar (credentials:include). */
export async function probeMyselfInBrowser(jiraUrl: string): Promise<BrowserProbeResult[]> {
  const root = new URL(jiraUrl).origin;
  const paths = ['/rest/api/2/myself', '/rest/api/3/myself', '/rest/auth/1/session'];
  const out: BrowserProbeResult[] = [];
  for (const path of paths) {
    try {
      const resp = await fetch(root + path, {
        method: 'GET',
        credentials: 'include',
        redirect: 'manual',
        headers: {
          Accept: 'application/json',
          'X-Atlassian-Token': 'no-check',
        },
      });
      const contentType = resp.headers.get('content-type') || '';
      const location = resp.headers.get('location') || '';
      let preview = '';
      try {
        preview = (await resp.text()).slice(0, 240).replace(/\s+/g, ' ').trim();
      } catch {
        /* ignore */
      }
      const result: BrowserProbeResult = {
        path,
        status: resp.status,
        ok: resp.status === 200 && contentType.includes('json'),
        contentType,
        location,
        preview,
      };
      out.push(result);
      console.log('[ticketmosquito] browser probe', result);
    } catch (err) {
      const result: BrowserProbeResult = {
        path,
        status: 0,
        ok: false,
        contentType: '',
        location: '',
        preview: String(err),
      };
      out.push(result);
      console.log('[ticketmosquito] browser probe error', result);
    }
  }
  return out;
}
