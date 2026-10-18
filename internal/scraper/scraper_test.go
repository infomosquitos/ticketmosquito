package scraper_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/datastore"
	"github.com/infomosquitos/ticketmosquito/internal/scraper"
	"github.com/infomosquitos/ticketmosquito/internal/session"
	"github.com/infomosquitos/ticketmosquito/internal/storage"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
	"go.uber.org/zap"
)

const testKey = "0123456789abcdef0123456789abcdef"

func TestCrawlProject_cloudFixture(t *testing.T) {
	adfDesc := json.RawMessage(`{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello ticket"}]}]}`)
	adfComment := json.RawMessage(`{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"A comment"}]}]}`)

	var searchCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/search/jql"):
			searchCalls++
			_, _ = w.Write([]byte(`{
			  "issues":[
			    {"id":"10001","key":"DEMO-1","fields":{"updated":"2024-06-01T12:00:00.000+0000"}},
			    {"id":"10002","key":"DEMO-2","fields":{"updated":"2024-06-02T12:00:00.000+0000"}}
			  ]
			}`))
		case r.Method == http.MethodGet && strings.Contains(path, "/issue/DEMO-1") && !strings.Contains(path, "/comment"):
			writeIssue(w, "10001", "DEMO-1", "First", "2024-06-01T12:00:00.000+0000", adfDesc)
		case r.Method == http.MethodGet && strings.Contains(path, "/issue/DEMO-2") && !strings.Contains(path, "/comment"):
			writeIssue(w, "10002", "DEMO-2", "Second", "2024-06-02T12:00:00.000+0000", adfDesc)
		case strings.Contains(path, "/comment"):
			payload := map[string]any{
				"startAt":    0,
				"maxResults": 100,
				"total":      1,
				"comments": []map[string]any{{
					"id":           "1",
					"author":       map[string]any{"displayName": "Alice"},
					"created":      "2024-06-01T13:00:00.000+0000",
					"body":         json.RawMessage(adfComment),
					"renderedBody": "<p>A comment</p>",
				}},
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			t.Errorf("unexpected %s %s", r.Method, path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", Path: filepath.Join(dir, "t.db")},
		Storage:  config.StorageConfig{BasePath: filepath.Join(dir, "saved")},
		Session:  config.SessionConfig{EncryptionKey: testKey, FilePath: filepath.Join(dir, "session.enc")},
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

	sessStore := session.NewStore(cfg.Session.FilePath, logging.Sugar{})
	if err := sessStore.Upsert(&session.Session{
		SiteURL:    srv.URL,
		Cookies:    []session.Cookie{{Name: "JSESSIONID", Value: "x"}},
		CapturedAt: time.Now(),
		Flavor:     session.FlavorCloud,
	}, testKey); err != nil {
		t.Fatal(err)
	}

	writer := storage.NewWriter(cfg.Storage.BasePath, logging.New("storage", log))
	s := scraper.New(db, sessStore, testKey, writer, logging.New("scraper", log))

	stats, err := s.CrawlProject(context.Background(), "DEMO")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Discovered != 2 || stats.Fetched != 2 {
		t.Fatalf("stats = %+v", stats)
	}
	if searchCalls < 1 {
		t.Fatal("expected search call")
	}

	iss, err := db.GetIssueByKey(context.Background(), "DEMO-1")
	if err != nil {
		t.Fatal(err)
	}
	if iss.Summary != "First" {
		t.Fatalf("summary = %q", iss.Summary)
	}
	if !strings.Contains(iss.Content, "Hello ticket") {
		t.Fatalf("content missing description: %q", iss.Content)
	}
	if !strings.Contains(iss.Content, "A comment") {
		t.Fatalf("content missing comment: %q", iss.Content)
	}
	if iss.FieldsJSON == "" {
		t.Fatal("expected fields_json")
	}
	if iss.BodyFormat != "adf" {
		t.Fatalf("body_format = %q", iss.BodyFormat)
	}

	stats2, err := s.CrawlProject(context.Background(), "DEMO")
	if err != nil {
		t.Fatal(err)
	}
	if stats2.Skipped != 2 || stats2.Fetched != 0 {
		t.Fatalf("incremental stats = %+v", stats2)
	}

	s.SetForce(true)
	stats3, err := s.CrawlProject(context.Background(), "DEMO")
	if err != nil {
		t.Fatal(err)
	}
	if stats3.Fetched != 2 {
		t.Fatalf("force stats = %+v", stats3)
	}
}

func writeIssue(w http.ResponseWriter, id, key, summary, updated string, desc json.RawMessage) {
	payload := map[string]any{
		"id":  id,
		"key": key,
		"fields": map[string]any{
			"summary":     summary,
			"updated":     updated,
			"description": json.RawMessage(desc),
			"attachment":  []any{},
		},
		"renderedFields": map[string]any{
			"description": "<p>Hello ticket</p>",
		},
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func TestCrawl_attachments(t *testing.T) {
	adfDesc := json.RawMessage(`{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"With files"}]}]}`)
	var okHits, failHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		origin := "http://" + r.Host
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/search/jql"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issues":[{"id":"1","key":"ATT-1","fields":{"updated":"2024-08-01T10:00:00.000+0000"}}]}`))
		case strings.Contains(path, "/issue/ATT-1") && !strings.Contains(path, "/comment"):
			w.Header().Set("Content-Type", "application/json")
			payload := map[string]any{
				"id":  "1",
				"key": "ATT-1",
				"fields": map[string]any{
					"summary":     "Has attachment",
					"updated":     "2024-08-01T10:00:00.000+0000",
					"description": json.RawMessage(adfDesc),
					"attachment": []map[string]any{
						{"id": 55, "filename": "notes.txt", "mimeType": "text/plain",
							"content": origin + "/rest/api/3/attachment/content/55", "size": 5},
						{"id": "66", "filename": "missing.bin", "mimeType": "application/octet-stream",
							"content": origin + "/rest/api/3/attachment/content/66", "size": 1},
					},
				},
				"renderedFields": map[string]any{"description": "<p>With files</p>"},
			}
			_ = json.NewEncoder(w).Encode(payload)
		case strings.Contains(path, "/comment"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"startAt":0,"maxResults":100,"total":0,"comments":[]}`))
		case strings.HasSuffix(path, "/attachment/content/55"):
			okHits++
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("hello"))
		case strings.HasSuffix(path, "/attachment/content/66"):
			failHits++
			http.Error(w, "gone", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected %s %s", r.Method, path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", Path: filepath.Join(dir, "t.db")},
		Storage:  config.StorageConfig{BasePath: filepath.Join(dir, "saved")},
		Session:  config.SessionConfig{EncryptionKey: testKey, FilePath: filepath.Join(dir, "session.enc")},
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

	sessStore := session.NewStore(cfg.Session.FilePath, logging.Sugar{})
	if err := sessStore.Upsert(&session.Session{
		SiteURL:    srv.URL,
		Cookies:    []session.Cookie{{Name: "JSESSIONID", Value: "z"}},
		CapturedAt: time.Now(),
		Flavor:     session.FlavorCloud,
	}, testKey); err != nil {
		t.Fatal(err)
	}

	s := scraper.New(db, sessStore, testKey, storage.NewWriter(cfg.Storage.BasePath, logging.New("s", log)), logging.New("scraper", log))
	if _, err := s.CrawlProject(context.Background(), "ATT"); err != nil {
		t.Fatal(err)
	}

	saved := filepath.Join(cfg.Storage.BasePath, "ATT", "ATT-1", "assets", "attachments", "55_notes.txt")
	b, err := os.ReadFile(saved)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("content=%q", b)
	}
	iss, err := db.GetIssueByKey(context.Background(), "ATT-1")
	if err != nil {
		t.Fatal(err)
	}
	if iss.Summary != "Has attachment" {
		t.Fatalf("summary=%q", iss.Summary)
	}
	metaBytes, err := os.ReadFile(filepath.Join(cfg.Storage.BasePath, "ATT", "ATT-1", "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(metaBytes), `"local_path": "assets/attachments/55_notes.txt"`) {
		t.Fatalf("metadata missing local_path: %s", metaBytes)
	}
	if okHits != 1 {
		t.Fatalf("okHits=%d want 1", okHits)
	}
	if failHits == 0 {
		t.Fatal("expected failed attachment attempts")
	}
	// Issue still saved despite attachment 66 failing
	_ = failHits

	if err := s.CrawlIssue(context.Background(), "ATT-1"); err != nil {
		t.Fatal(err)
	}
	if okHits != 1 {
		t.Fatalf("expected skip existing ok attachment; okHits=%d", okHits)
	}

	s.SetForce(true)
	if err := s.CrawlIssue(context.Background(), "ATT-1"); err != nil {
		t.Fatal(err)
	}
	if okHits != 2 {
		t.Fatalf("expected force re-download; okHits=%d", okHits)
	}
}

func TestCrawlProject_serverSearch(t *testing.T) {
	adfDesc := json.RawMessage(`{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"DC body"}]}]}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/search"):
			_, _ = w.Write([]byte(`{
			  "startAt":0,"maxResults":50,"total":1,
			  "issues":[{"id":"9","key":"DC-1","fields":{"updated":"2024-07-01T10:00:00.000+0000"}}]
			}`))
		case strings.Contains(path, "/issue/DC-1") && !strings.Contains(path, "/comment"):
			writeIssue(w, "9", "DC-1", "DC Issue", "2024-07-01T10:00:00.000+0000", adfDesc)
		case strings.Contains(path, "/comment"):
			_, _ = w.Write([]byte(`{"startAt":0,"maxResults":100,"total":0,"comments":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", Path: filepath.Join(dir, "t.db")},
		Storage:  config.StorageConfig{BasePath: filepath.Join(dir, "saved")},
		Session:  config.SessionConfig{EncryptionKey: testKey, FilePath: filepath.Join(dir, "session.enc")},
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

	sessStore := session.NewStore(cfg.Session.FilePath, logging.Sugar{})
	if err := sessStore.Upsert(&session.Session{
		SiteURL:    srv.URL,
		Cookies:    []session.Cookie{{Name: "JSESSIONID", Value: "y"}},
		CapturedAt: time.Now(),
		Flavor:     session.FlavorServer,
	}, testKey); err != nil {
		t.Fatal(err)
	}

	s := scraper.New(db, sessStore, testKey, storage.NewWriter(cfg.Storage.BasePath, logging.New("s", log)), logging.New("scraper", log))
	stats, err := s.CrawlProject(context.Background(), "DC")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Fetched != 1 {
		t.Fatalf("stats=%+v", stats)
	}
}
