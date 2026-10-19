package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/infomosquitos/ticketmosquito/internal/api"
	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/datastore"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
	"go.uber.org/zap"
)

func TestProjectsHandler_AddListDelete(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", Path: filepath.Join(dir, "p.db")},
	}
	log := zap.NewNop()
	if err := datastore.MigrateUp(cfg, "", logging.New("test", log)); err != nil {
		t.Fatal(err)
	}
	db, err := datastore.Open(cfg, log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)

	h := api.NewProjectsHandler(db, logging.New("test", log))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects", h.List)
	mux.HandleFunc("POST /api/projects", h.Add)
	mux.HandleFunc("DELETE /api/projects/{key}", h.Delete)

	body := `{"url":"https://jira.example.com/browse/DEMO-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add status %d body %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d", rec.Code)
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0]["project_key"] != "DEMO" {
		t.Fatalf("list = %#v", list)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/projects/DEMO", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status %d body %s", rec.Code, rec.Body.String())
	}
}
