package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/search"
	"github.com/infomosquitos/ticketmosquito/internal/store"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// CatalogHandler serves search and issue read HTTP endpoints.
type CatalogHandler struct {
	db  store.Store
	cfg *config.Config
	log logging.Sugar
}

// NewCatalogHandler creates a catalog HTTP handler.
func NewCatalogHandler(db store.Store, cfg *config.Config, log logging.Sugar) *CatalogHandler {
	return &CatalogHandler{db: db, cfg: cfg, log: log}
}

// Search handles GET /api/search?q=&project=&limit=
func (h *CatalogHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter 'q' is required"})
		return
	}
	projectKey := r.URL.Query().Get("project")
	if projectKey == "" {
		projectKey = r.URL.Query().Get("project_key")
	}
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	results, err := h.db.SearchIssues(r.Context(), query, projectKey, limit)
	if err != nil {
		h.log.Errorw("search failed", "query", query, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search failed: " + err.Error()})
		return
	}
	hits := search.ToSearchHits(results, h.cfg.MCP.ExposeInternalIDs)
	writeJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"count":   len(hits),
		"results": hits,
	})
}

// GetIssue handles GET /api/issues/{key}
func (h *CatalogHandler) GetIssue(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/api/issues/")
	key = strings.Trim(key, "/")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "issue key is required"})
		return
	}

	iss, err := h.db.GetIssueByKey(r.Context(), key)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "issue not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, search.ToIssueDetail(iss, h.cfg.MCP.ExposeInternalIDs))
}

// ListProjects handles GET /api/projects
func (h *CatalogHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.db.ListProjects(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count":    len(projects),
		"projects": search.ToProjectSummaries(projects),
	})
}

// Stats handles GET /api/search/stats
func (h *CatalogHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetCatalogStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// Reindex handles POST /api/search/reindex
func (h *CatalogHandler) Reindex(w http.ResponseWriter, r *http.Request) {
	if err := h.db.RebuildIssuesFTS(r.Context()); err != nil {
		h.log.Errorw("reindex failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "reindex failed: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "issues FTS reindexed"})
}
