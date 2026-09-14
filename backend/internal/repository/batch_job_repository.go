package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// BatchJobRepository reads and writes batch_jobs — the audit trail of every
// job run, errors included.
type BatchJobRepository interface {
	Start(ctx context.Context, name, triggeredBy string) (*models.BatchJob, error)
	Finish(ctx context.Context, runID, status, errorMessage string) error
	RecordProgress(ctx context.Context, runID string, processed, skipped int) error
	StopOrphans(ctx context.Context) (int64, error)
	RecordSkipped(ctx context.Context, name, triggeredBy, reason string) error
	List(ctx context.Context, query dto.PageQuery) ([]models.BatchJob, int64, error)
}

type batchJobRepository struct {
	db *gorm.DB
}

func NewBatchJobRepository(db *gorm.DB) BatchJobRepository {
	return &batchJobRepository{db: db}
}

// Start opens a run: inserts a RUNNING row. If another RUNNING row for the
// same job exists (partial unique index uq_batch_jobs_one_running), it fails
// with ErrJobRunning so jobs never run in parallel.
func (r *batchJobRepository) Start(ctx context.Context, name, triggeredBy string) (*models.BatchJob, error) {
	run := &models.BatchJob{
		JobName:     name,
		TriggeredBy: triggeredBy,
		Status:      models.BatchRunning,
		StartedAt:   time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(run).Error; err != nil {
		if apperrors.IsUniqueViolation(err) {
			return nil, apperrors.ErrJobRunning
		}
		return nil, fmt.Errorf("start batch job: %w", err)
	}
	return run, nil
}

// Finish closes the running row: final status, error and finished_at.
func (r *batchJobRepository) Finish(ctx context.Context, runID, status, errorMessage string) error {
	if err := r.db.WithContext(ctx).Model(&models.BatchJob{}).
		Where("id = ? AND status = ?", runID, models.BatchRunning).
		Updates(map[string]any{
			"status":        status,
			"error_message": errorMessage,
			"finished_at":   time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("finish batch job: %w", err)
	}
	return nil
}

// RecordProgress updates processed/skipped counts after every chunk so a crash
// leaves the last good state behind.
func (r *batchJobRepository) RecordProgress(ctx context.Context, runID string, processed, skipped int) error {
	if err := r.db.WithContext(ctx).Model(&models.BatchJob{}).
		Where("id = ?", runID).
		Updates(map[string]any{
			"processed_rows": processed,
			"skipped_rows":   skipped,
		}).Error; err != nil {
		return fmt.Errorf("record batch progress: %w", err)
	}
	return nil
}

// StopOrphans closes RUNNING rows left by a process that died mid-run. The
// server runs as a single instance, so at startup nothing is really running;
// without this the unique RUNNING index would block the job forever (H1).
func (r *batchJobRepository) StopOrphans(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).Model(&models.BatchJob{}).
		Where("status = ?", models.BatchRunning).
		Updates(map[string]any{
			"status":        models.BatchStopped,
			"error_message": "interrupted: process restarted",
			"finished_at":   time.Now(),
		})
	if res.Error != nil {
		return 0, fmt.Errorf("stop orphan batch jobs: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// RecordSkipped logs a scheduled run that did not start because the previous
// one was still running.
func (r *batchJobRepository) RecordSkipped(ctx context.Context, name, triggeredBy, reason string) error {
	now := time.Now()
	run := &models.BatchJob{
		JobName:      name,
		TriggeredBy:  triggeredBy,
		Status:       models.BatchSkipped,
		ErrorMessage: reason,
		StartedAt:    now,
		FinishedAt:   &now,
	}
	if err := r.db.WithContext(ctx).Create(run).Error; err != nil {
		return fmt.Errorf("record skipped batch job: %w", err)
	}
	return nil
}

func (r *batchJobRepository) List(ctx context.Context, query dto.PageQuery) ([]models.BatchJob, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.BatchJob{})

	if search := query.Search; search != "" {
		tx = tx.Where("job_name ILIKE ?", "%"+search+"%")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count batch jobs: %w", err)
	}

	jobs := make([]models.BatchJob, 0, query.PageSize)
	if err := tx.Order("started_at DESC").
		Limit(query.PageSize).
		Offset(query.Offset()).
		Find(&jobs).Error; err != nil {
		return nil, 0, fmt.Errorf("list batch jobs: %w", err)
	}
	return jobs, total, nil
}