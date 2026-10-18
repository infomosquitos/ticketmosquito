package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/infomosquitos/ticketmosquito/internal/store"
)

// SearchIssues runs FTS5 BM25 search (summary boosted) with title AND fallback.
func (d *DB) SearchIssues(ctx context.Context, query, projectKey string, limit int) ([]store.SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	results, err := d.searchIssuesFTS(ctx, query, projectKey, limit)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 {
		return results, nil
	}
	return d.searchIssuesSummaryFallback(ctx, query, projectKey, limit)
}

func (d *DB) searchIssuesFTS(ctx context.Context, query, projectKey string, limit int) ([]store.SearchResult, error) {
	ftsQuery := buildFTSQuery(query)
	baseQuery := `
		SELECT i.key, i.jira_id, p.key, i.summary,
		       snippet(issues_fts, 2, '', '', '...', 20) AS excerpt,
		       bm25(issues_fts, 0.0, 10.0, 1.0) AS similarity,
		       i.content_path, i.id
		FROM issues_fts
		JOIN issues i ON i.id = issues_fts.issue_id
		JOIN projects p ON p.id = i.project_id
		WHERE issues_fts MATCH ?`
	args := []any{ftsQuery}
	if projectKey != "" {
		baseQuery += " AND p.key = ?"
		args = append(args, strings.ToUpper(projectKey))
	}
	baseQuery += " ORDER BY similarity LIMIT ?"
	args = append(args, limit)

	rows, err := d.sql.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search issues: %w", err)
	}
	defer rows.Close()
	return scanSearchResults(rows)
}

func (d *DB) searchIssuesSummaryFallback(ctx context.Context, query, projectKey string, limit int) ([]store.SearchResult, error) {
	words := searchQueryTokens(query)
	if len(words) == 0 {
		return nil, nil
	}

	baseQuery := `
		SELECT i.key, i.jira_id, p.key, i.summary,
		       substr(i.summary, 1, 200) AS excerpt,
		       0.0 AS similarity,
		       i.content_path, i.id
		FROM issues i
		JOIN projects p ON p.id = i.project_id
		WHERE 1=1`
	args := make([]any, 0, len(words)+2)
	for _, w := range words {
		baseQuery += " AND lower(i.summary) LIKE ?"
		args = append(args, "%"+strings.ToLower(w)+"%")
	}
	if projectKey != "" {
		baseQuery += " AND p.key = ?"
		args = append(args, strings.ToUpper(projectKey))
	}
	baseQuery += " ORDER BY i.summary LIMIT ?"
	args = append(args, limit)

	rows, err := d.sql.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search issues summary fallback: %w", err)
	}
	defer rows.Close()
	return scanSearchResults(rows)
}

func scanSearchResults(rows *sql.Rows) ([]store.SearchResult, error) {
	var results []store.SearchResult
	for rows.Next() {
		var r store.SearchResult
		var idStr string
		var filePath sql.NullString
		if err := rows.Scan(&r.Key, &r.JiraID, &r.ProjectKey, &r.Summary, &r.Excerpt, &r.Similarity, &filePath, &idStr); err != nil {
			return nil, err
		}
		if filePath.Valid {
			r.FilePath = filePath.String
		}
		var err error
		r.InternalID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if results == nil {
		results = []store.SearchResult{}
	}
	return results, rows.Err()
}

// RebuildIssuesFTS rebuilds the FTS index from issues rows.
func (d *DB) RebuildIssuesFTS(ctx context.Context) error {
	_, err := d.sql.ExecContext(ctx, `INSERT INTO issues_fts(issues_fts) VALUES('rebuild')`)
	if err != nil {
		return fmt.Errorf("rebuild fts index: %w", err)
	}
	return nil
}

// GetCatalogStats returns issue/project counts and FTS status.
func (d *DB) GetCatalogStats(ctx context.Context) (*store.CatalogStats, error) {
	stats := &store.CatalogStats{}
	var lastCrawled sql.NullString
	err := d.sql.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM issues),
		       (SELECT COUNT(*) FROM projects),
		       (SELECT MAX(last_crawled) FROM projects),
		       CASE WHEN EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name='issues_fts')
		            THEN 'fts5' ELSE 'none' END
	`).Scan(&stats.TotalIssues, &stats.TotalProjects, &lastCrawled, &stats.ContentLang)
	if err != nil {
		return nil, err
	}
	if lastCrawled.Valid && lastCrawled.String != "" {
		stats.LastCrawledStr = lastCrawled.String
	}
	return stats, nil
}

func searchQueryTokens(q string) []string {
	return strings.Fields(strings.TrimSpace(q))
}

func buildFTSQuery(q string) string {
	words := searchQueryTokens(q)
	if len(words) == 0 {
		return q
	}
	for i, w := range words {
		w = strings.ReplaceAll(w, `"`, `""`)
		words[i] = `"` + w + `"`
	}
	return strings.Join(words, " AND ")
}
