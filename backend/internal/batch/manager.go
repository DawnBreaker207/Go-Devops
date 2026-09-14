// Package batch is a background job framework: job registry, per-run logging
// to batch_jobs, chunked processing with retry/skip, cron scheduling, and
// manual triggers. Jobs run in-process (single instance, no queue needed for
// scheduling); RabbitMQ is reserved for tasks that must leave the HTTP lane.
package batch

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// Job is a unit of background work: a 6-field cron schedule (seconds included),
// empty schedule = manual trigger only. A struct beats an interface here since
// every job shares the same shape.
type Job struct {
	Name     string
	Schedule string
	Run      func(ctx context.Context, opts RunOptions) error
}

// RunOptions carries run-level info a job reports progress through.
type RunOptions struct {
	DB          *gorm.DB
	RunID       string
	TriggeredBy string
	// Progress reports processed/skipped counts after each chunk (persisted to batch_jobs).
	Progress func(processed, skipped int) error
}

// Manager registers, schedules and runs jobs, logging runs to batch_jobs.
type Manager struct {
	db   *gorm.DB
	repo repository.BatchJobRepository
	jobs map[string]*Job
	cron *cronScheduler
}

func NewManager(db *gorm.DB, repo repository.BatchJobRepository) *Manager {
	return &Manager{db: db, repo: repo, jobs: make(map[string]*Job)}
}

// Register adds a job. Name is the identity; a duplicate silently replaces.
func (m *Manager) Register(job *Job) {
	m.jobs[job.Name] = job
}

// Names lists registered job names for the admin UI.
func (m *Manager) Names() []string {
	names := make([]string, 0, len(m.jobs))
	for name := range m.jobs {
		names = append(names, name)
	}
	return names
}

// Start schedules cron jobs that declare a schedule.
func (m *Manager) Start() {
	m.cron = newCron()
	for _, job := range m.jobs {
		if job.Schedule == "" {
			continue
		}
		name := job.Name
		if _, err := m.cron.addFunc(job.Schedule, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			if err := m.Run(ctx, name, models.TriggerCron); err != nil {
				logger.Error("scheduled batch job failed",
					logger.String("job", name), logger.Err(err))
			}
		}); err != nil {
			logger.Error("invalid cron schedule",
				logger.String("job", name), logger.String("schedule", job.Schedule), logger.Err(err))
			continue
		}
		logger.Info("batch job scheduled",
			logger.String("job", name), logger.String("schedule", job.Schedule))
	}
	m.cron.start()
}

// Stop halts scheduling, letting already-running jobs finish.
func (m *Manager) Stop() {
	if m.cron != nil {
		m.cron.stop()
	}
}

// Run executes one job now: opens a RUNNING row, calls the job, closes the row.
// A second call while the job is running returns ErrJobRunning.
func (m *Manager) Run(ctx context.Context, name, triggeredBy string) error {
	job, ok := m.jobs[name]
	if !ok {
		return apperrors.ErrJobNotFound
	}

	run, err := m.repo.Start(ctx, name, triggeredBy)
	if err != nil {
		return err
	}

	opts := RunOptions{
		DB:          m.db,
		RunID:       run.ID,
		TriggeredBy: triggeredBy,
		Progress: func(processed, skipped int) error {
			return m.repo.RecordProgress(context.Background(), run.ID, processed, skipped)
		},
	}

	runErr := job.Run(ctx, opts)
	status := models.BatchSuccess
	msg := ""
	if runErr != nil {
		status, msg = models.BatchFailed, runErr.Error()
	}
	if err := m.repo.Finish(context.Background(), run.ID, status, msg); err != nil {
		logger.Error("finish batch job failed", logger.String("run_id", run.ID), logger.Err(err))
	}
	if runErr != nil {
		runErr = fmt.Errorf("%s failed: %w", name, runErr)
	}
	return runErr
}

// RunInChunks processes items in chunks (~500), persisting progress after each
// chunk. A failing item is counted as skipped and does not stop the job.
func RunInChunks[T any](ctx context.Context, opts RunOptions, all []T, each func(ctx context.Context, item T) error) error {
	const chunkSize = 500
	processed, skipped := 0, 0
	for start := 0; start < len(all); start += chunkSize {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		end := start + chunkSize
		if end > len(all) {
			end = len(all)
		}
		for _, item := range all[start:end] {
			if err := each(ctx, item); err != nil {
				skipped++
				continue
			}
			processed++
		}
		if opts.Progress != nil {
			_ = opts.Progress(processed, skipped)
		}
	}
	return nil
}