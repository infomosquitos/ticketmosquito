package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// PerProjectCronConfig holds per-project interval overrides stored on disk.
type PerProjectCronConfig struct {
	ProjectKey        string `json:"project_key"`
	ProjectURL        string `json:"project_url"`
	FullCrawl         bool   `json:"full_crawl_enabled"`
	FullCrawlInterval string `json:"full_crawl_interval"`
	IncrCrawl         bool   `json:"incr_crawl_enabled"`
	IncrCrawlInterval string `json:"incr_crawl_interval"`
	Detection         string `json:"detection"`
}

// Manager persists per-project cron overrides.
type Manager struct {
	filePath string
	mu       sync.RWMutex
	log      logging.Sugar
	configs  []PerProjectCronConfig
}

// NewManager loads overrides from filePath (creates empty if missing).
func NewManager(filePath string, log logging.Sugar) *Manager {
	m := &Manager{filePath: filePath, configs: []PerProjectCronConfig{}, log: log}
	m.load()
	return m
}

func (m *Manager) load() {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		m.configs = []PerProjectCronConfig{}
		return
	}
	var configs []PerProjectCronConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		m.log.Warnw("failed to parse cron config", "error", err)
		m.configs = []PerProjectCronConfig{}
		return
	}
	m.configs = configs
}

func (m *Manager) save() error {
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	data, err := json.MarshalIndent(m.configs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0o644)
}

// GetOverride returns a copy of the override or nil.
func (m *Manager) GetOverride(projectKey string) *PerProjectCronConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.configs {
		if c.ProjectKey == projectKey {
			cp := c
			return &cp
		}
	}
	return nil
}

// Upsert saves or updates a per-project config.
func (m *Manager) Upsert(cfg PerProjectCronConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, c := range m.configs {
		if c.ProjectKey == cfg.ProjectKey {
			m.configs[i] = cfg
			return m.save()
		}
	}
	m.configs = append(m.configs, cfg)
	return m.save()
}

// Delete removes a per-project config.
func (m *Manager) Delete(projectKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := make([]PerProjectCronConfig, 0, len(m.configs))
	for _, c := range m.configs {
		if c.ProjectKey != projectKey {
			kept = append(kept, c)
		}
	}
	m.configs = kept
	return m.save()
}

// List returns all overrides.
func (m *Manager) List() []PerProjectCronConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]PerProjectCronConfig, len(m.configs))
	copy(out, m.configs)
	return out
}
