package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/session"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// Handler serves session HTTP endpoints.
type Handler struct {
	store *session.Store
	cfg   *config.Config
	log   logging.Sugar
}

// New creates an API handler.
func New(store *session.Store, cfg *config.Config, log logging.Sugar) *Handler {
	return &Handler{store: store, cfg: cfg, log: log}
}

type createSessionRequest struct {
	JiraURL string           `json:"jira_url"`
	Cookies []session.Cookie `json:"cookies"`
}

// CreateSession stores cookies for a Jira host.
func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}
	if req.JiraURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "jira_url is required"})
		return
	}
	if len(req.Cookies) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one cookie is required"})
		return
	}
	if session.HostnameKey(req.JiraURL) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "jira_url must include a hostname"})
		return
	}

	sess := &session.Session{
		SiteURL:    req.JiraURL,
		Cookies:    req.Cookies,
		CapturedAt: time.Now(),
	}
	sess.SetLogger(h.store.GetLogger())

	encKey := h.cfg.Session.EncryptionKey
	if encKey == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption key not configured"})
		return
	}
	if err := h.store.Upsert(sess, encKey); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save session: " + err.Error()})
		return
	}

	h.log.Infow("session upserted",
		"url", req.JiraURL,
		"host", session.HostnameKey(req.JiraURL),
		"cookie_count", len(req.Cookies))

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":      "session saved",
		"jira_url":     req.JiraURL,
		"cookie_count": len(req.Cookies),
	})
}

// DeleteSession clears all stored sessions.
func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	if !h.store.HasSession() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no session stored"})
		return
	}
	if err := h.store.Delete(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete session: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "session deleted"})
}

// SessionStatus reports stored session validity for a URL (or the only session).
func (h *Handler) SessionStatus(w http.ResponseWriter, r *http.Request) {
	if !h.store.HasSession() {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid": false, "message": "no session stored", "exists": false,
		})
		return
	}

	encKey := h.cfg.Session.EncryptionKey
	siteURL := r.URL.Query().Get("url")

	var sess *session.Session
	var err error
	if siteURL != "" {
		sess, err = h.store.GetForURL(encKey, siteURL)
	} else {
		sess, err = h.store.Load(encKey)
	}
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid": false, "message": err.Error(), "exists": h.store.HasSession(),
		})
		return
	}

	host := session.HostnameKey(sess.SiteURL)
	ttl := session.ValidateTTL
	status := map[string]interface{}{
		"exists":  true,
		"valid":   false,
		"message": "session requires validation",
		"host":    host,
		"flavor":  sess.Flavor,
	}

	if sess.ValidatedAt != nil {
		since := time.Since(*sess.ValidatedAt)
		status["validated_at"] = sess.ValidatedAt.UTC().Format(time.RFC3339)
		if since > ttl {
			status["message"] = "session validation expired"
		} else {
			status["valid"] = true
			status["message"] = "session is valid"
		}
	}
	if sess.IsExpired(ttl) && sess.ValidatedAt == nil {
		status["message"] = "session is stale"
	}

	writeJSON(w, http.StatusOK, status)
}

// ValidateSession probes Jira /myself and persists flavor + ValidatedAt.
func (h *Handler) ValidateSession(w http.ResponseWriter, r *http.Request) {
	encKey := h.cfg.Session.EncryptionKey

	var req struct {
		JiraURL string `json:"jira_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	var sess *session.Session
	var err error
	if req.JiraURL != "" {
		sess, err = h.store.GetForURL(encKey, req.JiraURL)
	} else {
		sess, err = h.store.Load(encKey)
	}
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"valid": false, "message": err.Error()})
		return
	}

	sess.SetLogger(h.log)
	result, err := sess.ValidateWithJira(req.JiraURL, 10)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "validation failed: " + err.Error()})
		return
	}

	if result.Valid {
		_ = h.store.Upsert(sess, encKey)
	} else {
		sess.ClearValidation()
		_ = h.store.Upsert(sess, encKey)
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
