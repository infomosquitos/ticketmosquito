import type { Cookie, SessionStatus } from './types';

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

  async captureSession(jiraUrl: string, cookies: Cookie[]): Promise<{ message: string }> {
    return this.request('/api/session', {
      method: 'POST',
      body: JSON.stringify({ jira_url: jiraUrl, cookies }),
    });
  }

  async getSessionStatus(jiraUrl?: string): Promise<SessionStatus> {
    const q = jiraUrl ? `?url=${encodeURIComponent(jiraUrl)}` : '';
    return this.request(`/api/session/status${q}`);
  }

  async validateSession(
    jiraUrl?: string,
    browserProbe?: unknown
  ): Promise<{ valid: boolean; message: string; flavor?: string }> {
    const body: Record<string, unknown> = {};
    if (jiraUrl) body.jira_url = jiraUrl;
    if (browserProbe) body.browser_probe = browserProbe;
    return this.request('/api/session/validate', {
      method: 'POST',
      body: JSON.stringify(body),
    });
  }

  async deleteSession(): Promise<{ message: string }> {
    return this.request('/api/session', { method: 'DELETE' });
  }
}
