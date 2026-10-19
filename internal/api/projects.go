package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/infomosquitos/ticketmosquito/internal/jira"
	"github.com/infomosquitos/ticketmosquito/internal/store"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// ProjectsHandler serves project registry CRUD for the extension.
type ProjectsHandler struct {
	db  store.Store
	log logging.Sugar
}

// NewProjectsHandler creates a projects handler.
func NewProjectsHandler(db store.Store, log logging.Sugar) *ProjectsHandler {
	return &ProjectsHandler{db: db, log: log}
}

// List handles GET /api/projects — SM-compatible array of project rows.
func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	projects, err := h.db.ListProjects(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	out := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		crawled, err := h.db.CountIssuesByProjectKey(r.Context(), p.Key)
		if err != nil {
			crawled = 0
		}
		row := map[string]any{
			"project_key":    p.Key,
			"project_name":   p.Name,
			"project_url":    p.URL,
			"issues_crawled": crawled,
			"issues_total":   p.IssuesTotal,
			"created_at":     p.CreatedAt.UTC().Format(time.RFC3339),
		}
		if p.LastCrawled != nil {
			row["last_crawled"] = p.LastCrawled.UTC().Format(time.RFC3339)
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, out)
}

// Add handles POST /api/projects — body { "url": "…" }.
func (h *ProjectsHandler) Add(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
		return
	}
	key, _, err := jira.ParseProjectRef(req.URL)
	if err != nil || key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid jira URL or project key"})
		return
	}
	projectURL := req.URL
	if !containsScheme(req.URL) {
		projectURL = key
	}
	id, err := h.db.UpsertProject(r.Context(), key, key, projectURL)
	if err != nil {
		h.log.Errorw("add project failed", "key", key, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create project"})
		return
	}
	h.log.Infow("project added", "id", id, "key", key)
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":     "project added",
		"project_key": key,
		"project_url": projectURL,
	})
}

// Get handles GET /api/projects/{key}.
func (h *ProjectsHandler) Get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project key is required"})
		return
	}
	p, err := h.db.GetProjectByKey(r.Context(), key)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	crawled, _ := h.db.CountIssuesByProjectKey(r.Context(), key)
	row := map[string]any{
		"project_key":    p.Key,
		"project_name":   p.Name,
		"project_url":    p.URL,
		"issues_crawled": crawled,
		"issues_total":   p.IssuesTotal,
	}
	if p.LastCrawled != nil {
		row["last_crawled"] = p.LastCrawled.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, row)
}

// Delete handles DELETE /api/projects/{key}.
func (h *ProjectsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project key is required"})
		return
	}
	if err := h.db.DeleteProject(r.Context(), key); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.log.Infow("project deleted", "project_key", key)
	writeJSON(w, http.StatusOK, map[string]string{"message": "project deleted"})
}

func containsScheme(s string) bool {
	return len(s) > 8 && (s[:8] == "https://" || s[:7] == "http://")
}
