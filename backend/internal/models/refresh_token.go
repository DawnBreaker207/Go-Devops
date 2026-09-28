package models

import "time"

// One FamilyID per login: a replay revokes every rotated token.
type RefreshToken struct {
	ID         string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     string     `gorm:"type:uuid;not null" json:"user_id"`
	FamilyID   string     `gorm:"type:uuid;not null" json:"family_id"`
	DeviceID   string     `gorm:"type:varchar(255);not null;default:''" json:"device_id"`
	UserAgent  string     `gorm:"type:varchar(512)" json:"user_agent,omitempty"`
	ExpiresAt  time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
