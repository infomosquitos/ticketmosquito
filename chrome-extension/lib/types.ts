export interface Cookie {
  name: string;
  value: string;
  domain: string;
  path: string;
  expires?: number;
  secure: boolean;
  httpOnly: boolean;
  sameSite?: string;
}

export interface SessionStatus {
  valid: boolean;
  message: string;
  exists: boolean;
  host?: string;
  flavor?: string;
}

export interface CrawlJob {
  id: string;
  project_url: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  progress: number;
  total_issues: number;
  completed: number;
  failed: number;
  error?: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface JobSnapshot {
  jobs: CrawlJob[];
  total: number;
  running: number;
  completed: number;
  failed: number;
  pending: number;
}

export interface CrawlProject {
  project_key: string;
  project_name: string;
  project_url: string;
  issues_crawled: number;
  issues_total?: number;
  last_crawled?: string;
  created_at?: string;
}

export interface CronProjectConfig {
  project_key: string;
  project_url: string;
  full_crawl_enabled: boolean;
  full_crawl_interval: string;
  incr_crawl_enabled: boolean;
  incr_crawl_interval: string;
  detection: string;
}

export interface ExtensionSettings {
  backend_url: string;
  auto_renew?: boolean;
}

export interface IssueInfo {
  projectKey: string;
  projectName: string;
  projectURL: string;
  issueKey: string;
  summary: string;
  host?: string;
  tabUrl?: string;
}
