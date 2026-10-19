package api

import (
	"encoding/json"
	"net/http"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/cron"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// CronAPIHandler serves cron config + scheduler control.
type CronAPIHandler struct {
	cfg     *config.Config
	log     logging.Sugar
	manager *cron.Manager
	sched   *cron.Scheduler
}

// NewCronAPIHandler creates cron HTTP handlers.
func NewCronAPIHandler(cfg *config.Config, manager *cron.Manager, sched *cron.Scheduler, log logging.Sugar) *CronAPIHandler {
	return &CronAPIHandler{cfg: cfg, log: log, manager: manager, sched: sched}
}

// List handles GET /api/cron
func (h *CronAPIHandler) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.sched.ListJobs())
}

// StartNow handles POST /api/cron/start
func (h *CronAPIHandler) StartNow(w http.ResponseWriter, r *http.Request) {
	h.sched.StartNow()
	writeJSON(w, http.StatusOK, map[string]string{"status": "jobs triggered"})
}

// ConfigGet handles GET /api/cron/config
func (h *CronAPIHandler) ConfigGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"yaml_full_crawl":       cronJobYAMLView(h.cfg.Cron.FullCrawl),
		"yaml_incremental":      cronJobYAMLView(h.cfg.Cron.Incremental),
		"per_project_overrides": h.manager.List(),
	})
}

func cronJobYAMLView(c *config.CronJobConfig) map[string]any {
	if c == nil {
		return map[string]any{"enabled": false, "interval": "", "projects": []string{}, "detection": "", "max_duration": ""}
	}
	projects := c.Projects
	if projects == nil {
		projects = []string{}
	}
	return map[string]any{
		"enabled":      c.Enabled,
		"interval":     c.Interval,
		"projects":     projects,
		"detection":    c.Detection,
		"max_duration": c.MaxDuration,
	}
}

// ConfigReload handles POST /api/cron/reload
func (h *CronAPIHandler) ConfigReload(w http.ResponseWriter, r *http.Request) {
	if err := h.sched.Restart(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "scheduler restarted with new config"})
}

// ProjectCronGet handles GET /api/cron/project/{key}
func (h *CronAPIHandler) ProjectCronGet(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	ov := h.manager.GetOverride(key)
	if ov == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"project_key":        key,
			"full_crawl_enabled": false,
			"incr_crawl_enabled": false,
		})
		return
	}
	writeJSON(w, http.StatusOK, ov)
}

// ProjectCronUpdate handles POST /api/cron/project/{key}
func (h *CronAPIHandler) ProjectCronUpdate(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	ov := cron.PerProjectCronConfig{ProjectKey: key}
	if existing := h.manager.GetOverride(key); existing != nil {
		ov = *existing
	}
	if v, ok := req["project_url"].(string); ok {
		ov.ProjectURL = v
	}
	if v, ok := req["full_crawl_enabled"].(bool); ok {
		ov.FullCrawl = v
	}
	if v, ok := req["full_crawl_interval"].(string); ok {
		ov.FullCrawlInterval = v
	}
	if v, ok := req["incr_crawl_enabled"].(bool); ok {
		ov.IncrCrawl = v
	}
	if v, ok := req["incr_crawl_interval"].(string); ok {
		ov.IncrCrawlInterval = v
	}
	if v, ok := req["detection"].(string); ok {
		ov.Detection = v
	}
	if err := h.manager.Upsert(ov); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save config"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cron config updated"})
}

// ProjectCronDelete handles DELETE /api/cron/project/{key}
func (h *CronAPIHandler) ProjectCronDelete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if err := h.manager.Delete(key); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cron config deleted"})
}
