package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, tx *gorm.DB, token *models.RefreshToken) error
	Lock(ctx context.Context, tx *gorm.DB, id string) (*models.RefreshToken, error)
	MarkUsed(ctx context.Context, tx *gorm.DB, id string) error
	RevokeFamily(ctx context.Context, tx *gorm.DB, familyID string) (int64, error)
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) conn(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx)
}

func (r *refreshTokenRepository) Create(ctx context.Context, tx *gorm.DB, token *models.RefreshToken) error {
	if err := r.conn(ctx, tx).Create(token).Error; err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) Lock(ctx context.Context, tx *gorm.DB, id string) (*models.RefreshToken, error) {
	return firstOrNil[models.RefreshToken](r.conn(ctx, tx).Clauses(forUpdate()).Where("id = ?", id), "lock refresh token")
}

func (r *refreshTokenRepository) MarkUsed(ctx context.Context, tx *gorm.DB, id string) error {
	if err := r.conn(ctx, tx).Exec(`UPDATE refresh_tokens SET used_at = NOW() WHERE id = ? AND used_at IS NULL`, id).Error; err != nil {
		return fmt.Errorf("mark refresh token used: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) RevokeFamily(ctx context.Context, tx *gorm.DB, familyID string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE refresh_tokens SET revoked_at = NOW() WHERE family_id = ? AND revoked_at IS NULL`, familyID)
	if res.Error != nil {
		return 0, fmt.Errorf("revoke refresh token family: %w", res.Error)
	}
	return res.RowsAffected, nil
}
