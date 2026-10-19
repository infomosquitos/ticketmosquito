import type {
  Cookie,
  CrawlJob,
  CrawlProject,
  CronProjectConfig,
  SessionStatus,
} from './types';

export class ApiClient {
  private baseUrl = 'http://127.0.0.1:11043';

  constructor(baseUrl?: string) {
    if (baseUrl) this.baseUrl = baseUrl.replace(/\/+$/, '');
  }

  getBackendUrl(): string {
    return this.baseUrl;
  }

  setBackendUrl(url: string): void {
    this.baseUrl = url.replace(/\/+$/, '');
  }

  async checkHealth(timeoutMs = 1500): Promise<boolean> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    try {
      const response = await fetch(`${this.baseUrl}/healthz`, {
        method: 'GET',
        signal: controller.signal,
      });
      if (!response.ok) return false;
      const text = (await response.text()).trim().toLowerCase();
      return text === 'ok' || text === '';
    } catch {
      return false;
    } finally {
      clearTimeout(timer);
    }
  }

  private async request<T>(path: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...(options?.headers as Record<string, string> | undefined),
      },
    });
    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`HTTP ${response.status}: ${errorText}`);
    }
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      return (await response.json()) as T;
    }
    return {} as T;
  }

  async captureSession(
    jiraUrl: string,
    cookies: Cookie[],
    captureDebug?: Record<string, unknown>
  ): Promise<{ message: string }> {
    const body: Record<string, unknown> = { jira_url: jiraUrl, cookies };
    if (captureDebug) {
      body.capture_debug = captureDebug;
    }
    return this.request('/api/session', {
      method: 'POST',
      body: JSON.stringify(body),
    });
  }

  async getSessionStatus(jiraUrl?: string): Promise<SessionStatus> {
    const q = jiraUrl ? `?url=${encodeURIComponent(jiraUrl)}` : '';
    return this.request(`/api/session/status${q}`);
  }

  async validateSession(jiraUrl?: string): Promise<{ valid: boolean; message: string; flavor?: string }> {
    return this.request('/api/session/validate', {
      method: 'POST',
      body: JSON.stringify(jiraUrl ? { jira_url: jiraUrl } : {}),
    });
  }

  async deleteSession(): Promise<{ message: string }> {
    return this.request('/api/session', { method: 'DELETE' });
  }

  async startCrawl(projectUrl: string, force = false): Promise<{ job_id: string }> {
    return this.request('/api/crawl', {
      method: 'POST',
      body: JSON.stringify({ project_url: projectUrl, force }),
    });
  }

  async listCrawls(): Promise<{ jobs: CrawlJob[]; total: number; running: number; completed: number; failed: number; pending: number }> {
    return this.request('/api/crawl');
  }

  async getCrawlStatus(jobId: string): Promise<CrawlJob> {
    return this.request(`/api/crawl/status?id=${encodeURIComponent(jobId)}`);
  }

  async cancelCrawl(jobId: string): Promise<void> {
    await this.request(`/api/crawl/cancel?id=${encodeURIComponent(jobId)}`, { method: 'POST' });
  }

  async addProject(url: string): Promise<{ message: string; project_key: string }> {
    return this.request('/api/projects', {
      method: 'POST',
      body: JSON.stringify({ url }),
    });
  }

  async listProjects(): Promise<CrawlProject[]> {
    return this.request('/api/projects');
  }

  async deleteProject(projectKey: string): Promise<void> {
    await this.request(`/api/projects/${encodeURIComponent(projectKey)}`, { method: 'DELETE' });
  }

  async getCronConfig(): Promise<{
    yaml_full_crawl: { enabled: boolean; interval: string; projects: string[]; max_duration: string };
    yaml_incremental: { enabled: boolean; interval: string; detection: string; projects: string[]; max_duration: string };
    per_project_overrides: CronProjectConfig[];
  }> {
    return this.request('/api/cron/config');
  }

  async updateProjectCron(projectKey: string, config: Partial<CronProjectConfig>): Promise<{ message: string }> {
    return this.request(`/api/cron/project/${encodeURIComponent(projectKey)}`, {
      method: 'POST',
      body: JSON.stringify(config),
    });
  }

  async getProjectCronConfig(projectKey: string): Promise<CronProjectConfig> {
    return this.request(`/api/cron/project/${encodeURIComponent(projectKey)}`);
  }

  async reloadCron(): Promise<{ message: string }> {
    return this.request('/api/cron/reload', { method: 'POST' });
  }

  async getIssue(issueKey: string): Promise<{
    issue_key: string;
    summary: string;
    updated?: string;
    body_format?: string;
  }> {
    return this.request(`/api/issues/${encodeURIComponent(issueKey)}`);
  }

  async compareIssue(issueKey: string): Promise<{
    issue_key: string;
    project_key: string;
    live: { updated?: string; summary?: string };
    stored: { updated?: string; summary?: string; body_format?: string } | null;
  }> {
    return this.request(`/api/issues/${encodeURIComponent(issueKey)}/compare`);
  }

  async refreshIssue(issueKey: string): Promise<{ issue_key: string; summary?: string; updated?: string }> {
    return this.request(`/api/issues/${encodeURIComponent(issueKey)}/refresh`, {
      method: 'POST',
      body: JSON.stringify({}),
    });
  }
}
