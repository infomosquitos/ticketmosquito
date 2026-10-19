package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/infomosquitos/ticketmosquito/internal/store"
)

// UpsertProject inserts or updates a project by key.
func (d *DB) UpsertProject(ctx context.Context, key, name, url string) (uuid.UUID, error) {
	id := uuid.New()
	var out string
	err := d.sql.QueryRowContext(ctx,
		`INSERT INTO projects (id, key, name, url) VALUES (?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
		   name=CASE WHEN excluded.name != '' THEN excluded.name ELSE projects.name END,
		   url=CASE WHEN excluded.url != '' THEN excluded.url ELSE projects.url END
		 RETURNING id`,
		id.String(), key, name, url,
	).Scan(&out)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(out)
}

// GetProjectByKey loads a project.
func (d *DB) GetProjectByKey(ctx context.Context, key string) (*store.Project, error) {
	var p store.Project
	var idStr string
	var lastCrawled sql.NullString
	var createdAt string
	err := d.sql.QueryRowContext(ctx,
		`SELECT id, key, name, url, last_crawled, COALESCE(issues_total, 0), created_at
		 FROM projects WHERE key = ?`, key,
	).Scan(&idStr, &p.Key, &p.Name, &p.URL, &lastCrawled, &p.IssuesTotal, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	if lastCrawled.Valid {
		t, err := parseStoredTime(lastCrawled.String)
		if err != nil {
			return nil, err
		}
		p.LastCrawled = &t
	}
	p.CreatedAt, _ = parseStoredTime(createdAt)
	return &p, nil
}

// ListProjects returns all projects ordered by key.
func (d *DB) ListProjects(ctx context.Context) ([]store.Project, error) {
	rows, err := d.sql.QueryContext(ctx,
		`SELECT id, key, name, url, last_crawled, COALESCE(issues_total, 0), created_at
		 FROM projects ORDER BY key ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []store.Project
	for rows.Next() {
		var p store.Project
		var idStr string
		var lastCrawled sql.NullString
		var createdAt string
		if err := rows.Scan(&idStr, &p.Key, &p.Name, &p.URL, &lastCrawled, &p.IssuesTotal, &createdAt); err != nil {
			return nil, err
		}
		p.ID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		if lastCrawled.Valid {
			t, err := parseStoredTime(lastCrawled.String)
			if err != nil {
				return nil, err
			}
			p.LastCrawled = &t
		}
		p.CreatedAt, _ = parseStoredTime(createdAt)
		out = append(out, p)
	}
	if out == nil {
		out = []store.Project{}
	}
	return out, rows.Err()
}

// DeleteProject removes a project and cascaded issues.
func (d *DB) DeleteProject(ctx context.Context, key string) error {
	res, err := d.sql.ExecContext(ctx, `DELETE FROM projects WHERE key = ?`, key)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

// CountIssuesByProjectKey returns how many issues are stored for a project.
func (d *DB) CountIssuesByProjectKey(ctx context.Context, key string) (int, error) {
	var n int
	err := d.sql.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM issues i
		 JOIN projects p ON p.id = i.project_id
		 WHERE p.key = ?`, key,
	).Scan(&n)
	return n, err
}

// TouchProjectCrawl updates last_crawled and issues_total.
func (d *DB) TouchProjectCrawl(ctx context.Context, key string, issuesTotal int, lastCrawled time.Time) error {
	res, err := d.sql.ExecContext(ctx,
		`UPDATE projects SET last_crawled = ?, issues_total = ? WHERE key = ?`,
		formatTime(lastCrawled), issuesTotal, key,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}
