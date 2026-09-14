package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type MaintenanceRepository interface {
	DeleteAuditOlderThan(ctx context.Context, days, limit int) (int64, error)
	DeleteDeadRefreshTokens(ctx context.Context, limit int) (int64, error)
	DeleteFinishedRunsOlderThan(ctx context.Context, days, limit int) (int64, error)
}

type maintenanceRepository struct {
	db *gorm.DB
}

func NewMaintenanceRepository(db *gorm.DB) MaintenanceRepository {
	return &maintenanceRepository{db: db}
}

func (r *maintenanceRepository) DeleteAuditOlderThan(ctx context.Context, days, limit int) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`DELETE FROM audit_logs WHERE id IN (
		SELECT id FROM audit_logs WHERE created_at < NOW() - make_interval(days => CAST(? AS int)) LIMIT ?)`, days, limit)
	if res.Error != nil {
		return 0, fmt.Errorf("delete old audit logs: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *maintenanceRepository) DeleteDeadRefreshTokens(ctx context.Context, limit int) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`DELETE FROM refresh_tokens WHERE id IN (
		SELECT id FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '1 day' LIMIT ?)`, limit)
	if res.Error != nil {
		return 0, fmt.Errorf("delete dead refresh tokens: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *maintenanceRepository) DeleteFinishedRunsOlderThan(ctx context.Context, days, limit int) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`DELETE FROM batch_jobs WHERE id IN (
		SELECT id FROM batch_jobs WHERE status <> 'running'
		AND started_at < NOW() - make_interval(days => CAST(? AS int)) LIMIT ?)`, days, limit)
	if res.Error != nil {
		return 0, fmt.Errorf("delete old batch runs: %w", res.Error)
	}
	return res.RowsAffected, nil
}
