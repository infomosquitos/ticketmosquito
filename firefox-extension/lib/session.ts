import type { Cookie } from './types';
import type { ApiClient } from './api';

function normalizeSameSite(value?: string): string {
  if (!value) return 'Lax';
  if (value === 'no_restriction' || value === 'none') return 'None';
  if (value === 'unspecified' || value === 'lax') return 'Lax';
  if (value === 'strict') return 'Strict';
  return value;
}

function cookieKey(c: chrome.cookies.Cookie & { storeId?: string; partitionKey?: unknown; firstPartyDomain?: string }): string {
  return `${c.storeId || ''}|${c.name}|${c.domain}|${c.path}|${JSON.stringify(c.partitionKey ?? null)}`;
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
 * Match SpaceMosquito Firefox capture: url + hostname, partitionKey, firstPartyDomain,
 * and the tab's cookieStoreId (Multi-Account Containers).
 */
async function getAllCookiesForTab(
  tabUrl: string,
  preferredStoreId?: string
): Promise<Array<chrome.cookies.Cookie & { storeId?: string }>> {
  const hostname = new URL(tabUrl).hostname;
  const domains = domainVariants(hostname);

  let storeIds: string[] = [];
  try {
    const stores = await chrome.cookies.getAllCookieStores();
    storeIds = stores.map((s) => s.id).filter(Boolean);
  } catch {
    /* ignore */
  }
  if (preferredStoreId && !storeIds.includes(preferredStoreId)) {
    storeIds.unshift(preferredStoreId);
  } else if (preferredStoreId) {
    storeIds = [preferredStoreId, ...storeIds.filter((id) => id !== preferredStoreId)];
  }
  if (storeIds.length === 0) {
    storeIds = preferredStoreId ? [preferredStoreId] : [];
  }

  const storeBases: Record<string, unknown>[] =
    storeIds.length > 0 ? storeIds.map((storeId) => ({ storeId })) : [{}];

  const queries: Record<string, unknown>[] = [];
  for (const base of storeBases) {
    queries.push(
      { ...base, url: tabUrl, partitionKey: {}, firstPartyDomain: null },
      { ...base, url: tabUrl, partitionKey: {} },
      { ...base, url: tabUrl, firstPartyDomain: null },
      { ...base, url: tabUrl }
    );
    for (const domain of domains) {
      queries.push(
        { ...base, domain, partitionKey: {}, firstPartyDomain: null },
        { ...base, domain, partitionKey: {} },
        { ...base, domain, firstPartyDomain: null },
        { ...base, domain }
      );
    }
  }

  const byKey = new Map<string, chrome.cookies.Cookie & { storeId?: string }>();
  for (const details of queries) {
    try {
      const batch = await chrome.cookies.getAll(details as chrome.cookies.GetAllDetails);
      for (const c of batch) {
        if (c?.name) byKey.set(cookieKey(c), c);
      }
    } catch {
      /* ignore unsupported filter combinations */
    }
  }
  return Array.from(byKey.values());
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
        name: String(c.name),
        value: String(c.value ?? ''),
        domain: String(c.domain || domain),
        path: String(c.path || '/'),
        expires,
        secure: Boolean(c.secure),
        httpOnly: Boolean(c.httpOnly),
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
