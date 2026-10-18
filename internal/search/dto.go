package search

import (
	"time"

	"github.com/google/uuid"
	"github.com/infomosquitos/ticketmosquito/internal/store"
)

// SearchHit is the public search result shape (MCP, REST, CLI).
type SearchHit struct {
	Key        string  `json:"key"`
	JiraID     string  `json:"jira_id"`
	ProjectKey string  `json:"project_key"`
	Summary    string  `json:"summary"`
	Excerpt    string  `json:"excerpt"`
	Similarity float64 `json:"similarity"`
	FilePath   string  `json:"file_path,omitempty"`
	InternalID string  `json:"internal_id,omitempty"`
}

// IssueDetail is the public full-issue shape for get-issue.
type IssueDetail struct {
	Key         string     `json:"key"`
	JiraID      string     `json:"jira_id"`
	ProjectKey  string     `json:"project_key"`
	Summary     string     `json:"summary"`
	Content     string     `json:"content"`
	BodyFormat  string     `json:"body_format,omitempty"`
	JiraUpdated *time.Time `json:"jira_updated,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
	InternalID  string     `json:"internal_id,omitempty"`
}

// ProjectSummary is a list-projects row.
type ProjectSummary struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	URL         string     `json:"url,omitempty"`
	IssuesTotal int        `json:"issues_total"`
	LastCrawled *time.Time `json:"last_crawled,omitempty"`
}

// ToSearchHits maps DB rows to API-facing search hits.
func ToSearchHits(results []store.SearchResult, exposeInternalIDs bool) []SearchHit {
	if len(results) == 0 {
		return []SearchHit{}
	}
	hits := make([]SearchHit, len(results))
	for i, r := range results {
		hits[i] = SearchHit{
			Key:        r.Key,
			JiraID:     r.JiraID,
			ProjectKey: r.ProjectKey,
			Summary:    r.Summary,
			Excerpt:    NormalizeExcerpt(r.Excerpt, DefaultExcerptMaxRunes),
			Similarity: r.Similarity,
			FilePath:   r.FilePath,
		}
		if exposeInternalIDs && r.InternalID != uuid.Nil {
			hits[i].InternalID = r.InternalID.String()
		}
	}
	return hits
}

// ToIssueDetail maps a DB issue to the public detail shape.
func ToIssueDetail(iss *store.Issue, exposeInternalIDs bool) IssueDetail {
	d := IssueDetail{
		Key:         iss.Key,
		JiraID:      iss.JiraID,
		ProjectKey:  iss.ProjectKey,
		Summary:     iss.Summary,
		Content:     iss.Content,
		BodyFormat:  iss.BodyFormat,
		JiraUpdated: iss.JiraUpdated,
		UpdatedAt:   iss.UpdatedAt,
	}
	if exposeInternalIDs && iss.ID != uuid.Nil {
		d.InternalID = iss.ID.String()
	}
	return d
}

// ToProjectSummaries maps store projects to API rows.
func ToProjectSummaries(projects []store.Project) []ProjectSummary {
	if len(projects) == 0 {
		return []ProjectSummary{}
	}
	out := make([]ProjectSummary, len(projects))
	for i, p := range projects {
		out[i] = ProjectSummary{
			Key: p.Key, Name: p.Name, URL: p.URL,
			IssuesTotal: p.IssuesTotal, LastCrawled: p.LastCrawled,
		}
	}
	return out
}
