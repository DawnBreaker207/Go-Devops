package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PasswordResetToken stores only the SHA-256 of an emailed reset token: the hex
// itself is never persisted, so a DB leak can not redeem it.
type PasswordResetToken struct {
	ID        string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    string     `gorm:"type:uuid;not null" json:"user_id"`
	TokenHash string     `gorm:"type:char(64);not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (PasswordResetToken) TableName() string { return "password_reset_tokens" }

func (t *PasswordResetToken) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}
