package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// PasswordResetTokenRepository stores only hashed reset tokens. Each user keeps
// at most one live token: issuing a new one invalidates the previous.
type PasswordResetTokenRepository interface {
	Create(ctx context.Context, db *gorm.DB, token *models.PasswordResetToken) error
	InvalidateUser(ctx context.Context, tx *gorm.DB, userID string) error
	// LockValid locks one unused, unexpired token by its sha256 so concurrent
	// redeems can not both win; nil means it is missing, used, or expired.
	LockValid(ctx context.Context, tx *gorm.DB, hash string) (*models.PasswordResetToken, error)
	MarkUsed(ctx context.Context, tx *gorm.DB, id string) error
}

type passwordResetTokenRepository struct {
	db *gorm.DB
}

func NewPasswordResetTokenRepository(db *gorm.DB) PasswordResetTokenRepository {
	return &passwordResetTokenRepository{db: db}
}

func (r *passwordResetTokenRepository) Create(ctx context.Context, db *gorm.DB, token *models.PasswordResetToken) error {
	if err := db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

func (r *passwordResetTokenRepository) InvalidateUser(ctx context.Context, tx *gorm.DB, userID string) error {
	if err := tx.WithContext(ctx).Exec(`UPDATE password_reset_tokens SET used_at = NOW()
		WHERE user_id = ? AND used_at IS NULL`, userID).Error; err != nil {
		return fmt.Errorf("invalidate password reset tokens: %w", err)
	}
	return nil
}

func (r *passwordResetTokenRepository) LockValid(ctx context.Context, tx *gorm.DB, hash string) (*models.PasswordResetToken, error) {
	var token models.PasswordResetToken
	err := tx.WithContext(ctx).Clauses(forUpdate()).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > NOW()", hash).
		First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lock password reset token: %w", err)
	}
	return &token, nil
}

func (r *passwordResetTokenRepository) MarkUsed(ctx context.Context, tx *gorm.DB, id string) error {
	if err := tx.WithContext(ctx).Exec(`UPDATE password_reset_tokens SET used_at = NOW()
		WHERE id = ? AND used_at IS NULL`, id).Error; err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	return nil
}
