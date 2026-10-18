PRAGMA foreign_keys = ON;

CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    last_crawled TEXT,
    issues_total INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE issues (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    jira_id TEXT NOT NULL,
    key TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    content TEXT,
    fields_json TEXT,
    file_dir TEXT,
    content_path TEXT,
    metadata_path TEXT,
    body_format TEXT,
    jira_updated TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(project_id, jira_id),
    UNIQUE(key)
);

CREATE INDEX idx_issues_project ON issues(project_id);
CREATE INDEX idx_issues_jira_updated ON issues(jira_updated);
