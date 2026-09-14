// Package batch is a background job framework: job registry, per-run logging
// to batch_jobs, chunked processing with retry/skip, cron scheduling, and
// manual triggers. Jobs run in-process (single instance, no queue needed for
// scheduling); RabbitMQ is reserved for tasks that must leave the HTTP lane.
package batch

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
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

const (
	// runTimeout bounds one run of any job.
	runTimeout = 30 * time.Minute
	// bookkeepingTimeout bounds writing a run's progress or final row, which
	// must happen even when the run's own context is already canceled.
	bookkeepingTimeout = 5 * time.Second
)

// Manager registers, schedules and runs jobs, logging runs to batch_jobs.
type Manager struct {
	db   *gorm.DB
	repo repository.BatchJobRepository
	jobs map[string]*Job
	cron *cronScheduler

	// ctx is canceled by Stop: every running job sees it and winds down.
	ctx    context.Context
	cancel context.CancelFunc
	// manual tracks runs started by Trigger.
	manual sync.WaitGroup
}

func NewManager(db *gorm.DB, repo repository.BatchJobRepository) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{db: db, repo: repo, jobs: make(map[string]*Job), ctx: ctx, cancel: cancel}
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

// Start closes runs a previous process left RUNNING, then schedules cron jobs
// that declare a schedule.
func (m *Manager) Start(ctx context.Context) error {
	stopped, err := m.repo.StopOrphans(ctx)
	if err != nil {
		return err
	}
	if stopped > 0 {
		logger.Warn("batch runs interrupted by the last shutdown marked stopped", logger.Int("runs", int(stopped)))
	}

	m.cron = newCron()
	for _, job := range m.jobs {
		if job.Schedule == "" {
			continue
		}
		name := job.Name
		if _, err := m.cron.addFunc(job.Schedule, func() { m.runScheduled(name) }); err != nil {
			logger.Error("invalid cron schedule",
				logger.String("job", name), logger.String("schedule", job.Schedule), logger.Err(err))
			continue
		}
		logger.Info("batch job scheduled",
			logger.String("job", name), logger.String("schedule", job.Schedule))
	}
	m.cron.start()
	return nil
}

// runScheduled is one cron tick. A tick that finds the previous run still going
// is logged as a skipped run instead of an error.
func (m *Manager) runScheduled(name string) {
	ctx, cancel := context.WithTimeout(m.ctx, runTimeout)
	defer cancel()
	err := m.Run(ctx, name, models.TriggerCron)
	switch {
	case err == nil:
	case errors.Is(err, apperrors.ErrJobRunning):
		bctx, bcancel := context.WithTimeout(context.Background(), bookkeepingTimeout)
		defer bcancel()
		if rerr := m.repo.RecordSkipped(bctx, name, models.TriggerCron, "previous run still running"); rerr != nil {
			logger.Warn("skipped batch run not recorded", logger.String("job", name), logger.Err(rerr))
		}
	default:
		logger.Error("scheduled batch job failed", logger.String("job", name), logger.Err(err))
	}
}

// Stop halts scheduling, cancels the context of running jobs (they are chunked
// and safe to rerun) and waits, bounded by ctx, for them to finish so shutdown
// never closes the database under a job (E-B1, NFR-DEP-03).
func (m *Manager) Stop(ctx context.Context) {
	m.cancel()
	done := make(chan struct{})
	go func() {
		if m.cron != nil {
			<-m.cron.stop().Done()
		}
		m.manual.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		logger.Warn("batch jobs still running at shutdown")
	}
}

// Run executes one job now and waits for it: opens a RUNNING row, calls the
// job, closes the row whatever happens (error, panic, cancellation). A second
// call while the job is running returns ErrJobRunning.
func (m *Manager) Run(ctx context.Context, name, triggeredBy string) error {
	job, run, err := m.begin(ctx, name, triggeredBy)
	if err != nil {
		return err
	}
	return m.execute(ctx, job, run)
}

// Trigger starts a manual run in the background. It returns once the run's
// RUNNING row exists, so an unknown job (404) or a run already going (409) is
// reported to the caller; the result lands in batch_jobs.
func (m *Manager) Trigger(name, triggeredBy string) (string, error) {
	job, run, err := m.begin(m.ctx, name, triggeredBy)
	if err != nil {
		return "", err
	}
	m.manual.Add(1)
	go func() {
		defer m.manual.Done()
		ctx, cancel := context.WithTimeout(m.ctx, runTimeout)
		defer cancel()
		if err := m.execute(ctx, job, run); err != nil {
			logger.Error("manual batch job failed", logger.String("job", name), logger.Err(err))
		}
	}()
	return run.ID, nil
}

func (m *Manager) begin(ctx context.Context, name, triggeredBy string) (*Job, *models.BatchJob, error) {
	job, ok := m.jobs[name]
	if !ok {
		return nil, nil, apperrors.ErrJobNotFound
	}
	run, err := m.repo.Start(ctx, name, triggeredBy)
	if err != nil {
		return nil, nil, err
	}
	return job, run, nil
}

// execute runs the job and always closes its row: success, failed (error or
// panic) or stopped (canceled by shutdown).
func (m *Manager) execute(ctx context.Context, job *Job, run *models.BatchJob) (runErr error) {
	bookkeeping := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.WithoutCancel(ctx), bookkeepingTimeout)
	}
	opts := RunOptions{
		DB:          m.db,
		RunID:       run.ID,
		TriggeredBy: run.TriggeredBy,
		Progress: func(processed, skipped int) error {
			pctx, cancel := bookkeeping()
			defer cancel()
			return m.repo.RecordProgress(pctx, run.ID, processed, skipped)
		},
	}

	defer func() {
		status, msg := models.BatchSuccess, ""
		if r := recover(); r != nil {
			runErr = fmt.Errorf("%s panicked: %v", job.Name, r)
			status, msg = models.BatchFailed, runErr.Error()
			logger.Error("batch job panicked", logger.String("job", job.Name),
				logger.String("panic", fmt.Sprint(r)), logger.String("stack", string(debug.Stack())))
		} else if runErr != nil {
			status, msg = models.BatchFailed, runErr.Error()
			if ctx.Err() != nil {
				status = models.BatchStopped
			}
			runErr = fmt.Errorf("%s failed: %w", job.Name, runErr)
		}
		fctx, cancel := bookkeeping()
		defer cancel()
		if err := m.repo.Finish(fctx, run.ID, status, msg); err != nil {
			logger.Error("finish batch job failed", logger.String("run_id", run.ID), logger.Err(err))
		}
	}()
	return job.Run(ctx, opts)
}

// ItemAttempts is how many times a failing item is tried before it is skipped.
const ItemAttempts = 3

// NoRetry marks an item error that trying again right away can not fix (the
// item keeps its own retry schedule): RunInChunks skips the item at once.
func NoRetry(err error) error {
	if err == nil {
		return nil
	}
	return noRetryError{err}
}

type noRetryError struct{ error }

func (e noRetryError) Unwrap() error { return e.error }

// RunInChunks processes items in chunks (~500), persisting progress after each
// chunk. A failing item is retried up to ItemAttempts times, then counted as
// skipped; it never stops the job (E-B4).
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
			var err error
			for attempt := 0; attempt < ItemAttempts; attempt++ {
				if err = each(ctx, item); err == nil {
					break
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
				var noRetry noRetryError
				if errors.As(err, &noRetry) {
					break
				}
			}
			if err != nil {
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
