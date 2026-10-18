package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/session"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
	"go.uber.org/zap"
)

func testHandler(t *testing.T) *Handler {
	t.Helper()
	cfg := &config.Config{
		Session: config.SessionConfig{
			EncryptionKey: "0123456789abcdef0123456789abcdef",
			FilePath:      filepath.Join(t.TempDir(), "session.enc"),
		},
	}
	store := session.NewStore(cfg.Session.FilePath, logging.New("test", zap.NewNop()))
	return New(store, cfg, logging.New("api", zap.NewNop()))
}

func TestCreateSession(t *testing.T) {
	h := testHandler(t)
	body := `{"jira_url":"https://acme.atlassian.net/browse/X-1","cookies":[{"name":"JSESSIONID","value":"abc","domain":"acme.atlassian.net","path":"/"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewReader([]byte(body)))
	rr := httptest.NewRecorder()
	h.CreateSession(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/session/status?url=https://acme.atlassian.net/browse/X-2", nil)
	statusRR := httptest.NewRecorder()
	h.SessionStatus(statusRR, statusReq)
	if statusRR.Code != http.StatusOK {
		t.Fatalf("status code = %d", statusRR.Code)
	}
	var st map[string]interface{}
	if err := json.Unmarshal(statusRR.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if st["exists"] != true {
		t.Fatalf("exists = %v", st["exists"])
	}
}

func TestCreateSession_requiresURL(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewReader([]byte(`{"cookies":[{"name":"a","value":"b"}]}`)))
	rr := httptest.NewRecorder()
	h.CreateSession(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestDeleteSession(t *testing.T) {
	h := testHandler(t)
	_ = h.store.Upsert(&session.Session{
		SiteURL:    "https://acme.atlassian.net/browse/X-1",
		Cookies:    []session.Cookie{{Name: "a", Value: "b"}},
		CapturedAt: time.Now(),
	}, h.cfg.Session.EncryptionKey)

	req := httptest.NewRequest(http.MethodDelete, "/api/session", nil)
	rr := httptest.NewRecorder()
	h.DeleteSession(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
}
