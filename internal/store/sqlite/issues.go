package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/infomosquitos/ticketmosquito/internal/store"
)

// UpsertIssue inserts or updates an issue by project_id + jira_id (and key).
func (d *DB) UpsertIssue(ctx context.Context, issue *store.Issue) error {
	if issue == nil {
		return errors.New("issue is nil")
	}
	if issue.ID == uuid.Nil {
		issue.ID = uuid.New()
	}
	now := time.Now().UTC()
	issue.UpdatedAt = now
	if issue.CreatedAt.IsZero() {
		issue.CreatedAt = now
	}
	jiraUpdated := ""
	if issue.JiraUpdated != nil {
		jiraUpdated = formatTime(*issue.JiraUpdated)
	}
	_, err := d.sql.ExecContext(ctx,
		`INSERT INTO issues (
		   id, project_id, jira_id, key, summary, content, fields_json,
		   file_dir, content_path, metadata_path, body_format, jira_updated, created_at, updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(project_id, jira_id) DO UPDATE SET
		   key=excluded.key,
		   summary=excluded.summary,
		   content=excluded.content,
		   fields_json=excluded.fields_json,
		   file_dir=excluded.file_dir,
		   content_path=excluded.content_path,
		   metadata_path=excluded.metadata_path,
		   body_format=excluded.body_format,
		   jira_updated=excluded.jira_updated,
		   updated_at=excluded.updated_at`,
		issue.ID.String(), issue.ProjectID.String(), issue.JiraID, issue.Key, issue.Summary, issue.Content, issue.FieldsJSON,
		issue.FileDir, issue.ContentPath, issue.MetadataPath, issue.BodyFormat, jiraUpdated,
		formatTime(issue.CreatedAt), formatTime(issue.UpdatedAt),
	)
	return err
}

// GetIssueByKey loads an issue by KEY-N (includes project_key).
func (d *DB) GetIssueByKey(ctx context.Context, key string) (*store.Issue, error) {
	return d.scanIssue(ctx,
		`SELECT i.id, i.project_id, i.jira_id, i.key, i.summary, i.content, i.fields_json,
		        i.file_dir, i.content_path, i.metadata_path, i.body_format, i.jira_updated, i.created_at, i.updated_at,
		        p.key
		 FROM issues i
		 JOIN projects p ON p.id = i.project_id
		 WHERE i.key = ?`, strings.ToUpper(key))
}

// GetIssueByJiraID loads an issue by project key + numeric jira id.
func (d *DB) GetIssueByJiraID(ctx context.Context, projectKey, jiraID string) (*store.Issue, error) {
	return d.scanIssue(ctx,
		`SELECT i.id, i.project_id, i.jira_id, i.key, i.summary, i.content, i.fields_json,
		        i.file_dir, i.content_path, i.metadata_path, i.body_format, i.jira_updated, i.created_at, i.updated_at,
		        p.key
		 FROM issues i
		 JOIN projects p ON p.id = i.project_id
		 WHERE p.key = ? AND i.jira_id = ?`, strings.ToUpper(projectKey), jiraID)
}

func (d *DB) scanIssue(ctx context.Context, query string, args ...any) (*store.Issue, error) {
	var iss store.Issue
	var idStr, projectIDStr string
	var jiraUpdated, createdAt, updatedAt sql.NullString
	err := d.sql.QueryRowContext(ctx, query, args...).Scan(
		&idStr, &projectIDStr, &iss.JiraID, &iss.Key, &iss.Summary, &iss.Content, &iss.FieldsJSON,
		&iss.FileDir, &iss.ContentPath, &iss.MetadataPath, &iss.BodyFormat, &jiraUpdated, &createdAt, &updatedAt,
		&iss.ProjectKey,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	iss.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	iss.ProjectID, err = uuid.Parse(projectIDStr)
	if err != nil {
		return nil, err
	}
	if jiraUpdated.Valid && jiraUpdated.String != "" {
		t, err := parseStoredTime(jiraUpdated.String)
		if err != nil {
			return nil, err
		}
		iss.JiraUpdated = &t
	}
	if createdAt.Valid {
		iss.CreatedAt, _ = parseStoredTime(createdAt.String)
	}
	if updatedAt.Valid {
		iss.UpdatedAt, _ = parseStoredTime(updatedAt.String)
	}
	return &iss, nil
}
