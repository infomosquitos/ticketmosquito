package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the TicketMosquito YAML configuration.
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Storage  StorageConfig  `yaml:"storage"`
	Session  SessionConfig  `yaml:"session"`
	Browser  BrowserConfig  `yaml:"browser"`
	MCP      MCPConfig      `yaml:"mcp"`
}

// DatabaseConfig configures the embedded SQLite database.
type DatabaseConfig struct {
	Driver string `yaml:"driver"` // sqlite
	Path   string `yaml:"path"`   // file name or absolute path
}

// DriverName returns the configured driver, defaulting to sqlite.
func (c DatabaseConfig) DriverName() string {
	if c.Driver == "" {
		return "sqlite"
	}
	return c.Driver
}

// StorageConfig holds on-disk saved issue paths.
type StorageConfig struct {
	BasePath string `yaml:"base_path"`
}

// SessionConfig holds encrypted session blob settings.
type SessionConfig struct {
	EncryptionKey string `yaml:"encryption_key"`
	FilePath      string `yaml:"file_path"`
}

// BrowserConfig optionally overrides the Chromium path for rod fallback.
type BrowserConfig struct {
	Path string `yaml:"path"`
}

// MCPConfig configures the local HTTP/MCP listener.
type MCPConfig struct {
	Port              int    `yaml:"port"`
	Host              string `yaml:"host"`
	ExposeInternalIDs bool   `yaml:"expose_internal_ids"`
}

// Load reads and unmarshals a YAML config file.
func Load(path string) (*Config, error) {
	expanded := os.ExpandEnv(path)
	if expanded != path {
		if p, err := filepath.Abs(expanded); err == nil {
			path = p
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Storage.BasePath == "" {
		cfg.Storage.BasePath = "./saved"
	}
	if cfg.MCP.Port == 0 {
		cfg.MCP.Port = 11043
	}
	if cfg.MCP.Host == "" {
		cfg.MCP.Host = "127.0.0.1"
	}

	return &cfg, nil
}
