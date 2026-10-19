package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/infomosquitos/ticketmosquito/internal/api"
	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/cron"
	"github.com/infomosquitos/ticketmosquito/internal/datastore"
	"github.com/infomosquitos/ticketmosquito/internal/mcp"
	"github.com/infomosquitos/ticketmosquito/internal/paths"
	"github.com/infomosquitos/ticketmosquito/internal/scraper"
	"github.com/infomosquitos/ticketmosquito/internal/session"
	"github.com/infomosquitos/ticketmosquito/internal/storage"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
	"go.uber.org/zap"
)

// Run starts the HTTP API, catalog search, crawl/cron, and MCP Streamable HTTP endpoint.
func Run(ctx context.Context, cfg *config.Config, log *zap.Logger) error {
	sugar := logging.New("app", log)

	migrationsPath, err := paths.ResolveMigrations()
	if err != nil {
		return fmt.Errorf("resolve migrations: %w", err)
	}
	if err := datastore.MigrateUp(cfg, migrationsPath, sugar); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	db, err := datastore.Open(cfg, log)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	sessionStore := session.NewStore(cfg.Session.FilePath, logging.New("session", log))
	writer := storage.NewWriter(cfg.Storage.BasePath, logging.New("storage", log))
	scr := scraper.New(db, sessionStore, cfg.Session.EncryptionKey, writer, logging.New("scraper", log))
	jobMgr := scraper.NewJobManager(cfg, db, sessionStore, writer, logging.New("crawljobs", log))

	cronPath, err := paths.ResolveCronConfig()
	if err != nil {
		return fmt.Errorf("resolve cron config: %w", err)
	}
	cronMan := cron.NewManager(cronPath, logging.New("cron", log))
	cronSched := cron.NewScheduler(cfg, cronMan, jobMgr, logging.New("cronsched", log))
	if err := cronSched.Start(ctx); err != nil {
		sugar.Warnw("cron scheduler start failed", "error", err)
	}
	defer cronSched.Stop()

	apiHandler := api.New(sessionStore, cfg, logging.New("api", log))
	catalog := api.NewCatalogHandler(db, cfg, logging.New("catalog", log))
	projects := api.NewProjectsHandler(db, logging.New("projects", log))
	crawlH := api.NewCrawlHandler(jobMgr, cfg, logging.New("crawl", log))
	issueOps := api.NewIssueOpsHandler(db, sessionStore, scr, cfg, logging.New("issueops", log))
	cronAPI := api.NewCronAPIHandler(cfg, cronMan, cronSched, logging.New("cronapi", log))
	mcpServer := mcp.New(db, cfg, logging.New("mcp", log))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ticketmosquito — session, crawl, search, MCP ready\n"))
	})

	mux.HandleFunc("POST /api/session", apiHandler.CreateSession)
	mux.HandleFunc("DELETE /api/session", apiHandler.DeleteSession)
	mux.HandleFunc("GET /api/session/status", apiHandler.SessionStatus)
	mux.HandleFunc("POST /api/session/validate", apiHandler.ValidateSession)

	mux.HandleFunc("GET /api/search", catalog.Search)
	mux.HandleFunc("GET /api/search/stats", catalog.Stats)
	mux.HandleFunc("POST /api/search/reindex", catalog.Reindex)

	mux.HandleFunc("GET /api/projects", projects.List)
	mux.HandleFunc("POST /api/projects", projects.Add)
	mux.HandleFunc("GET /api/projects/{key}", projects.Get)
	mux.HandleFunc("DELETE /api/projects/{key}", projects.Delete)

	mux.HandleFunc("POST /api/crawl", crawlH.Create)
	mux.HandleFunc("GET /api/crawl", crawlH.List)
	mux.HandleFunc("GET /api/crawl/status", crawlH.Status)
	mux.HandleFunc("POST /api/crawl/cancel", crawlH.Cancel)
	mux.HandleFunc("POST /api/crawl/cleanup", crawlH.Cleanup)

	mux.HandleFunc("GET /api/issues/{key}/compare", issueOps.Compare)
	mux.HandleFunc("POST /api/issues/{key}/refresh", issueOps.Refresh)
	mux.HandleFunc("GET /api/issues/{key}", catalog.GetIssueByKey)

	mux.HandleFunc("GET /api/cron", cronAPI.List)
	mux.HandleFunc("POST /api/cron/start", cronAPI.StartNow)
	mux.HandleFunc("GET /api/cron/config", cronAPI.ConfigGet)
	mux.HandleFunc("POST /api/cron/reload", cronAPI.ConfigReload)
	mux.HandleFunc("GET /api/cron/project/{key}", cronAPI.ProjectCronGet)
	mux.HandleFunc("POST /api/cron/project/{key}", cronAPI.ProjectCronUpdate)
	mux.HandleFunc("DELETE /api/cron/project/{key}", cronAPI.ProjectCronDelete)

	mux.HandleFunc("GET /mcp", mcpServer.HandleRequest)
	mux.HandleFunc("POST /mcp", mcpServer.HandleRequest)

	handler := api.CORSMiddleware(mux, sugar)

	addr := fmt.Sprintf("%s:%d", cfg.MCP.Host, cfg.MCP.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		sugar.Infow("listening", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
