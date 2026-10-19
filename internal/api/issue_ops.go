package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/jira"
	"github.com/infomosquitos/ticketmosquito/internal/scraper"
	"github.com/infomosquitos/ticketmosquito/internal/session"
	"github.com/infomosquitos/ticketmosquito/internal/store"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// IssueOpsHandler serves compare + refresh for a single issue.
type IssueOpsHandler struct {
	db   store.Store
	sess *session.Store
	scr  *scraper.Scraper
	cfg  *config.Config
	log  logging.Sugar
}

// NewIssueOpsHandler creates issue compare/refresh handlers.
func NewIssueOpsHandler(db store.Store, sess *session.Store, scr *scraper.Scraper, cfg *config.Config, log logging.Sugar) *IssueOpsHandler {
	return &IssueOpsHandler{db: db, sess: sess, scr: scr, cfg: cfg, log: log}
}

// Compare handles GET /api/issues/{key}/compare
func (h *IssueOpsHandler) Compare(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "issue key is required"})
		return
	}
	encKey := h.cfg.Session.EncryptionKey
	if encKey == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption key not configured"})
		return
	}

	stored, err := h.db.GetIssueByKey(r.Context(), key)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		stored = nil
	}

	sess, siteURL, err := h.resolveSession(r.Context(), key, stored)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	client, err := scraper.ClientFromSession(sess, h.log)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	detail, err := client.GetIssue(r.Context(), key)
	if err != nil {
		if jira.IsUnauthorized(err) {
			_ = h.sess.ClearValidationForURL(encKey, siteURL)
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to fetch live issue: " + err.Error()})
		return
	}

	projectKey, _, _ := jira.ParseProjectRef(key)
	out := map[string]any{
		"issue_key":   key,
		"project_key": projectKey,
		"live": map[string]any{
			"updated": formatOptTimePtr(&detail.Updated),
			"summary": detail.Summary,
		},
	}
	if stored != nil {
		out["stored"] = map[string]any{
			"updated":     formatOptTimePtr(stored.JiraUpdated),
			"summary":     stored.Summary,
			"body_format": stored.BodyFormat,
			"updated_at":  stored.UpdatedAt.UTC().Format(time.RFC3339),
		}
	} else {
		out["stored"] = nil
	}
	writeJSON(w, http.StatusOK, out)
}

// Refresh handles POST /api/issues/{key}/refresh
func (h *IssueOpsHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "issue key is required"})
		return
	}
	if err := h.scr.CrawlIssue(r.Context(), key); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	iss, err := h.db.GetIssueByKey(r.Context(), key)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"issue_key": key, "message": "refreshed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"issue_key":   iss.Key,
		"project_key": iss.ProjectKey,
		"summary":     iss.Summary,
		"updated":     formatOptTimePtr(iss.JiraUpdated),
		"body_format": iss.BodyFormat,
	})
}

func (h *IssueOpsHandler) resolveSession(ctx context.Context, issueKey string, stored *store.Issue) (*session.Session, string, error) {
	encKey := h.cfg.Session.EncryptionKey
	ref := issueKey
	if stored != nil && stored.ProjectKey != "" {
		if p, err := h.db.GetProjectByKey(ctx, stored.ProjectKey); err == nil && p.URL != "" {
			ref = p.URL
		}
	}
	sess, err := h.sess.GetForURL(encKey, ref)
	if err != nil {
		sess, err = h.sess.Load(encKey)
		if err != nil {
			return nil, "", err
		}
	}
	return sess, sess.SiteURL, nil
}

func formatOptTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
