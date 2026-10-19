package scraper

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/infomosquitos/ticketmosquito/internal/config"
	"github.com/infomosquitos/ticketmosquito/internal/session"
	"github.com/infomosquitos/ticketmosquito/internal/storage"
	"github.com/infomosquitos/ticketmosquito/internal/store"
	"github.com/infomosquitos/ticketmosquito/pkg/logging"
)

// JobStatus is the lifecycle state of an async crawl job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// CrawlJob is an in-memory async project crawl.
type CrawlJob struct {
	ID          string     `json:"id"`
	ProjectURL  string     `json:"project_url"`
	Force       bool       `json:"force"`
	Status      JobStatus  `json:"status"`
	Progress    int        `json:"progress"`
	TotalIssues int        `json:"total_issues"`
	Completed   int        `json:"completed"`
	Failed      int        `json:"failed"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// JobSnapshot is a list+counts view for GET /api/crawl.
type JobSnapshot struct {
	Jobs      []CrawlJob `json:"jobs"`
	Total     int        `json:"total"`
	Running   int        `json:"running"`
	Completed int        `json:"completed"`
	Failed    int        `json:"failed"`
	Pending   int        `json:"pending"`
}

// CrawlJobManager tracks async CrawlProject runs.
type CrawlJobManager struct {
	jobs    map[string]*CrawlJob
	cancels map[string]context.CancelFunc
	mu      sync.RWMutex
	log     logging.Sugar
	cfg     *config.Config
	db      store.Store
	sess    *session.Store
	writer  *storage.Writer
}

// NewJobManager creates a crawl job manager.
func NewJobManager(cfg *config.Config, db store.Store, sess *session.Store, writer *storage.Writer, log logging.Sugar) *CrawlJobManager {
	return &CrawlJobManager{
		jobs:    make(map[string]*CrawlJob),
		cancels: make(map[string]context.CancelFunc),
		log:     log,
		cfg:     cfg,
		db:      db,
		sess:    sess,
		writer:  writer,
	}
}

// CreateJob registers a pending job (does not start it).
func (m *CrawlJobManager) CreateJob(projectURL string, force bool) (*CrawlJob, error) {
	job := &CrawlJob{
		ID:         uuid.New().String(),
		ProjectURL: projectURL,
		Force:      force,
		Status:     JobStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()
	m.log.Infow("crawl job created", "job_id", job.ID, "project_url", projectURL, "force", force)
	return job, nil
}

// GetJob returns a job by id.
func (m *CrawlJobManager) GetJob(jobID string) (*CrawlJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	cp := *job
	return &cp, nil
}

// ListJobs returns a snapshot of all jobs.
func (m *CrawlJobManager) ListJobs() *JobSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap := &JobSnapshot{Jobs: make([]CrawlJob, 0, len(m.jobs))}
	for _, job := range m.jobs {
		snap.Jobs = append(snap.Jobs, *job)
		switch job.Status {
		case JobStatusRunning:
			snap.Running++
		case JobStatusCompleted:
			snap.Completed++
		case JobStatusFailed:
			snap.Failed++
		case JobStatusPending:
			snap.Pending++
		}
	}
	snap.Total = len(m.jobs)
	return snap
}

// CancelJob cancels a pending or running job.
func (m *CrawlJobManager) CancelJob(jobID string) error {
	m.mu.Lock()
	job, ok := m.jobs[jobID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("job not found: %s", jobID)
	}
	if job.Status != JobStatusPending && job.Status != JobStatusRunning {
		m.mu.Unlock()
		return fmt.Errorf("cannot cancel job with status: %s", job.Status)
	}
	job.Status = JobStatusCancelled
	job.UpdatedAt = time.Now()
	now := time.Now()
	job.CompletedAt = &now
	cancel := m.cancels[jobID]
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	m.log.Infow("crawl job cancelled", "job_id", jobID)
	return nil
}

// RunJob starts a pending crawl.
func (m *CrawlJobManager) RunJob(ctx context.Context, jobID string) error {
	m.mu.Lock()
	job, ok := m.jobs[jobID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("job not found: %s", jobID)
	}
	if job.Status == JobStatusCancelled {
		m.mu.Unlock()
		return context.Canceled
	}
	if job.Status != JobStatusPending {
		m.mu.Unlock()
		return fmt.Errorf("job is not pending: %s", job.Status)
	}
	jobCtx, cancel := context.WithCancel(ctx)
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	job.UpdatedAt = now
	m.cancels[jobID] = cancel
	m.mu.Unlock()

	defer func() {
		cancel()
		m.mu.Lock()
		delete(m.cancels, jobID)
		m.mu.Unlock()
	}()

	scr := New(m.db, m.sess, m.cfg.Session.EncryptionKey, m.writer, m.log)
	scr.SetForce(job.Force)
	scr.SetProgress(func(completed, total, failed int) {
		m.mu.Lock()
		defer m.mu.Unlock()
		job.Completed = completed
		job.TotalIssues = total
		job.Failed = failed
		if total > 0 {
			job.Progress = int(float64(completed) / float64(total) * 100)
		}
		job.UpdatedAt = time.Now()
	})

	stats, err := scr.CrawlProject(jobCtx, job.ProjectURL)
	m.finishJob(job, err, stats)
	return err
}

func (m *CrawlJobManager) finishJob(job *CrawlJob, err error, stats *Stats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job.UpdatedAt = time.Now()
	if stats != nil {
		job.TotalIssues = stats.Discovered
		job.Completed = stats.Fetched + stats.Skipped
		job.Failed = stats.Failed
		if job.TotalIssues > 0 {
			job.Progress = int(float64(job.Completed) / float64(job.TotalIssues) * 100)
		} else {
			job.Progress = 100
		}
	}
	if job.Status == JobStatusCancelled {
		return
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		job.Status = JobStatusCancelled
		now := time.Now()
		job.CompletedAt = &now
		job.Error = ""
		return
	}
	if err != nil {
		job.Status = JobStatusFailed
		job.Error = err.Error()
		now := time.Now()
		job.CompletedAt = &now
		m.log.Errorw("crawl job failed", "job_id", job.ID, "project_url", job.ProjectURL, "error", err)
		return
	}
	job.Status = JobStatusCompleted
	now := time.Now()
	job.CompletedAt = &now
	m.log.Infow("crawl job completed",
		"job_id", job.ID,
		"project_url", job.ProjectURL,
		"discovered", job.TotalIssues,
		"completed", job.Completed,
		"failed", job.Failed)
}

// Cleanup removes finished jobs from memory.
func (m *CrawlJobManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, job := range m.jobs {
		switch job.Status {
		case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
			delete(m.jobs, id)
			delete(m.cancels, id)
		}
	}
}
