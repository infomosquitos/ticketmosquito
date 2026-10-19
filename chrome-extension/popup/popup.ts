// @ts-nocheck
import { ApiClient } from '../lib/api';
import { CrawlJob, CrawlProject, CronProjectConfig, ExtensionSettings, IssueInfo } from '../lib/types';

let api: ApiClient;
let healthPollInterval: number | null = null;
let crawlPollInterval: number | null = null;
let currentIssueInfo: IssueInfo | null = null;
let sessionValid = false;
/** Running/pending crawl jobs keyed by project_key */
const runningByProject = new Map<string, CrawlJob>();

type BackendState = 'unknown' | 'up' | 'down';
let backendState: BackendState = 'unknown';

const GATED_CONTROL_IDS = [
  'btn-refresh-issue',
  'btn-add-project',
  'btn-crawl-all',
];

let cronReloadTimer: number | null = null;
let yamlCronDefaults: {
  full_interval: string;
  incr_interval: string;
  detection: string;
} = { full_interval: '24h', incr_interval: '2h', detection: 'dom' };

function formatBackendHost(url: string): string {
  try {
    return new URL(url).host || url;
  } catch {
    return url;
  }
}

function formatShortDate(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });
}

function projectKeyFromUrl(url: string): string {
  if (!url) return '';
  try {
    const parsed = new URL(url);
    const proj =
      url.match(/\/projects\/([^/?#]+)/i) ||
      url.match(/\/jira(?:\/software)?\/c?\/projects\/([^/?#]+)/i);
    if (proj) return decodeURIComponent(proj[1]).toUpperCase();
    const browse = url.match(/\/browse\/([A-Z][A-Z0-9]+)-\d+/i);
    if (browse) return browse[1].toUpperCase();
    return (parsed.searchParams.get('projectKey') || '').toUpperCase();
  } catch {
    return '';
  }
}

function isBackendDown(): boolean {
  return backendState === 'down';
}

function actionsAllowed(): boolean {
  return backendState === 'up' && sessionValid;
}

function isIssueTitlePlaceholder(text: string | null): boolean {
  const t = (text || '').trim();
  return !t || t === '—' || t === 'Not on a Jira issue' || t === 'Could not read tab' || t === 'Jira';
}

/** Page-tab version line next to the refresh button. */
function formatIssueUpdatedMeta(liveUpdated?: string | null, storedUpdated?: string | null): string {
  const live = (liveUpdated || '').trim();
  const stored = (storedUpdated || '').trim();
  if (live && stored) {
    if (live === stored) return `${formatShortDate(live)} (Up to date)`;
    return `Live: ${formatShortDate(live)} · Stored: ${formatShortDate(stored)}`;
  }
  if (live) return `Live: ${formatShortDate(live)} · Stored: —`;
  if (stored) return `— · Stored: ${formatShortDate(stored)}`;
  return '—';
}

const BODY_FORMAT_LABELS: Record<string, { label: string; title: string }> = {
  adf: { label: 'ADF', title: 'Atlassian Document Format from REST API' },
  wiki: { label: 'Wiki', title: 'Jira wiki markup body' },
  rendered: { label: 'Rendered', title: 'Rendered HTML fallback' },
};

function setIssueFormat(bodyFormat?: string | null): void {
  const el = document.getElementById('issue-format');
  if (!el) return;
  const meta = bodyFormat ? BODY_FORMAT_LABELS[bodyFormat] : undefined;
  if (meta) {
    el.textContent = meta.label;
    el.title = meta.title;
    el.classList.remove('hidden');
  } else {
    el.textContent = '';
    el.removeAttribute('title');
    el.classList.add('hidden');
  }
}

function setControlEnabled(el: HTMLButtonElement | HTMLSelectElement | null, enabled: boolean): void {
  if (!el) return;
  el.disabled = !enabled;
}

function applyGating(): void {
  const banner = document.getElementById('backend-banner');
  const urlEl = document.getElementById('backend-banner-url');
  const down = isBackendDown();
  const allow = actionsAllowed();

  if (banner) {
    if (down) {
      banner.classList.remove('hidden');
      if (urlEl) urlEl.textContent = `at ${formatBackendHost(api.getBackendUrl())}`;
    } else {
      banner.classList.add('hidden');
    }
  }

  const hint = document.getElementById('issue-session-hint');
  if (hint) {
    if (!down && !sessionValid && currentIssueInfo?.tabUrl) {
      hint.classList.remove('hidden');
    } else {
      hint.classList.add('hidden');
    }
  }

  // Session chip stays clickable when backend up (even if session invalid)
  const chip = document.getElementById('session-chip') as HTMLButtonElement | null;
  if (chip && !chip.classList.contains('busy')) {
    chip.disabled = down;
  }

  for (const id of GATED_CONTROL_IDS) {
    const el = document.getElementById(id) as HTMLButtonElement | null;
    if (!el) continue;
    if (id === 'btn-refresh-issue') {
      el.disabled = !allow || !currentIssueInfo?.issueKey;
    } else {
      el.disabled = !allow;
    }
  }

  document.querySelectorAll<HTMLButtonElement | HTMLSelectElement | HTMLInputElement>(
    '#projects-list .btn-icon, #projects-list select, #projects-list input[type="checkbox"]'
  ).forEach((el) => {
    el.disabled = !allow;
  });
}

async function probeBackend(): Promise<void> {
  const prev = backendState;
  const ok = await api.checkHealth();
  backendState = ok ? 'up' : 'down';
  applyGating();

  if (prev !== 'up' && backendState === 'up') {
    await Promise.allSettled([
      loadSessionStatus(),
      loadIssueContext(),
      loadProjects(),
      syncRunningCrawls(),
    ]);
  }
}

function startBackendPolling(): void {
  if (healthPollInterval) clearInterval(healthPollInterval);
  void probeBackend();
  healthPollInterval = window.setInterval(() => {
    void probeBackend();
  }, 2000);
}

async function init() {
  const settings = await getSettings();
  api = new ApiClient(settings.backend_url);

  const urlInput = document.getElementById('backend-url') as HTMLInputElement | null;
  if (urlInput) urlInput.value = settings.backend_url;

  const autoRenew = document.getElementById('auto-renew') as HTMLInputElement | null;
  if (autoRenew) autoRenew.checked = !!settings.auto_renew;

  await setupTabs();
  startBackendPolling();

  // Session before page meta (compare needs valid session)
  await loadSessionStatus();
  await Promise.allSettled([
    loadIssueContext(),
    detectCurrentIssue(),
    loadProjects(),
    syncRunningCrawls(),
  ]);
}

const POPUP_TABS = new Set(['issue', 'projects', 'settings']);

function activatePopupTab(tab: string) {
  if (!POPUP_TABS.has(tab)) tab = 'issue';
  document.querySelectorAll('.tab-btn').forEach((t) => {
    t.classList.toggle('active', (t as HTMLElement).dataset.tab === tab);
  });
  document.querySelectorAll('.tab-content').forEach((c) => {
    c.classList.toggle('active', c.id === `tab-${tab}`);
  });
}

async function setupTabs() {
  try {
    const data: any = await chrome.storage.local.get('popup_last_tab');
    if (typeof data.popup_last_tab === 'string' && POPUP_TABS.has(data.popup_last_tab)) {
      activatePopupTab(data.popup_last_tab);
    }
  } catch { /* keep HTML default (Page) */ }

  document.querySelectorAll('.tab-btn').forEach((btn: HTMLElement) => {
    btn.addEventListener('click', () => {
      const tab = btn.dataset.tab || 'issue';
      activatePopupTab(tab);
      void chrome.storage.local.set({ popup_last_tab: tab });
    });
  });
}

async function getSettings(): Promise<ExtensionSettings & { backend_url: string; auto_renew: boolean }> {
  const data: any = await chrome.storage.local.get(['backend_url', 'auto_renew']);
  return {
    backend_url: data.backend_url || 'http://127.0.0.1:11043',
    crawl_depth: 'all',
    auto_renew: !!data.auto_renew,
  };
}

function setSessionDisc(state: 'valid' | 'invalid' | 'checking', title?: string) {
  const dot = document.getElementById('session-dot');
  const chip = document.getElementById('session-chip') as HTMLButtonElement | null;
  dot?.classList.remove('connected', 'disconnected', 'checking');
  if (state === 'valid') {
    sessionValid = true;
    dot?.classList.add('connected');
  } else if (state === 'invalid') {
    sessionValid = false;
    dot?.classList.add('disconnected');
  } else {
    // Keep prior sessionValid while checking so Issue hint / gates don't flicker.
    dot?.classList.add('checking');
  }
  if (chip) {
    chip.title = title || (state === 'valid'
      ? 'Session is valid'
      : state === 'invalid'
        ? 'Session is invalid or missing'
        : 'Checking session…');
  }
  applyGating();
}

async function loadSessionStatus() {
  setSessionDisc('checking', 'Checking session…');
  try {
    let tabUrl = '';
    try {
      const tabs: any[] = await chrome.tabs.query({ active: true, currentWindow: true });
      tabUrl = tabs[0]?.url || '';
    } catch { /* ignore */ }
    const status = await api.getSessionStatus(tabUrl || undefined);
    if (status.exists && status.valid) {
      setSessionDisc('valid', 'Session is valid');
    } else {
      setSessionDisc('invalid', 'Session is invalid or missing');
    }
  } catch {
    if (isBackendDown()) {
      setSessionDisc('checking', 'Waiting for backend…');
    } else {
      setSessionDisc('invalid', 'Session is invalid or missing');
    }
  }
}

async function captureAndValidate(): Promise<void> {
  if (isBackendDown()) return;
  const chip = document.getElementById('session-chip') as HTMLButtonElement | null;
  const hintBtn = document.getElementById('btn-refresh-session') as HTMLButtonElement | null;
  chip?.classList.add('busy');
  if (chip) chip.disabled = true;
  if (hintBtn) hintBtn.disabled = true;
  setSessionDisc('checking', 'Capturing & validating…');

  try {
    const result: any = await chrome.runtime.sendMessage({ type: 'capture-and-validate' });
    if (result?.success && result?.valid) {
      setSessionDisc('valid', 'Session is valid');
      await loadIssueContext();
    } else if (result?.success && result?.valid === false) {
      setSessionDisc('invalid', 'Session is invalid or missing');
    } else {
      setSessionDisc('invalid', 'Session is invalid or missing');
      alert(result?.error || 'Capture & validate failed');
    }
  } catch (error) {
    setSessionDisc('invalid', 'Session is invalid or missing');
    alert('Capture failed: ' + (error as Error).message);
  } finally {
    chip?.classList.remove('busy');
    if (hintBtn) hintBtn.disabled = false;
    applyGating();
  }
}

(document.getElementById('session-chip') as HTMLButtonElement)?.addEventListener('click', () => {
  void captureAndValidate();
});

(document.getElementById('btn-refresh-session') as HTMLButtonElement)?.addEventListener('click', () => {
  void captureAndValidate();
});

async function detectCurrentIssue() {
  try {
    const info: any = await chrome.runtime.sendMessage({ type: 'get-issue-info' });
    if (info?.projectKey || info?.issueKey) {
      currentIssueInfo = { ...currentIssueInfo, ...info };
    }
  } catch {
    /* ignore */
  }
  applyGating();
}

async function loadIssueContext() {
  const hostEl = document.getElementById('issue-host') as HTMLParagraphElement;
  const titleEl = document.getElementById('issue-title') as HTMLHeadingElement;
  const metaEl = document.getElementById('issue-meta') as HTMLSpanElement;

  try {
    const info: any = await chrome.runtime.sendMessage({ type: 'get-issue-info' });
    currentIssueInfo = info?.issueKey || info?.projectKey ? info : null;

    if (!info?.tabUrl && !info?.host) {
      hostEl.textContent = '—';
      titleEl.textContent = 'Not on a Jira issue';
      metaEl.textContent = '—';
      setIssueFormat(null);
      applyGating();
      return;
    }

    hostEl.textContent = info.host || '—';

    const tabTitle = info.summary || (info.projectKey ? `Project ${info.projectKey}` : 'Jira');
    const willFetchTitle = !!info.issueKey && backendState === 'up';
    // Avoid tab-title flash (e.g. "… - RADAR-base") when we already show the API title
    // and are about to refresh compare/stored.
    if (!willFetchTitle || isIssueTitlePlaceholder(titleEl.textContent)) {
      titleEl.textContent = tabTitle;
    }

    if (!info.issueKey) {
      metaEl.textContent = '—';
      setIssueFormat(null);
      applyGating();
      return;
    }

    if (isBackendDown() || !sessionValid) {
      // Still try stored-only if backend up
      if (backendState === 'up') {
        try {
          const stored = await api.getIssue(info.issueKey);
          metaEl.textContent = formatIssueUpdatedMeta(null, stored.updated);
          setIssueFormat(stored.body_format);
          if (stored.summary) titleEl.textContent = stored.summary;
        } catch {
          metaEl.textContent = '—';
          setIssueFormat(null);
          if (isIssueTitlePlaceholder(titleEl.textContent)) titleEl.textContent = tabTitle;
        }
      } else {
        metaEl.textContent = '—';
        setIssueFormat(null);
        if (isIssueTitlePlaceholder(titleEl.textContent)) titleEl.textContent = tabTitle;
      }
      applyGating();
      return;
    }

    try {
      const cmp = await api.compareIssue(info.issueKey);
      metaEl.textContent = formatIssueUpdatedMeta(cmp.live?.updated, cmp.stored?.updated);
      setIssueFormat(cmp.stored?.body_format);
      if (cmp.live?.summary) titleEl.textContent = cmp.live.summary;
      else if (cmp.stored?.summary) titleEl.textContent = cmp.stored.summary;
      else if (isIssueTitlePlaceholder(titleEl.textContent)) titleEl.textContent = tabTitle;
    } catch {
      try {
        const stored = await api.getIssue(info.issueKey);
        metaEl.textContent = formatIssueUpdatedMeta(null, stored.updated);
        setIssueFormat(stored.body_format);
        if (stored.summary) titleEl.textContent = stored.summary;
      } catch {
        metaEl.textContent = '—';
        setIssueFormat(null);
        if (isIssueTitlePlaceholder(titleEl.textContent)) titleEl.textContent = tabTitle;
      }
    }
  } catch {
    hostEl.textContent = '—';
    titleEl.textContent = 'Could not read tab';
    metaEl.textContent = '—';
    setIssueFormat(null);
  }
  applyGating();
}

(document.getElementById('btn-refresh-issue') as HTMLButtonElement)?.addEventListener('click', async () => {
  if (!actionsAllowed() || !currentIssueInfo?.issueKey) return;
  const btn = document.getElementById('btn-refresh-issue') as HTMLButtonElement;
  setControlEnabled(btn, false);
  try {
    await api.refreshIssue(currentIssueInfo.issueKey);
    await loadIssueContext();
  } catch (error) {
    alert('Refresh failed: ' + (error as Error).message);
  } finally {
    applyGating();
  }
});

function ensureCrawlPolling(): void {
  if (crawlPollInterval) return;
  crawlPollInterval = window.setInterval(() => {
    void syncRunningCrawls();
  }, 2000);
}

function stopCrawlPollingIfIdle(): void {
  if (runningByProject.size > 0) return;
  if (crawlPollInterval) {
    clearInterval(crawlPollInterval);
    crawlPollInterval = null;
  }
}

function applyJobToRow(projectKey: string, job: CrawlJob | null): void {
  const row = Array.from(document.querySelectorAll<HTMLElement>('.project-row[data-key]'))
    .find((r) => r.dataset.key === projectKey) || null;
  if (!row) return;

  const playBtn = row.querySelector('.btn-play') as HTMLButtonElement | null;
  const panel = row.querySelector('.project-crawl-panel') as HTMLElement | null;
  const fill = row.querySelector('.project-progress-fill') as HTMLElement | null;
  const label = row.querySelector('.project-progress-label') as HTMLElement | null;
  const counts = row.querySelector('.project-row-counts') as HTMLElement | null;

  const active = job && (job.status === 'running' || job.status === 'pending');
  if (active && job) {
    const percent = job.total_issues > 0
      ? Math.round((job.completed / job.total_issues) * 100)
      : (job.progress || 0);
    panel?.classList.remove('hidden');
    if (fill) fill.style.width = `${percent}%`;
    if (label) {
      label.textContent = job.total_issues > 0
        ? `Crawling… ${job.completed} / ${job.total_issues} · ${percent}%`
        : `Crawling… ${percent}%`;
    }
    if (counts && job.total_issues > 0) {
      counts.textContent = `${job.completed} / ${job.total_issues}`;
    }
    if (playBtn) {
      playBtn.textContent = '⏹';
      playBtn.title = 'Stop crawl';
      playBtn.classList.add('btn-stop');
      playBtn.dataset.jobId = job.id;
    }
  } else {
    panel?.classList.add('hidden');
    if (fill) fill.style.width = '0%';
    if (playBtn) {
      playBtn.textContent = '▶';
      playBtn.title = 'Crawl';
      playBtn.classList.remove('btn-stop');
      delete playBtn.dataset.jobId;
    }
  }
}

function renderAllCrawlRows(): void {
  document.querySelectorAll<HTMLElement>('.project-row[data-key]').forEach((row) => {
    const key = row.dataset.key || '';
    applyJobToRow(key, runningByProject.get(key) || null);
  });
  applyGating();
}

async function syncRunningCrawls(): Promise<void> {
  if (isBackendDown()) {
    stopCrawlPollingIfIdle();
    return;
  }
  try {
    const snapshot = await api.listCrawls();
    const prevKeys = new Set(runningByProject.keys());
    runningByProject.clear();

    for (const job of snapshot.jobs || []) {
      if (job.status !== 'running' && job.status !== 'pending') continue;
      const key = projectKeyFromUrl(job.project_url);
      if (!key) continue;
      runningByProject.set(key, job);
    }

    const finished = [...prevKeys].filter((k) => !runningByProject.has(k));
    renderAllCrawlRows();

    if (finished.length > 0) {
      await loadProjects();
      renderAllCrawlRows();
    }

    if (runningByProject.size > 0) ensureCrawlPolling();
    else stopCrawlPollingIfIdle();
  } catch (error) {
    console.error('[ticketmosquito] crawl sync error:', error);
  }
}

async function startProjectCrawl(projectUrl: string, projectKey?: string): Promise<void> {
  const key = projectKey || projectKeyFromUrl(projectUrl);
  if (key && runningByProject.has(key)) return;

  const result: any = await chrome.runtime.sendMessage({ type: 'start-crawl', projectUrl });
  if (!result?.success) {
    alert(result?.error || 'Crawl failed');
    return;
  }

  const jobId = result.jobId || result.job_id;
  const optimistic: CrawlJob = {
    id: jobId,
    project_url: projectUrl,
    status: 'running',
    progress: 0,
    total_issues: 0,
    completed: 0,
    failed: 0,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };
  if (key) {
    runningByProject.set(key, optimistic);
    applyJobToRow(key, optimistic);
  }
  ensureCrawlPolling();
  // Immediate sync so totals appear quickly
  void syncRunningCrawls();
}

async function stopProjectCrawl(projectKey: string): Promise<void> {
  const job = runningByProject.get(projectKey);
  if (!job) return;
  try {
    const result: any = await chrome.runtime.sendMessage({ type: 'cancel-crawl', jobId: job.id });
    if (!result?.success) {
      // Fallback: cancel via API client
      try {
        await api.cancelCrawl(job.id);
      } catch {
        alert('Cancel failed: ' + (result?.error || 'unknown error'));
        return;
      }
    }
    runningByProject.delete(projectKey);
    applyJobToRow(projectKey, null);
    stopCrawlPollingIfIdle();
    await loadProjects();
  } catch (error) {
    alert('Cancel failed: ' + (error as Error).message);
  }
}

(document.getElementById('btn-save-url') as HTMLButtonElement)?.addEventListener('click', async () => {
  const input = document.getElementById('backend-url') as HTMLInputElement;
  const url = input.value.trim();
  if (!url) return;
  try {
    api.setBackendUrl(url);
    await chrome.storage.local.set({ backend_url: url });
    await probeBackend();
    alert('Backend URL saved');
  } catch (error) {
    alert('Save failed: ' + (error as Error).message);
  }
});

(document.getElementById('auto-renew') as HTMLInputElement)?.addEventListener('change', async (e) => {
  const checked = (e.target as HTMLInputElement).checked;
  await chrome.storage.local.set({ auto_renew: checked });
  await chrome.runtime.sendMessage({ type: 'auto-renew-changed', enabled: checked });
});

(document.getElementById('btn-add-project') as HTMLButtonElement)?.addEventListener('click', async () => {
  if (!actionsAllowed()) return;
  const input = document.getElementById('add-project-url') as HTMLInputElement;
  const url = input.value.trim();
  if (!url) return;
  try {
    await api.addProject(url);
    input.value = '';
    await loadProjects();
  } catch (error) {
    alert('Add failed: ' + (error as Error).message);
  }
});

function formatCounts(space: CrawlProject): string {
  const crawled = space.issues_crawled ?? 0;
  const total = space.issues_total ?? 0;
  if (total > 0) return `${crawled} / ${total}`;
  if (crawled > 0) return `${crawled} / —`;
  return '— / —';
}

function formatProjectTooltip(_project: CrawlProject): string {
  return '';
}

function scheduleCronReload(): void {
  if (cronReloadTimer) clearTimeout(cronReloadTimer);
  cronReloadTimer = window.setTimeout(async () => {
    try {
      await api.reloadCron();
    } catch (err) {
      console.error('[ticketmosquito] cron reload failed:', err);
    }
  }, 600);
}

function optionList(values: string[], selected: string): string {
  return values.map((v) => `<option value="${v}" ${v === selected ? 'selected' : ''}>${v}</option>`).join('');
}

async function resolveCronDefaults(): Promise<void> {
  try {
    const cfg = await api.getCronConfig();
    if (cfg.yaml_full_crawl?.interval) yamlCronDefaults.full_interval = cfg.yaml_full_crawl.interval;
    if (cfg.yaml_incremental?.interval) yamlCronDefaults.incr_interval = cfg.yaml_incremental.interval;
    if (cfg.yaml_incremental?.detection) yamlCronDefaults.detection = cfg.yaml_incremental.detection;
  } catch {
    /* keep defaults */
  }
}

function cronEnabledFromOverride(ov: Partial<CronProjectConfig> | undefined): boolean {
  return !!(ov?.full_crawl_enabled || ov?.incr_crawl_enabled);
}

async function loadCronEnabledKeys(): Promise<Set<string>> {
  const keys = new Set<string>();
  try {
    const cfg = await api.getCronConfig();
    for (const ov of cfg.per_project_overrides || []) {
      if (cronEnabledFromOverride(ov) && ov.project_key) keys.add(ov.project_key);
    }
  } catch {
    /* ignore — icons stay disabled styling */
  }
  return keys;
}

function setCronButtonState(btn: Element | null, enabled: boolean): void {
  if (!btn) return;
  btn.classList.toggle('cron-on', enabled);
  btn.setAttribute('title', enabled ? 'Cron enabled' : 'Cron disabled');
}

async function toggleCronPanel(row: HTMLElement, space: CrawlProject): Promise<void> {
  const panel = row.querySelector('.project-cron-panel') as HTMLElement | null;
  if (!panel) return;
  const opening = panel.classList.contains('hidden');
  document.querySelectorAll('.project-cron-panel').forEach((p) => p.classList.add('hidden'));
  if (!opening) return;

  await resolveCronDefaults();
  let ov: Partial<CronProjectConfig> = {};
  try {
    ov = await api.getProjectCronConfig(space.project_key);
  } catch {
    ov = {};
  }

  const fullInterval = ov.full_crawl_interval || yamlCronDefaults.full_interval;
  const incrInterval = ov.incr_crawl_interval || yamlCronDefaults.incr_interval;
  const detection = ov.detection || yamlCronDefaults.detection;
  const enabled = cronEnabledFromOverride(ov);

  panel.innerHTML = `
    <div class="project-cron-row">
      <label>Full crawl</label>
      <select data-field="full_crawl_interval">${optionList(['1h','6h','12h','24h','48h','7d'], fullInterval)}</select>
    </div>
    <div class="project-cron-row">
      <label>Incremental</label>
      <select data-field="incr_crawl_interval">${optionList(['30m','1h','2h','4h','6h','12h'], incrInterval)}</select>
    </div>
    <div class="project-cron-row">
      <label>Detection</label>
      <select data-field="detection">${optionList(['api','dom'], detection)}</select>
    </div>
    <div class="project-cron-row">
      <label>Enabled</label>
      <input type="checkbox" data-field="enabled" ${enabled ? 'checked' : ''}>
    </div>
  `;
  panel.classList.remove('hidden');
  applyGating();

  const cronBtn = row.querySelector('.btn-cron');
  setCronButtonState(cronBtn, enabled);

  const persist = async () => {
    if (!actionsAllowed()) return;
    const fullSel = panel.querySelector('select[data-field="full_crawl_interval"]') as HTMLSelectElement;
    const incrSel = panel.querySelector('select[data-field="incr_crawl_interval"]') as HTMLSelectElement;
    const detSel = panel.querySelector('select[data-field="detection"]') as HTMLSelectElement;
    const en = panel.querySelector('input[data-field="enabled"]') as HTMLInputElement;
    setCronButtonState(cronBtn, en.checked);
    const payload: Partial<CronProjectConfig> = {
      project_key: space.project_key,
      project_url: space.project_url,
      full_crawl_interval: fullSel.value,
      incr_crawl_interval: incrSel.value,
      detection: detSel.value,
      full_crawl_enabled: en.checked,
      incr_crawl_enabled: en.checked,
    };
    try {
      await api.updateProjectCron(space.project_key, payload);
      scheduleCronReload();
    } catch (error) {
      alert('Cron save failed: ' + (error as Error).message);
    }
  };

  panel.querySelectorAll('select, input').forEach((el) => {
    el.addEventListener('change', () => { void persist(); });
  });
}

function sortProjectsForDisplay(spaces: CrawlProject[], currentKey: string): CrawlProject[] {
  const rest = spaces
    .filter((s) => s.project_key !== currentKey)
    .sort((a, b) => (a.project_key || '').localeCompare(b.project_key || '', undefined, { sensitivity: 'base' }));
  const current = spaces.find((s) => s.project_key === currentKey);
  return current ? [current, ...rest] : rest;
}

async function loadProjects() {
  const list = document.getElementById('projects-list');
  const crawlAllBtn = document.getElementById('btn-crawl-all');
  if (!list || !crawlAllBtn) return;

  try {
    if (!currentIssueInfo?.projectKey) {
      try {
        const info: any = await chrome.runtime.sendMessage({ type: 'get-issue-info' });
        if (info?.projectKey) currentIssueInfo = { ...currentIssueInfo, ...info };
      } catch { /* ignore */ }
    }

    let spaces = await api.listProjects();
    const cronEnabled = await loadCronEnabledKeys();
    const currentKey = currentIssueInfo?.projectKey || '';

    if (currentKey && !spaces.some((s) => s.project_key === currentKey)) {
      spaces = [
        {
          project_key: currentKey,
          project_name: currentIssueInfo?.spaceName || currentKey,
          project_url: currentIssueInfo?.spaceURL || '',
          issues_crawled: 0,
          issues_total: 0,
        },
        ...spaces,
      ];
    }

    const ordered = sortProjectsForDisplay(spaces, currentKey);
    list.innerHTML = '';

    if (ordered.length === 0) {
      crawlAllBtn.classList.add('hidden');
      list.innerHTML = '<p class="info-text">No projects yet. Add a project URL below.</p>';
      applyGating();
      return;
    }

    crawlAllBtn.classList.remove('hidden');

    ordered.forEach((space) => {
      const isCurrent = space.project_key === currentKey;
      const neverCrawled = !space.last_crawled && !(space.issues_crawled > 0) && !runningByProject.has(space.project_key);
      const dateLabel = space.last_crawled ? formatShortDate(space.last_crawled) : '—';
      const dateTitle = space.last_crawled ? new Date(space.last_crawled).toLocaleString() : 'Never crawled';
      const running = runningByProject.get(space.project_key);

      const countsTitle = formatProjectTooltip(space);
      const row = document.createElement('div');
      row.className = 'project-row' + (isCurrent ? ' current' : '');
      row.dataset.key = space.project_key;
      row.dataset.url = space.project_url || '';
      row.innerHTML = `
        <div class="project-row-main">
          <span class="project-row-name" title="${space.project_url || ''}">${space.project_key}</span>
          <span class="project-row-counts"${countsTitle ? ` title="${countsTitle}"` : ''}>${formatCounts(space)}</span>
          <span class="project-row-date" title="${dateTitle}">${dateLabel}</span>
          <button type="button" class="btn-icon btn-play${running ? ' btn-stop' : ''}" title="${running ? 'Stop crawl' : 'Crawl'}" data-url="${space.project_url}">${running ? '⏹' : '▶'}</button>
          <button type="button" class="btn-icon btn-cron${cronEnabled.has(space.project_key) ? ' cron-on' : ''}" title="${cronEnabled.has(space.project_key) ? 'Cron enabled' : 'Cron disabled'}">⏱</button>
        </div>
        ${neverCrawled && isCurrent ? '<div class="project-row-sub">Not crawled yet</div>' : ''}
        <div class="project-crawl-panel${running ? '' : ' hidden'}">
          <div class="project-progress-track"><div class="project-progress-fill" style="width:0%"></div></div>
          <div class="project-progress-label">Crawling…</div>
        </div>
        <div class="project-cron-panel hidden"></div>
      `;
      list.appendChild(row);

      if (running) applyJobToRow(space.project_key, running);

      row.querySelector('.btn-play')?.addEventListener('click', async (e) => {
        e.stopPropagation();
        if (!actionsAllowed()) return;
        const btn = e.currentTarget as HTMLElement;
        if (btn.classList.contains('btn-stop') || runningByProject.has(space.project_key)) {
          await stopProjectCrawl(space.project_key);
          return;
        }
        const url = btn.dataset.url;
        if (!url) {
          alert('Project has no URL');
          return;
        }
        try {
          await startProjectCrawl(url, space.project_key);
        } catch (error) {
          alert('Crawl failed: ' + (error as Error).message);
        }
      });

      row.querySelector('.btn-cron')?.addEventListener('click', async (e) => {
        e.stopPropagation();
        if (!actionsAllowed()) return;
        await toggleCronPanel(row, space);
      });
    });

    applyGating();
    if (runningByProject.size > 0) ensureCrawlPolling();
  } catch (error) {
    list.innerHTML = isBackendDown()
      ? '<p class="info-text">Waiting for backend…</p>'
      : `<p class="crawl-error">Failed to load: ${(error as Error).message}</p>`;
  }
}

(document.getElementById('btn-crawl-all') as HTMLButtonElement)?.addEventListener('click', async () => {
  if (!actionsAllowed()) return;
  try {
    const spaces = await api.listProjects();
    for (const space of spaces) {
      if (!space.project_url) continue;
      if (runningByProject.has(space.project_key)) continue;
      await startProjectCrawl(space.project_url, space.project_key);
      await new Promise((r) => setTimeout(r, 400));
    }
  } catch (error) {
    alert('Crawl all failed: ' + (error as Error).message);
  }
});

init();
