package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/scraper"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// CrawlHandler serves async crawl job endpoints.
type CrawlHandler struct {
	manager *scraper.CrawlJobManager
	cfg     *config.Config
	log     logging.Sugar
}

// NewCrawlHandler creates a crawl handler.
func NewCrawlHandler(mgr *scraper.CrawlJobManager, cfg *config.Config, log logging.Sugar) *CrawlHandler {
	return &CrawlHandler{manager: mgr, cfg: cfg, log: log}
}

// Create handles POST /api/crawl — body { "project_url": "…", "force": bool }.
func (h *CrawlHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectURL string `json:"project_url"`
		Force      bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ProjectURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project_url is required"})
		return
	}
	job, err := h.manager.CreateJob(req.ProjectURL, req.Force)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	go func() { _ = h.manager.RunJob(context.Background(), job.ID) }()
	writeJSON(w, http.StatusAccepted, map[string]any{
		"message":     "crawl job created",
		"job_id":      job.ID,
		"status":      job.Status,
		"project_url": job.ProjectURL,
	})
}

// Status handles GET /api/crawl/status?id=
func (h *CrawlHandler) Status(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id query parameter is required"})
		return
	}
	job, err := h.manager.GetJob(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// List handles GET /api/crawl
func (h *CrawlHandler) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.manager.ListJobs())
}

// Cancel handles POST /api/crawl/cancel?id=
func (h *CrawlHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id query parameter is required"})
		return
	}
	if err := h.manager.CancelJob(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "job cancelled"})
}

// Cleanup handles POST /api/crawl/cleanup
func (h *CrawlHandler) Cleanup(w http.ResponseWriter, r *http.Request) {
	h.manager.Cleanup()
	writeJSON(w, http.StatusOK, map[string]string{"message": "completed jobs cleaned up"})
}
