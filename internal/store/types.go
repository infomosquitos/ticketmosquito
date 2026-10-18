package store

import (
	"time"

	"github.com/google/uuid"
)

// Project is a tracked Jira project.
type Project struct {
	ID          uuid.UUID
	Key         string
	Name        string
	URL         string
	LastCrawled *time.Time
	IssuesTotal int
	CreatedAt   time.Time
}

// Issue is a crawled Jira issue in the local catalog.
type Issue struct {
	ID           uuid.UUID
	ProjectID    uuid.UUID
	JiraID       string
	Key          string
	Summary      string
	Content      string // Markdown (description + comments)
	FieldsJSON   string // raw fields JSON
	FileDir      string
	ContentPath  string
	MetadataPath string
	BodyFormat   string
	JiraUpdated  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ProjectKey   string // filled on joins (get/search); not a DB column
}

// SearchResult is one FTS hit.
type SearchResult struct {
	Key        string
	JiraID     string
	ProjectKey string
	Summary    string
	Excerpt    string
	Similarity float64
	FilePath   string
	InternalID uuid.UUID
}

// CatalogStats is aggregate catalog info for health/stats.
type CatalogStats struct {
	TotalIssues    int    `json:"total_issues"`
	TotalProjects  int    `json:"total_projects"`
	ContentLang    string `json:"content_lang"`
	LastCrawledStr string `json:"last_crawled,omitempty"`
}
