package cron

import (
	"context"
	"sync"
	"time"

	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/scraper"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// JobInfo is a list-row for GET /api/cron.
type JobInfo struct {
	ID       string    `json:"id"`
	NextRun  time.Time `json:"next_run"`
	LastRun  time.Time `json:"last_run"`
	Disabled bool      `json:"disabled"`
}

// Scheduler runs per-project full/incremental crawls on intervals.
type Scheduler struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	entries map[string]*schedEntry
	log     logging.Sugar
	cfg     *config.Config
	man     *Manager
	jobs    *scraper.CrawlJobManager
}

type schedEntry struct {
	id       string
	next     time.Time
	last     time.Time
	disabled bool
	force    bool
	url      string
	interval time.Duration
}

// NewScheduler creates a scheduler that starts crawl jobs via jobs manager.
func NewScheduler(cfg *config.Config, man *Manager, jobs *scraper.CrawlJobManager, log logging.Sugar) *Scheduler {
	return &Scheduler{
		entries: make(map[string]*schedEntry),
		log:     log,
		cfg:     cfg,
		man:     man,
		jobs:    jobs,
	}
}

// Start rebuilds timers from YAML + per-project overrides.
func (s *Scheduler) Start(ctx context.Context) error {
	s.Stop()
	runCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.entries = make(map[string]*schedEntry)
	s.mu.Unlock()

	s.rebuildEntries()
	go s.loop(runCtx)
	s.log.Info("cron scheduler started")
	return nil
}

// Restart stops and starts again.
func (s *Scheduler) Restart(ctx context.Context) error {
	return s.Start(ctx)
}

// Stop cancels the scheduler loop.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// ListJobs returns scheduled entries.
func (s *Scheduler) ListJobs() []JobInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]JobInfo, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, JobInfo{ID: e.id, NextRun: e.next, LastRun: e.last, Disabled: e.disabled})
	}
	return out
}

// StartNow triggers all enabled entries immediately.
func (s *Scheduler) StartNow() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		if e.disabled {
			continue
		}
		e.next = time.Now()
	}
}

func (s *Scheduler) rebuildEntries() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = make(map[string]*schedEntry)
	now := time.Now()

	add := func(id, url string, interval time.Duration, force bool) {
		if url == "" || interval <= 0 {
			return
		}
		s.entries[id] = &schedEntry{
			id: id, url: url, interval: interval, force: force,
			next: now.Add(interval),
		}
	}

	if s.cfg.Cron.FullCrawl != nil && s.cfg.Cron.FullCrawl.Enabled {
		d, _ := config.ParseCronDuration(s.cfg.Cron.FullCrawl.Interval)
		for _, p := range s.cfg.Cron.FullCrawl.Projects {
			add("yaml-full:"+p, p, d, true)
		}
	}
	if s.cfg.Cron.Incremental != nil && s.cfg.Cron.Incremental.Enabled {
		d, _ := config.ParseCronDuration(s.cfg.Cron.Incremental.Interval)
		for _, p := range s.cfg.Cron.Incremental.Projects {
			add("yaml-incr:"+p, p, d, false)
		}
	}

	for _, ov := range s.man.List() {
		url := ov.ProjectURL
		if url == "" {
			url = ov.ProjectKey
		}
		if ov.FullCrawl {
			d, _ := config.ParseCronDuration(ov.FullCrawlInterval)
			if d == 0 {
				d = 24 * time.Hour
			}
			add("ov-full:"+ov.ProjectKey, url, d, true)
		}
		if ov.IncrCrawl {
			d, _ := config.ParseCronDuration(ov.IncrCrawlInterval)
			if d == 0 {
				d = 2 * time.Hour
			}
			add("ov-incr:"+ov.ProjectKey, url, d, false)
		}
	}
}

func (s *Scheduler) loop(ctx context.Context) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	now := time.Now()
	type due struct {
		id    string
		url   string
		force bool
		iv    time.Duration
	}
	var run []due
	s.mu.Lock()
	for _, e := range s.entries {
		if e.disabled || e.next.After(now) {
			continue
		}
		run = append(run, due{id: e.id, url: e.url, force: e.force, iv: e.interval})
		e.last = now
		e.next = now.Add(e.interval)
	}
	s.mu.Unlock()

	for _, d := range run {
		job, err := s.jobs.CreateJob(d.url, d.force)
		if err != nil {
			s.log.Errorw("cron create job failed", "id", d.id, "error", err)
			continue
		}
		go func(jobID string) {
			_ = s.jobs.RunJob(context.Background(), jobID)
		}(job.ID)
		s.log.Infow("cron started crawl", "entry", d.id, "job_id", job.ID, "force", d.force)
		_ = ctx
	}
}
