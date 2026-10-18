package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/store"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
	"go.uber.org/zap"

	_ "modernc.org/sqlite"
)

var _ store.Store = (*DB)(nil)

// DB is the SQLite catalog implementation.
type DB struct {
	sql *sql.DB
	log logging.Sugar
}

// New opens (and creates) the SQLite database file.
func New(cfg *config.Config, log *zap.Logger) (*DB, error) {
	path, err := DBFilePath(cfg)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	log.Info("connected to database", zap.String("path", path), zap.String("driver", "sqlite"))

	return &DB{
		sql: db,
		log: logging.New("sqlite", log),
	}, nil
}

// Close closes the database.
func (d *DB) Close() {
	d.sql.Close()
}

// Ping checks connectivity.
func (d *DB) Ping(ctx context.Context) error {
	return d.sql.PingContext(ctx)
}
