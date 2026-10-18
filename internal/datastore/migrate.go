package datastore

import (
	"fmt"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/store/sqlite"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// MigrateUp applies pending SQLite schema migrations.
// migrationsRoot is the directory that contains a sqlite/ subdirectory
// (or empty to use embedded migrations).
func MigrateUp(cfg *config.Config, migrationsRoot string, log logging.Sugar) error {
	if err := requireSQLite(cfg); err != nil {
		return err
	}
	return sqlite.MigrateUp(cfg, migrationsRoot, log)
}

// MigrateDown rolls back one SQLite migration step.
func MigrateDown(cfg *config.Config, migrationsRoot string, log logging.Sugar) error {
	if err := requireSQLite(cfg); err != nil {
		return err
	}
	return sqlite.MigrateDown(cfg, migrationsRoot, log)
}

func requireSQLite(cfg *config.Config) error {
	switch cfg.Database.DriverName() {
	case "sqlite":
		return nil
	default:
		return fmt.Errorf("unsupported database driver %q (only sqlite is supported)", cfg.Database.Driver)
	}
}
