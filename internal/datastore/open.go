package datastore

import (
	"fmt"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/store"
	"github.com/infomosquitos/ticketmosquito/internal/store/sqlite"
	"go.uber.org/zap"
)

// Open connects to the SQLite database.
func Open(cfg *config.Config, log *zap.Logger) (store.Store, error) {
	switch cfg.Database.DriverName() {
	case "sqlite":
		return sqlite.New(cfg, log)
	default:
		return nil, fmt.Errorf("unsupported database driver %q (only sqlite is supported)", cfg.Database.Driver)
	}
}
