package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/infomosquitos/ticketmosquito/internal/api"
	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/datastore"
	"github.com/infomosquitos/ticketmosquito/internal/mcp"
	"github.com/infomosquitos/ticketmosquito/internal/paths"
	"github.com/infomosquitos/ticketmosquito/internal/session"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
	"go.uber.org/zap"
)

// Run starts the HTTP API, catalog search, and MCP Streamable HTTP endpoint.
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
	apiHandler := api.New(sessionStore, cfg, logging.New("api", log))
	catalog := api.NewCatalogHandler(db, cfg, logging.New("catalog", log))
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
		_, _ = w.Write([]byte("ticketmosquito — session, search, MCP ready\n"))
	})
	mux.HandleFunc("POST /api/session", apiHandler.CreateSession)
	mux.HandleFunc("DELETE /api/session", apiHandler.DeleteSession)
	mux.HandleFunc("GET /api/session/status", apiHandler.SessionStatus)
	mux.HandleFunc("POST /api/session/validate", apiHandler.ValidateSession)

	mux.HandleFunc("GET /api/search", catalog.Search)
	mux.HandleFunc("GET /api/search/stats", catalog.Stats)
	mux.HandleFunc("POST /api/search/reindex", catalog.Reindex)
	mux.HandleFunc("GET /api/projects", catalog.ListProjects)
	mux.HandleFunc("GET /api/issues/", catalog.GetIssue)

	// Method-specific patterns required: bare "/mcp" conflicts with "GET /" on Go 1.22+ ServeMux.
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
