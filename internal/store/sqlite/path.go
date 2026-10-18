package sqlite

import (
	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/paths"
)

// DBFilePath resolves the SQLite database file path from config.
func DBFilePath(cfg *config.Config) (string, error) {
	return paths.ResolveDB(cfg)
}
