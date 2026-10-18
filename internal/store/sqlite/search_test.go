package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/datastore"
	"github.com/infomosquitos/ticketmosquito/internal/store"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
	"go.uber.org/zap"
)

func TestSearchIssues_FTS(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", Path: filepath.Join(dir, "s.db")},
	}
	log := zap.NewNop()
	if err := datastore.MigrateUp(cfg, "", logging.New("test", log)); err != nil {
		t.Fatal(err)
	}
	db, err := datastore.Open(cfg, log)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	pid, err := db.UpsertProject(ctx, "DEMO", "Demo", "https://example/browse/DEMO")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertIssue(ctx, &store.Issue{
		ID: uuid.New(), ProjectID: pid, JiraID: "1", Key: "DEMO-1",
		Summary: "Login button broken",
		Content: "Users cannot authenticate with SSO after the upgrade.",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertIssue(ctx, &store.Issue{
		ID: uuid.New(), ProjectID: pid, JiraID: "2", Key: "DEMO-2",
		Summary: "Unrelated ticket",
		Content: "Something about widgets only.",
	}); err != nil {
		t.Fatal(err)
	}

	hits, err := db.SearchIssues(ctx, "login authenticate", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected FTS hits")
	}
	if hits[0].Key != "DEMO-1" {
		t.Fatalf("top hit = %s, want DEMO-1", hits[0].Key)
	}

	filtered, err := db.SearchIssues(ctx, "login", "DEMO", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) == 0 {
		t.Fatal("expected project-filtered hits")
	}

	iss, err := db.GetIssueByKey(ctx, "demo-1")
	if err != nil {
		t.Fatal(err)
	}
	if iss.ProjectKey != "DEMO" || iss.Summary == "" {
		t.Fatalf("issue = %+v", iss)
	}

	projects, err := db.ListProjects(ctx)
	if err != nil || len(projects) != 1 {
		t.Fatalf("projects=%v err=%v", projects, err)
	}

	stats, err := db.GetCatalogStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalIssues != 2 || stats.ContentLang != "fts5" {
		t.Fatalf("stats=%+v", stats)
	}
}
