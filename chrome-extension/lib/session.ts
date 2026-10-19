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
  return `${c.name}|${c.domain}|${c.path}|${c.value}|${JSON.stringify(partition ?? null)}`;
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

function domainVariants(hostname: string): string[] {
  return [hostname, '.' + hostname];
}

/**
 * Collect cookies for a Jira tab.
 *
 * Important Chrome quirk (CHIPS): getAll with `url` + `partitionKey` can miss
 * partitioned cookies. Prefer `domain` + `partitionKey: {}` (returns both
 * partitioned and unpartitioned), then merge URL-scoped unpartitioned results.
 */
async function getAllCookiesForTab(
  tabUrl: string,
  preferredStoreId?: string
): Promise<{ cookies: chrome.cookies.Cookie[]; queryHits: Record<string, number> }> {
  const u = new URL(tabUrl);
  const hostname = u.hostname;
  const origin = u.origin;
  const domains = domainVariants(hostname);

  const storeBases: chrome.cookies.GetAllDetails[] = preferredStoreId
    ? [{ storeId: preferredStoreId }, {}]
    : [{}];

  const queries: { label: string; details: chrome.cookies.GetAllDetails }[] = [];
  for (const base of storeBases) {
    const storeLabel = (base as { storeId?: string }).storeId || 'default';

    // Unpartitioned URL match (what the browser would send for this URL).
    queries.push(
      { label: `${storeLabel}:url`, details: { ...base, url: tabUrl } },
      { label: `${storeLabel}:origin`, details: { ...base, url: origin + '/' } },
    );

    // CHIPS: domain + empty partitionKey — do NOT combine with url.
    for (const domain of domains) {
      queries.push(
        { label: `${storeLabel}:domain:${domain}`, details: { ...base, domain } },
        {
          label: `${storeLabel}:domain+partAny:${domain}`,
          details: { ...base, domain, partitionKey: {} },
        },
        {
          label: `${storeLabel}:domain+partTLS:${domain}`,
          details: { ...base, domain, partitionKey: { topLevelSite: origin } },
        },
        {
          label: `${storeLabel}:domain+partTLS0:${domain}`,
          details: {
            ...base,
            domain,
            partitionKey: { topLevelSite: origin, hasCrossSiteAncestor: false },
          },
        },
        {
          label: `${storeLabel}:domain+partTLS1:${domain}`,
          details: {
            ...base,
            domain,
            partitionKey: { topLevelSite: origin, hasCrossSiteAncestor: true },
          },
        },
      );
    }

    // Whole jar with partitionKey:{} (then host-filter below).
    queries.push({
      label: `${storeLabel}:jar+partAny`,
      details: { ...base, partitionKey: {} },
    });
  }

  const byKey = new Map<string, chrome.cookies.Cookie>();
  const queryHits: Record<string, number> = {};

  for (const q of queries) {
    try {
      const batch = await chrome.cookies.getAll(q.details);
      queryHits[q.label] = batch.length;
      for (const c of batch) {
        if (!c?.name) continue;
        // For jar-wide queries, keep only cookies for this host / parent.
        if (q.label.includes('jar+')) {
          let dom = (c.domain || '').toLowerCase();
          if (dom.startsWith('.')) dom = dom.slice(1);
          const host = hostname.toLowerCase();
          if (!dom || !(host === dom || host.endsWith('.' + dom) || dom.endsWith('.' + host))) {
            continue;
          }
        }
        byKey.set(cookieKey(c), c);
      }
    } catch {
      queryHits[q.label] = -1;
    }
  }

  return { cookies: Array.from(byKey.values()), queryHits };
}

export type CaptureDebug = {
  rawCount: number;
  keptCount: number;
  rawNames: string[];
  keptMeta: string[];
  queryHits: Record<string, number>;
};

export async function captureCookies(
  tabUrl: string,
  cookieStoreId?: string
): Promise<{
  cookies: Cookie[];
  debug: CaptureDebug;
}> {
  const domain = new URL(tabUrl).hostname;
  const { cookies: rawCookies, queryHits } = await getAllCookiesForTab(tabUrl, cookieStoreId);
  const rawNames = [...new Set(rawCookies.map((c) => c.name).filter(Boolean))];

  const mapped = rawCookies
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

  // Exact-duplicate only (name+domain+path+value). Keep same-name variants.
  const byIdentity = new Map<string, Cookie>();
  for (const c of mapped) {
    byIdentity.set(`${c.name}|${c.domain}|${c.path}|${c.value}`, c);
  }
  const cookies = Array.from(byIdentity.values());

  const debug: CaptureDebug = {
    rawCount: rawCookies.length,
    keptCount: cookies.length,
    rawNames,
    keptMeta: cookies.map(
      (c) => `${c.name}@${c.domain} path=${c.path} len=${c.value.length} httpOnly=${c.httpOnly}`
    ),
    queryHits,
  };

  console.log('[ticketmosquito] cookie capture', { tabUrl, cookieStoreId, ...debug });

  return { cookies, debug };
}

export async function captureAndSave(
  jiraUrl: string,
  apiClient: ApiClient,
  cookieStoreId?: string
): Promise<{ success: boolean; cookieCount: number; error?: string; debug?: CaptureDebug }> {
  const { cookies, debug } = await captureCookies(jiraUrl, cookieStoreId);
  if (cookies.length === 0) {
    return {
      success: false,
      cookieCount: 0,
      debug,
      error:
        `No cookies visible for this tab (saw: ${debug.rawNames.slice(0, 12).join(', ') || 'none'}). ` +
        `Stay logged in, then capture again.`,
    };
  }
  await apiClient.captureSession(jiraUrl, cookies, debug);
  return { success: true, cookieCount: cookies.length, debug };
}
