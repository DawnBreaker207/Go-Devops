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

type BatchJobRepository interface {
	Start(ctx context.Context, name, triggeredBy string) (*models.BatchJob, error)
	Finish(ctx context.Context, runID, status, errorMessage string) error
	RecordProgress(ctx context.Context, runID string, processed, skipped int) error
	StopOrphans(ctx context.Context) (int64, error)
	RecordSkipped(ctx context.Context, name, triggeredBy, reason string) error
	List(ctx context.Context, query dto.PageQuery) ([]models.BatchJob, int64, error)
	RecentFailed(ctx context.Context, since time.Time, limit int) ([]models.BatchJob, error)
}

type batchJobRepository struct {
	db *gorm.DB
}

func NewBatchJobRepository(db *gorm.DB) BatchJobRepository {
	return &batchJobRepository{db: db}
}

// A second RUNNING row for the same job violates uq_batch_jobs_one_running and
// returns ErrJobRunning, so a job never runs in parallel with itself.
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

// StopOrphans runs at startup: the server is a single instance, so any RUNNING row is left by a
// dead process and would otherwise block its job forever through the unique RUNNING index.
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

// RecentFailed is for the admin overview's operational alerts, not the full
// run history List already serves.
func (r *batchJobRepository) RecentFailed(ctx context.Context, since time.Time, limit int) ([]models.BatchJob, error) {
	var out []models.BatchJob
	if err := r.db.WithContext(ctx).
		Where("status = ? AND started_at >= ?", models.BatchFailed, since).
		Order("started_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, fmt.Errorf("find recent failed batch jobs: %w", err)
	}
	return out, nil
}
