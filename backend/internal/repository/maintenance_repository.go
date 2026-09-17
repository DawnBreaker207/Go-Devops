package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type MaintenanceRepository interface {
	DeleteAuditOlderThan(ctx context.Context, days, limit int) (int64, error)
	DeleteDeadRefreshTokens(ctx context.Context, limit int) (int64, error)
	DeleteDeadResetTokens(ctx context.Context, limit int) (int64, error)
	DeleteFinishedRunsOlderThan(ctx context.Context, days, limit int) (int64, error)
}

type maintenanceRepository struct {
	db *gorm.DB
}

func NewMaintenanceRepository(db *gorm.DB) MaintenanceRepository {
	return &maintenanceRepository{db: db}
}

// DeleteAuditOlderThan is the one caller allowed past the append-only trigger
// (trg_audit_logs_immutable, migration 000006): it sets the transaction-local
// flag the trigger checks right before its own DELETE, inside the same
// transaction so the flag can never leak to any other statement.
func (r *maintenanceRepository) DeleteAuditOlderThan(ctx context.Context, days, limit int) (int64, error) {
	var affected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SET LOCAL audit.allow_retention_delete = 'on'`).Error; err != nil {
			return err
		}
		res := tx.Exec(`DELETE FROM audit_logs WHERE id IN (
			SELECT id FROM audit_logs WHERE created_at < NOW() - make_interval(days => CAST(? AS int)) LIMIT ?)`, days, limit)
		if res.Error != nil {
			return res.Error
		}
		affected = res.RowsAffected
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("delete old audit logs: %w", err)
	}
	return affected, nil
}

func (r *maintenanceRepository) DeleteDeadRefreshTokens(ctx context.Context, limit int) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`DELETE FROM refresh_tokens WHERE id IN (
		SELECT id FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '1 day' LIMIT ?)`, limit)
	if res.Error != nil {
		return 0, fmt.Errorf("delete dead refresh tokens: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *maintenanceRepository) DeleteDeadResetTokens(ctx context.Context, limit int) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`DELETE FROM password_reset_tokens WHERE id IN (
		SELECT id FROM password_reset_tokens WHERE expires_at < NOW() - INTERVAL '1 day' LIMIT ?)`, limit)
	if res.Error != nil {
		return 0, fmt.Errorf("delete dead password reset tokens: %w", res.Error)
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
