package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/datastore"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
	"go.uber.org/zap"
)

func TestMigrateUp_embedded(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(dir, "embed.db"),
		},
	}
	log := zap.NewNop()
	if err := datastore.MigrateUp(cfg, "", logging.New("test", log)); err != nil {
		t.Fatalf("MigrateUp embedded: %v", err)
	}
	db, err := datastore.Open(cfg, log)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(t.Context()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
