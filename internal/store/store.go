package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Store is the catalog persistence interface.
type Store interface {
	Close()
	Ping(ctx context.Context) error

	UpsertProject(ctx context.Context, key, name, url string) (uuid.UUID, error)
	GetProjectByKey(ctx context.Context, key string) (*Project, error)
	ListProjects(ctx context.Context) ([]Project, error)
	TouchProjectCrawl(ctx context.Context, key string, issuesTotal int, lastCrawled time.Time) error

	UpsertIssue(ctx context.Context, issue *Issue) error
	GetIssueByKey(ctx context.Context, key string) (*Issue, error)
	GetIssueByJiraID(ctx context.Context, projectKey, jiraID string) (*Issue, error)

	SearchIssues(ctx context.Context, query, projectKey string, limit int) ([]SearchResult, error)
	RebuildIssuesFTS(ctx context.Context) error
	GetCatalogStats(ctx context.Context) (*CatalogStats, error)
}
