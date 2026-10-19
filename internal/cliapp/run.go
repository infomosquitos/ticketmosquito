package cliapp

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/infomosquitos/ticketmosquito/internal/app"
	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/datastore"
	"github.com/infomosquitos/ticketmosquito/internal/paths"
	"github.com/infomosquitos/ticketmosquito/internal/scraper"
	"github.com/infomosquitos/ticketmosquito/internal/search"
	"github.com/infomosquitos/ticketmosquito/internal/session"
	"github.com/infomosquitos/ticketmosquito/internal/storage"
	"github.com/infomosquitos/ticketmosquito/internal/store"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
	"go.uber.org/zap"
)

// Version is stamped at link time via -ldflags.
var Version = "dev"

// Run is the CLI entrypoint. args should include the program name (os.Args).
func Run(args []string) int {
	if len(args) < 2 {
		printUsage()
		return 1
	}

	if args[1] == "version" || args[1] == "--version" {
		fmt.Println(Version)
		return 0
	}

	log, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		return 1
	}
	defer func() { _ = log.Sync() }()

	cmd := args[1]
	if cmd == "init" {
		return runInit(args[2:], log)
	}

	rest := applyDataDirFlag(args[2:])

	cfgPath, err := paths.ResolveConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve config path: %v\n", err)
		return 1
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		fmt.Fprintln(os.Stderr, "hint: run `ticketmosquito init` first")
		return 1
	}
	if err := paths.NormalizeConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to normalize config paths: %v\n", err)
		return 1
	}

	switch cmd {
	case "serve":
		return runServe(cfg, log)
	case "crawl":
		return runCrawl(cfg, rest, log)
	case "crawl-issue":
		return runCrawlIssue(cfg, rest, log)
	case "search":
		return runSearch(cfg, rest, log)
	case "get-issue":
		return runGetIssue(cfg, rest, log)
	case "reindex":
		return runReindex(cfg, log)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		return 1
	}
}

// applyDataDirFlag peels --data-dir from args and sets paths.SetDataDir.
func applyDataDirFlag(args []string) []string {
	out := make([]string, 0, len(args))
	var dataDir string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--data-dir" && i+1 < len(args):
			dataDir = args[i+1]
			i++
		case len(a) > 11 && a[:11] == "--data-dir=":
			dataDir = a[11:]
		default:
			out = append(out, a)
		}
	}
	if dataDir != "" {
		paths.SetDataDir(dataDir)
	}
	return out
}

func runInit(args []string, log *zap.Logger) int {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dataDir := fs.String("data-dir", "", "data directory (default ~/.ticketmosquito)")
	encKey := fs.String("encryption-key", "", "session encryption key (auto-generated if omitted)")
	force := fs.Bool("force-config", false, "overwrite existing config.yaml")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	result, err := paths.InitWorkspace(paths.InitOptions{
		DataDir:       *dataDir,
		EncryptionKey: *encKey,
		ForceConfig:   *force,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "init failed: %v\n", err)
		return 1
	}

	cfg, err := config.Load(result.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		return 1
	}
	if err := paths.NormalizeConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to normalize config paths: %v\n", err)
		return 1
	}

	migrationsPath, err := paths.ResolveMigrations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve migrations path: %v\n", err)
		return 1
	}

	sugar := logging.New("cli", log)
	if err := datastore.MigrateUp(cfg, migrationsPath, sugar); err != nil {
		sugar.Errorw("migration failed", "error", err)
		return 1
	}

	fmt.Printf("Initialized TicketMosquito data directory:\n")
	fmt.Printf("  data dir:    %s\n", result.DataDir)
	fmt.Printf("  config:      %s\n", result.ConfigPath)
	fmt.Printf("  session:     %s\n", result.SessionPath)
	if result.ConfigCreated {
		fmt.Printf("  encryption key written to config (keep this file private)\n")
	}
	fmt.Printf("Next: ticketmosquito serve\n")
	return 0
}

func runServe(cfg *config.Config, log *zap.Logger) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, log); err != nil {
		fmt.Fprintf(os.Stderr, "serve failed: %v\n", err)
		return 1
	}
	return 0
}

func runCrawl(cfg *config.Config, args []string, log *zap.Logger) int {
	fs := flag.NewFlagSet("crawl", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	force := fs.Bool("force", false, "re-fetch all issues (ignore updated skip)")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: ticketmosquito crawl [--force] <project-url-or-key>")
		return 1
	}
	s, cleanup, err := newScraper(cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "crawl setup failed: %v\n", err)
		return 1
	}
	defer cleanup()
	s.SetForce(*force)

	stats, err := s.CrawlProject(context.Background(), fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "crawl failed: %v\n", err)
		return 1
	}
	fmt.Printf("crawl done: discovered=%d fetched=%d skipped=%d failed=%d\n",
		stats.Discovered, stats.Fetched, stats.Skipped, stats.Failed)
	return 0
}

func runCrawlIssue(cfg *config.Config, args []string, log *zap.Logger) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: ticketmosquito crawl-issue <ISSUE-KEY>")
		return 1
	}
	s, cleanup, err := newScraper(cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "crawl setup failed: %v\n", err)
		return 1
	}
	defer cleanup()
	if err := s.CrawlIssue(context.Background(), args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "crawl-issue failed: %v\n", err)
		return 1
	}
	fmt.Printf("crawl-issue done: %s\n", args[0])
	return 0
}

func runSearch(cfg *config.Config, args []string, log *zap.Logger) int {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	project := fs.String("project", "", "filter by project key")
	limit := fs.Int("limit", 10, "max results")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, `usage: ticketmosquito search [--project KEY] [--limit N] "query"`)
		return 1
	}
	query := fs.Arg(0)
	db, cleanup, err := openDB(cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "search setup failed: %v\n", err)
		return 1
	}
	defer cleanup()

	results, err := db.SearchIssues(context.Background(), query, *project, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "search failed: %v\n", err)
		return 1
	}
	hits := search.ToSearchHits(results, cfg.MCP.ExposeInternalIDs)
	if len(hits) == 0 {
		fmt.Println("No results found")
		return 0
	}
	fmt.Printf("\n=== Search Results for %q ===\n", query)
	if *project != "" {
		fmt.Printf("Project: %s\n", *project)
	}
	fmt.Printf("Total: %d results\n\n", len(hits))
	for i, h := range hits {
		fmt.Printf("%d. %s — %s (project %s)\n", i+1, h.Key, h.Summary, h.ProjectKey)
		fmt.Printf("   Similarity: %.4f\n", h.Similarity)
		if h.Excerpt != "" {
			fmt.Printf("   Excerpt: %s\n", h.Excerpt)
		}
		fmt.Println()
	}
	return 0
}

func runGetIssue(cfg *config.Config, args []string, log *zap.Logger) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: ticketmosquito get-issue <ISSUE-KEY>")
		return 1
	}
	db, cleanup, err := openDB(cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get-issue setup failed: %v\n", err)
		return 1
	}
	defer cleanup()

	iss, err := db.GetIssueByKey(context.Background(), args[0])
	if errors.Is(err, store.ErrNotFound) {
		fmt.Fprintf(os.Stderr, "issue not found: %s\n", args[0])
		return 1
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "get-issue failed: %v\n", err)
		return 1
	}
	detail := search.ToIssueDetail(iss, cfg.MCP.ExposeInternalIDs)
	fmt.Printf("Key:     %s\n", detail.Key)
	fmt.Printf("Project: %s\n", detail.ProjectKey)
	fmt.Printf("Jira ID: %s\n", detail.JiraID)
	fmt.Printf("Summary: %s\n", detail.Summary)
	if detail.BodyFormat != "" {
		fmt.Printf("Format:  %s\n", detail.BodyFormat)
	}
	fmt.Println("---")
	fmt.Println(detail.Content)
	return 0
}

func runReindex(cfg *config.Config, log *zap.Logger) int {
	db, cleanup, err := openDB(cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reindex setup failed: %v\n", err)
		return 1
	}
	defer cleanup()
	if err := db.RebuildIssuesFTS(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "reindex failed: %v\n", err)
		return 1
	}
	fmt.Println("issues FTS reindexed")
	return 0
}

func openDB(cfg *config.Config, log *zap.Logger) (store.Store, func(), error) {
	migrationsPath, err := paths.ResolveMigrations()
	if err != nil {
		return nil, nil, err
	}
	sugar := logging.New("cli", log)
	if err := datastore.MigrateUp(cfg, migrationsPath, sugar); err != nil {
		return nil, nil, err
	}
	db, err := datastore.Open(cfg, log)
	if err != nil {
		return nil, nil, err
	}
	return db, db.Close, nil
}

func newScraper(cfg *config.Config, log *zap.Logger) (*scraper.Scraper, func(), error) {
	migrationsPath, err := paths.ResolveMigrations()
	if err != nil {
		return nil, nil, err
	}
	sugar := logging.New("cli", log)
	if err := datastore.MigrateUp(cfg, migrationsPath, sugar); err != nil {
		return nil, nil, err
	}
	db, err := datastore.Open(cfg, log)
	if err != nil {
		return nil, nil, err
	}
	sessStore := session.NewStore(cfg.Session.FilePath, logging.New("session", log))
	writer := storage.NewWriter(cfg.Storage.BasePath, logging.New("storage", log))
	s := scraper.New(db, sessStore, cfg.Session.EncryptionKey, writer, logging.New("scraper", log))
	return s, db.Close, nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `ticketmosquito — local Jira mirror

Usage:
  ticketmosquito init [--data-dir DIR] [--encryption-key KEY] [--force-config]
  ticketmosquito serve [--data-dir DIR]
  ticketmosquito crawl [--force] <project-url-or-key>
  ticketmosquito crawl-issue <ISSUE-KEY>
  ticketmosquito search [--project KEY] [--limit N] "query"
  ticketmosquito get-issue <ISSUE-KEY>
  ticketmosquito reindex
  ticketmosquito version

HTTP (with serve):
  GET/POST /api/projects           list / add projects
  POST /api/crawl                  start project crawl job
  GET  /api/issues/{KEY}           get issue; …/compare, …/refresh
  GET  /api/search?q=…&project=…   FTS search
  GET  /api/cron*                  scheduled crawls
  POST /mcp                        MCP Streamable HTTP (jira_* tools)

Environment:
  TICKETMOSQUITO_DATA_DIR   Data directory (default ~/.ticketmosquito)
  CONFIG_PATH               Override config.yaml path

`)
}

